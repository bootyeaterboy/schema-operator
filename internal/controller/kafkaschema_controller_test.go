package controller

import (
	"context"
	// "reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// batchv1 "k8s.io/api/batch/v1"
	// v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kafkav1alpha1 "schema-operator/api/v1alpha1"
)

var _ = Describe("KafkaSchemacontroller", func() {

	const (
		KafkaSchemaName      = "test-kafkaschema"
		KafkaSchemaNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When adding KafkaSchema", func() {
		It("Should work", func() {
			By("By creating a new KafkaSchema")
			ctx := context.Background()
			schema := &kafkav1alpha1.KafkaSchema{
				ObjectMeta: metav1.ObjectMeta{
					Name:      KafkaSchemaName,
					Namespace: KafkaSchemaNamespace,
				},
				Spec: kafkav1alpha1.KafkaSchemaSpec{
					Schema: "protobuf here",
					References: []kafkav1alpha1.Reference{
						{Subject: "import.proto", Version: 1},
					},
				},
			}
			Expect(k8sClient.Create(ctx, schema)).Should(Succeed())

			kafkaSchemaLookupKey := types.NamespacedName{Name: KafkaSchemaName, Namespace: KafkaSchemaNamespace}
			createdKafkaSchema := &kafkav1alpha1.KafkaSchema{}

			// We'll need to retry getting this newly created KafkaSchema, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, kafkaSchemaLookupKey, createdKafkaSchema)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdKafkaSchema.Spec.Schema).Should(Equal("protobuf here"))
		})
	})
})
