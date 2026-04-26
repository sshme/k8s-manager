# Example service

Example service for testing plugins.

## Deploy

```bash
minikube start --cpus=4 --memory=8192 --disk-size=20g
# Minikube docker registry
eval $(minikube docker-env) # eval $(minikube docker-env -u)
docker build -t example-service:latest .
kubectl apply -f deployment.yml
kubectl apply -f service.yml
```