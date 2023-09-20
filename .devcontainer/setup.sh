print_in_color() {
  echo -e "\033[95m$1\033[0m"
}

print_in_color "start minikube and enable ingress"
minikube start
minikube addons enable ingress
echo $(minikube ip) minikube | sudo tee -a /etc/hosts

print_in_color "add kubectl alias"
echo alias k=kubectl >> ~/.bash_aliases

print_in_color "point the local Docker daemon to the minikube internal Docker registry"
eval $(minikube -p minikube docker-env)

print_in_color "install CRDs into the cluster"
make install

print_in_color "install Apicurio Registry"
helm upgrade --install apicurio ./apicurio/ --create-namespace --namespace apicurio
kubectl config set-context --current --namespace=apicurio
