# kubectl-podview

A kubectl plugin that provides enhanced pod status viewing with resource configuration analysis and ECI (Elastic Container Instance) detection.

## Features

- **Pod Status Overview**: Quickly see the health status of all pods
- **All Namespaces Support**: Query pods across the entire cluster with `-A`
- **ECI Pod Detection**: Identify pods running on Alibaba Cloud ECI (Virtual Kubelet)
- **Running Time Tracking**: Shows actual container running time (not just pod age)
- **Issue Highlighting**: Automatically highlights pods with errors, warnings, or pending status
- **Resource Config Check**: Detect missing resource requests/limits and health probes
- **Restart Tracking**: Shows restart counts and last termination reasons
- **Smart Recommendations**: Provides actionable suggestions based on detected issues

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/fishpie/kubectl-podview.git
cd kubectl-podview

# Build the binary
go build -o kubectl-podview .

# Move to PATH (Linux/macOS)
sudo mv kubectl-podview /usr/local/bin/

# Or for user-only installation
mv kubectl-podview ~/.local/bin/
```

### Verify Installation

```bash
kubectl podview --help
```

## Usage

### Basic Usage

```bash
# View problematic pods in default namespace
kubectl podview

# View pods in a specific namespace
kubectl podview -n test-gatekeeper

# View pods across ALL namespaces
kubectl podview -A

# Show all pods including healthy ones
kubectl podview -n kube-system --all

# Check resource configuration issues
kubectl podview -n production --check-config

# Combine options
kubectl podview -A --all --check-config
```

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--namespace` | `-n` | Kubernetes namespace to inspect (default: "default") |
| `--all-namespaces` | `-A` | Query all namespaces in the cluster |
| `--all` | `-a` | Show all pods, including healthy ones |
| `--check-config` | | Check and highlight resource configuration issues |
| `--kubeconfig` | | Path to kubeconfig file |

### Example Output

**Single Namespace:**

```
🔗 Connecting to cluster...
📦 Fetching pods in namespace 'test-gatekeeper'...
🔍 Analyzing 5 pods...

NAME                                     STATUS     READY    RESTARTS   AGE      RUNNING    ECI   REASON
---------------------------------------------------------------------------------------------------------------
nginx-deployment-7c79c4bf97-abc12        ✓ Healthy  1/1      0          2d5h     2d5h       -     
app-backend-6f8b9d4c5-xyz99              ⚠ Warning  0/1      15         1h30m    45m        ECI   CrashLoopBackOff
  └─ Missing resource limits
redis-master-0                           ◷ Pending  0/1      0          5m       -          -     Unschedulable...

📊 Summary
----------------------------------------
Total Pods:     5
Healthy:        3
Pending:        1
Warning:        1
Total Restarts: 15
ECI Pods:       1 (20.0%)
Config Issues:  1

💡 Recommendations
----------------------------------------
  • Container keeps crashing - check application logs and resource limits
  • Check node resources and taints
  • Set resource limits to prevent resource exhaustion
```

**All Namespaces (-A):**

```
🔗 Connecting to cluster...
📦 Fetching pods across all namespaces...
🔍 Analyzing 127 pods...

NAMESPACE            NAME                                STATUS     READY    RESTARTS   AGE      RUNNING    ECI   REASON
----------------------------------------------------------------------------------------------------------------------------------
kube-system          coredns-7ff77c879f-abc12            ✓ Healthy  1/1      0          15d      15d        -     
production           api-gateway-5f8b9d4c5-xyz99         ⚠ Warning  0/1      8          3h       1h20m      ECI   CrashLoopBackOff
production           worker-batch-6d4e8f7a2-def45        ✓ Healthy  1/1      0          6h       5h55m      ECI   
staging              nginx-ingress-controller-hjk78      ◷ Pending  0/1      0          10m      -          -     ImagePullBackOff

📊 Summary
----------------------------------------
Total Pods:     127
Healthy:        120
Pending:        3
Warning:        4
Total Restarts: 45
ECI Pods:       23 (18.1%)
```

## ECI Detection

The plugin detects ECI pods through multiple methods:

1. **ECI Instance ID Annotation**: Checks for `k8s.aliyun.com/eci-instance-id`
2. **Node Name**: Detects nodes with `virtual-kubelet` prefix
3. **ECI-related Annotations**: Checks for `k8s.aliyun.com/eci-instance-spec`, `k8s.aliyun.com/eci-use-specs`, etc.

ECI pods are marked with cyan `ECI` label in the output.

## Column Descriptions

| Column | Description |
|--------|-------------|
| NAMESPACE | Pod's namespace (shown with `-A` flag) |
| NAME | Pod name |
| STATUS | Health status: Healthy, Warning, Error, Pending |
| READY | Ready containers / Total containers |
| RESTARTS | Total container restart count |
| AGE | Time since pod creation |
| RUNNING | Actual container running time |
| ECI | `ECI` if running on Elastic Container Instance, `-` otherwise |
| REASON | Issue description if not healthy |

## Project Structure

```
kubectl-podview/
├── main.go                 # Entry point
├── cmd/
│   └── root.go             # CLI command definition (cobra)
├── pkg/
│   ├── client/
│   │   └── client.go       # Kubernetes client wrapper
│   ├── analyzer/
│   │   └── analyzer.go     # Pod analysis logic + ECI detection
│   └── printer/
│       └── printer.go      # Output formatting
├── go.mod
├── go.sum
└── README.md
```

## Development

### Prerequisites

- Go 1.21+
- Access to a Kubernetes cluster (preferably Alibaba Cloud ACK for ECI testing)
- kubectl configured

### Building

```bash
go build -o kubectl-podview .
```

### Testing

```bash
go test ./...
```

### Running Locally

```bash
# Test with all namespaces
go run . -A --all

# Test ECI detection
go run . -n your-eci-namespace --all
```

## License

MIT
