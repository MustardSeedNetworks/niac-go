## Kubernetes Deployment

1. Build and push the NIAC image (or reuse the published image):

```bash
docker build -t niac:latest .
kind load docker-image niac:latest # if using kind
```

2. Apply the manifests:

```bash
kubectl apply -f deploy/kubernetes/niac-deployment.yaml
```

3. Port-forward the Web UI:

```bash
kubectl port-forward svc/niac 8080:8080
```

4. Visit `http://localhost:8080` and enter the API token defined in the manifest (`changeme` by default).
