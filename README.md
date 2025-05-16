# Igneos.Cloud K3s installer (beta)

The project is written in **Go (Golang)** and provides a **modular, interactive CLI tool** designed to **automate the installation, configuration, and management of lightweight Kubernetes clusters using K3s**. Its main objective is to significantly reduce the manual overhead of setting up a distributed Kubernetes environment while ensuring consistency, repeatability, and operational simplicity.

This tool enables users to **provision a complete K3s cluster** consisting of a **single master node and multiple worker nodes** across remote systems. It establishes **SSH connections** to the target hosts (either physical servers or virtual machines), where it executes installation routines, applies configurations, and starts necessary services. Authentication is handled via user/password or SSH key pairs, as defined in a declarative JSON-based configuration file.

The core functionality covers the entire cluster lifecycle:

* **Master node initialization**: Installs K3s on the designated master host, configures secure networking interfaces, and generates the join token for worker registration.
* **Worker node integration**: Installs K3s on each worker, configures them using the master’s join token, and seamlessly adds them to the cluster.
* **Cluster-wide customization**: Includes support for mounting NFS volumes, setting up private container registries, deploying Ingress resources, and automating TLS certificate issuance via cert-manager.

All operations are orchestrated based on a single declarative **`config.json`** file, which defines the full topology and behavior of the cluster, including IP addresses, SSH access credentials, cluster metadata (e.g., domain names, ACME email), and optional services (such as Docker Registry, NFS Provisioner).

```json
{
  "masters": [
    {
      "ip": "",
      "ssh_user": "",
      "ssh_pass": ""
    }
  ],
  "workers": [
    {
      "ip": "",
      "ssh_user": "",
      "ssh_pass": ""
    }
  ],
  "docker_registry":{
    "url": "",
    "pvc_storagy_capacity":"10Gi",
    "pass": "123456",
    "user": "registry"
  },
  "k3s_token_file": "master-node-token",
  "nfs": {
    "nfs_server": "",
    "nfs_user": "",
    "nfs_pass": "",
    "server": "10.0.0.10",
    "export": "/mnt/k3s-nfs-localstorage",
    "capacity": "100Gi"
  },
  "email": "",
  "domain": "",
  "cluster_issuer_name": "letsencrypt-prod"
}
```



Thanks to this configuration-driven approach, the K3s installer is suitable for **developers, DevOps engineers, and platform teams** who require a fast, repeatable way to stand up Kubernetes clusters—whether for local development, internal testing, or hybrid infrastructure scenarios.



## Docker Registry
Guide: Using Kubernetes imagePullSecrets with Your Private Docker Registry

### Prerequisites
- kubectl configured to point to the desired cluster/context
- A reachable private registry, e.g. data.docker-registry.igneos.cloud
- Valid credentials (username / password or robot token)

### 1) (Optional) Create a dedicated namespace

Namespaces help you separate environments or teams.

```bash
kubectl create namespace my-app
```

### 2) Export registry credentials as environment variables

```bash
export REGISTRY_URL="data.docker-registry.igneos.cloud"
export DOCKER_USER="registry"          # ideally a read-only user
export DOCKER_PASS="123456"            # do *NOT* hard-code this in scripts
export NAMESPACE="my-app"
```

### 3) Create the docker-registry secret

```bash
kubectl create secret docker-registry my-private-docker-registry \
  --docker-server="https://${REGISTRY_URL}" \
  --docker-username="${DOCKER_USER}" \
  --docker-password="${DOCKER_PASS}" \
  --namespace="${NAMESPACE}"
```

Verify creation

```bash
kubectl get secret my-private-docker-registry -n "${NAMESPACE}" -o yaml
```
### 4) (GitOps alternative) Define the secret as YAML
If you manage manifests in Git, encode the auth JSON with base64.

```bash
kubectl create secret [...] --dry-run=client -o yaml > registry-secret.yaml
```

Example (truncated for brevity):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-private-docker-registry
  namespace: my-app
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-json>
```

### 5) Attach the secret to the default ServiceAccount (optional)
This makes every Pod in the namespace inherit the secret automatically.

```bash
kubectl patch serviceaccount default \
  -n "${NAMESPACE}" \
  -p '{"imagePullSecrets":[{"name":"my-private-docker-registry"}]}'
```

### 6) Reference the secret in a Deployment (explicit variant)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: used
  namespace: my-app
  labels:
    app: used
    author: andrej.schefer
  annotations:
    author: Andrej Schefer <andrej.schefer@igneos.cloud>
spec:
  replicas: 1
  selector:
    matchLabels:
      app: used
  template:
    metadata:
      labels:
        app: used
      annotations:
        author: Andrej Schefer <andrej.schefer@igneos.cloud>
    spec:
      # Explicitly tell the Pod which secret to use
      imagePullSecrets:
        - name: my-private-docker-registry # <-- set name of secret
      containers:
        - name: used
          # Always include the registry hostname
          image: data.docker-registry.igneos.cloud/schefer/used:latest
          ports:
            - containerPort: 8080

```
### 7) Validate the Deployment
```bash
kubectl rollout status deploy/used -n "${NAMESPACE}"
kubectl logs -l app=used -n "${NAMESPACE}" --tail=50
```

### 8) Rotating credentials
1. Delete or patch the secret with new auth data
2. Trigger a rolling restart so Pods pick up the update

```bash
kubectl delete secret my-private-docker-registry -n "${NAMESPACE}"
```

...then repeat step 3 with the new password/token.

```bash
kubectl rollout restart deployment/used -n "${NAMESPACE}"
```

### 9) Troubleshooting checklist
- "ErrImagePull"/"ImagePullBackOff": check .dockerconfigjson and registry URL
- Incorrect namespace: the secret must exist in the same namespace as the Pod
- Expired token: recreate the secret with fresh credentials
- DNS / network issues: ensure the cluster can resolve and reach the registry
- Self-signed certs: either add the CA to every node or use an internal CA
