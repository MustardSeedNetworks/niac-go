# Deployment Guide

## Docker Deployment

### Basic Dockerfile

```dockerfile
FROM golang:1.25.4-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o niac ./cmd/niac

FROM alpine:latest
RUN apk add --no-cache libpcap-dev
COPY --from=builder /build/niac /usr/local/bin/
ENTRYPOINT ["niac"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  niac:
    build: .
    network_mode: host  # Required for raw packets
    privileged: true    # Required for packet capture
    volumes:
      - ./config.yaml:/etc/niac/config.yaml:ro
      - ./storage:/var/lib/niac
    command: run /etc/niac/config.yaml --api :8080 --storage /var/lib/niac/storage.db
    environment:
      - NIAC_API_TOKEN=${NIAC_API_TOKEN}
```

## Kubernetes Deployment

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: niac
spec:
  replicas: 1
  selector:
    matchLabels:
      app: niac
  template:
    metadata:
      labels:
        app: niac
    spec:
      hostNetwork: true  # Required for raw sockets
      containers:
      - name: niac
        image: your-registry/niac:2.6.0
        securityContext:
          privileged: true  # Required for packet capture
          capabilities:
            add:
              - NET_RAW
              - NET_ADMIN
        command: ["niac", "run", "/config/config.yaml", "--api", ":8080"]
        env:
        - name: NIAC_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: niac-secret
              key: api-token
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        - name: storage
          mountPath: /var/lib/niac
      volumes:
      - name: config
        configMap:
          name: niac-config
      - name: storage
        persistentVolumeClaim:
          claimName: niac-storage
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: niac-api
spec:
  selector:
    app: niac
  ports:
  - name: api
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

### Best Practices

- Use `hostNetwork: true` for layer 2 simulation
- Set resource limits based on device count
- Use PersistentVolume for storage
- Store API token in Kubernetes Secret
- Enable metrics for monitoring

## Systemd Service

```ini
[Unit]
Description=NIAC-Go Network Simulator
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/niac run /etc/niac/config.yaml --api :8080
Restart=on-failure
RestartSec=5s
Environment="NIAC_API_TOKEN=your-token-here"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable niac
sudo systemctl start niac
```

## Cloud Deployment

### AWS EC2
- Use instance with Enhanced Networking
- t3.medium minimum for 50 devices
- c5n.xlarge for high performance
- Attach Elastic IP for stability

### Azure VM
- Use Accelerated Networking
- Standard_D2s_v3 minimum
- Standard_F4s_v2 for high performance

### GCP Compute Engine
- Enable VirtIO SCSI
- n2-standard-2 minimum
- c2-standard-4 for high performance

## Security Hardening

1. **API Token**: Always use strong tokens
2. **Firewall**: Restrict API access
3. **TLS**: Use reverse proxy with HTTPS
4. **Updates**: Keep NIAC-Go updated
5. **Monitoring**: Enable audit logging
