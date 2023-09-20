/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/riferrei/srclient"

	kafkav1alpha1 "schema-operator/api/v1alpha1"
)

const (
	finalizerName  = "schema-operator"
	defaultRequeue = 20 * time.Second
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	SchemaRegistryClient *srclient.SchemaRegistryClient
}

// All schema registry API endpoints use a standard error message format for any requests
// that return an HTTP status indicating an error (any 400 or 500 statuses)
// https://docs.confluent.io/platform/current/schema-registry/develop/api.html#errors
type SchemaRegistryError struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

func (schemaRegistryError *SchemaRegistryError) Error() string {
	return schemaRegistryError.Message
}

//+kubebuilder:rbac:groups=kafka.duss.me,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kafka.duss.me,resources=kafkaschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kafka.duss.me,resources=kafkaschemas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Enter Reconcile", "req", req)

	// Fetch the KafkaSchema object
	schema := &kafkav1alpha1.KafkaSchema{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, schema); err != nil {
		if !k8serr.IsNotFound(err) {
			l.Error(err, "Error getting schema")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	schema = schema.DeepCopy()
	subject := getDefaultIfEmpty(schema.Spec.Subject, schema.Name)

	// Is schema marked for deletion?
	if schema.ObjectMeta.DeletionTimestamp.IsZero() {
		// Register finalizer if it does not exist
		if !controllerutil.ContainsFinalizer(schema, finalizerName) {
			controllerutil.AddFinalizer(schema, finalizerName)
			if err := r.Update(ctx, schema); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Delete schema if finalizer is present
		l.Info("Schema is marked for deletion")
		if controllerutil.ContainsFinalizer(schema, finalizerName) {
			deletedSubject := getDefaultIfEmpty(schema.Status.Subject, subject)
			escapedSubject := url.QueryEscape(deletedSubject)
			if err := r.SchemaRegistryClient.DeleteSubject(escapedSubject, false); err != nil {
				schemaRegistryError, unmarshalErr := unmarshalSchemaRegistryError(err)
				if unmarshalErr == nil && schemaRegistryError.ErrorCode == 40401 {
					l.Info("The schema to be deleted does not exist")
				} else {
					l.Error(err, "Error deleting schema")
					setSchemaRegistryErrorStatus(schema, err)
					return ctrl.Result{RequeueAfter: defaultRequeue}, r.Status().Update(ctx, schema)
				}
			}
			controllerutil.RemoveFinalizer(schema, finalizerName)
			return ctrl.Result{}, r.Update(ctx, schema)
		}
		return ctrl.Result{}, nil
	}

	// Validate schema name
	if schema.Status.Subject != "" && schema.Status.Subject != subject {
		err := fmt.Errorf("schemas cannot be renamed, but schema's spec.subject has changed")
		l.Error(err, "Error during validation")
		setErrorStatus(schema, err, "IllegalArgumentException")
		return ctrl.Result{}, r.Status().Update(ctx, schema)
	}

	// Check if schema references exist and map them to a srclient.Reference slice
	references := []srclient.Reference{}
	for _, schemaReference := range schema.Spec.References {
		referencedSchema, err := r.getSchema(schemaReference.Subject, schemaReference.Version)
		if err != nil {
			l.Info("Schema reference does not exist, trying again in 20 seconds",
				"subject", schemaReference.Subject, "version", schemaReference.Version)
			setPendingStatus(schema, &schemaReference)
			return ctrl.Result{RequeueAfter: defaultRequeue}, r.Status().Update(ctx, schema)
		}
		reference := srclient.Reference{Name: schemaReference.Subject, Subject: schemaReference.Subject, Version: referencedSchema.Version()}
		references = append(references, reference)
	}

	// Check if schema already exists
	if existingSchema, err := r.SchemaRegistryClient.LookupSchema(subject, schema.Spec.Schema, srclient.Protobuf, references...); err == nil {
		if isOutOfDate(schema, existingSchema) {
			setCreatedStatus(schema, existingSchema, subject)
			return ctrl.Result{}, r.Status().Update(ctx, schema)
		}
		return ctrl.Result{}, nil
	}

	// Create schema and set schema ID and version
	if _, err := r.SchemaRegistryClient.CreateSchema(subject, schema.Spec.Schema, srclient.Protobuf, references...); err != nil {
		l.Error(err, "Error creating schema")
		setSchemaRegistryErrorStatus(schema, err)
		return ctrl.Result{RequeueAfter: defaultRequeue}, r.Status().Update(ctx, schema)
	} else {
		l.Info("Schema created")
		if createdSchema, err := r.SchemaRegistryClient.LookupSchema(subject, schema.Spec.Schema, srclient.Protobuf, references...); err != nil {
			setSchemaRegistryErrorStatus(schema, err)
			return ctrl.Result{RequeueAfter: defaultRequeue}, r.Status().Update(ctx, schema)
		} else {
			setCreatedStatus(schema, createdSchema, subject)
			return ctrl.Result{}, r.Status().Update(ctx, schema)
		}
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kafkav1alpha1.KafkaSchema{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// Get the latest schema version if no specific version is defined
func (r *KafkaSchemaReconciler) getSchema(subject string, version int) (*srclient.Schema, error) {
	if version == 0 {
		return r.SchemaRegistryClient.GetLatestSchema(subject)
	} else {
		return r.SchemaRegistryClient.GetSchemaByVersion(subject, version)
	}
}

// Checks if the status of a schema is out of date
func isOutOfDate(schema *kafkav1alpha1.KafkaSchema, schemaInRegistry *srclient.Schema) bool {
	return schema.Status.Phase != kafkav1alpha1.CreatedStatusPhase ||
		schema.Status.ID != schemaInRegistry.ID() ||
		schema.Status.Version != schemaInRegistry.Version()
}

// Set created status if schema was successfully created in schema registry
func setCreatedStatus(schema *kafkav1alpha1.KafkaSchema, schemaInRegistry *srclient.Schema, subject string) {
	schema.Status.Phase = kafkav1alpha1.CreatedStatusPhase
	schema.Status.ID = schemaInRegistry.ID()
	schema.Status.Version = schemaInRegistry.Version()
	schema.Status.Subject = subject
	schema.Status.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "HttpStatus200",
	}}
}

// Set pending status if referenced schema is not ready
func setPendingStatus(schema *kafkav1alpha1.KafkaSchema, reference *kafkav1alpha1.Reference) {
	schema.Status.Phase = kafkav1alpha1.PendingStatusPhase
	schema.Status.Conditions = []metav1.Condition{{
		Type:               "ReferenceNotReady",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             string(metav1.StatusReasonNotFound),
		Message:            fmt.Sprintf("Referenced schema '%s' is not available in registry.", reference.Subject),
	}}
}

// Set error status including error message
func setErrorStatus(schema *kafkav1alpha1.KafkaSchema, err error, reason string) {
	schema.Status.Phase = kafkav1alpha1.ErrorStatusPhase
	schema.Status.Conditions = []metav1.Condition{{
		Type:               "NotReady",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	}}
}

// Set error status including error code from schema registry
func setSchemaRegistryErrorStatus(schema *kafkav1alpha1.KafkaSchema, httpError error) {
	schema.Status.Phase = kafkav1alpha1.ErrorStatusPhase
	if schemaRegistryError, err := unmarshalSchemaRegistryError(httpError); err != nil {
		reason := "Unknown"
		setErrorStatus(schema, httpError, reason)
	} else {
		reason := fmt.Sprintf("HttpStatus%d", schemaRegistryError.ErrorCode)
		setErrorStatus(schema, schemaRegistryError, reason)
	}
}

// Convert JSON error from Schema Registry to SchemaRegistryError struct
func unmarshalSchemaRegistryError(httpError error) (*SchemaRegistryError, error) {
	var schemaRegistryError SchemaRegistryError
	data := []byte(httpError.Error())
	if err := json.Unmarshal(data, &schemaRegistryError); err != nil {
		return nil, err
	}
	return &schemaRegistryError, nil
}

// Return default value if first argument is an empty string
func getDefaultIfEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
