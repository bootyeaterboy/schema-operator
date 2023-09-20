# Schema Operator
Strimzi provides a Kafka Cluster Operator, a Topic Operator, and a User Operator to manage Kafka resources within Kubernetes. Out of the box, however, there is no schema operator. This Kubernetes operator written in Go closes this gap.

## Description
The schema operator reconciles Kafka schemas in the form of custom resources (CRs) with the Apicurio schema registry. Schemas can be created, customized, or deleted as KafkaSchema CRs in the Kubernetes cluster. The operator is then responsible for creating, customizing, or deleting the corresponding schemas in the registry. This allows schemas to be created declaratively à la Infrastructure as Code.

To do its job, the operator receives events from Kubernetes when something changes in the KafkaSchema CRs. An event consists exclusively of a CR name and a namespace. After that, it is up to the operator to figure out if the corresponding schema needs to be created, modified, or deleted in the registry. Or not.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [minikube](https://minikube.sigs.k8s.io/) or [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl kustomize config/samples/ | kubectl apply -f -
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/schema-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/schema-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## How this operator was created
This operator was created using the [Kubebuilder](https://book.kubebuilder.io/) tool.

### Download kubebuilder and install locally
```sh
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && sudo mv kubebuilder /usr/local/bin/
```

### Initialize the operator
```sh
kubebuilder init --domain duss.me --repo schema-operator --plugins=go/v4-alpha
kubebuilder create api --group kafka --version v1alpha1 --kind KafkaSchema
```

## License

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

