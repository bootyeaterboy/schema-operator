# Deploy Apicurio schema registry

## Install helm chart
```sh
helm upgrade --install apicurio ./apicurio/ --create-namespace --namespace apicurio
```

## Set namespace
```sh
kubectl config set-context --current --namespace=apicurio
```
