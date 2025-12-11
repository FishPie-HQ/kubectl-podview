# kubectl-podview

A kubectl plugin that provides enhanced pod status viewing with resource configuration analysis.

## Features

- **Pod Status Overview**: Quickly see the health status of all pods in a namespace
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
# Should show plugin help
kubectl podview --help
```

## Usage

### Basic Usage

```bash
# View problematic pods in default namespace
kubectl podview

# View pods in a specific namespace
kubectl podview -n test-gatekeeper

# Show all pods including healthy ones
kubectl podview -n kube-system --all

# Check resource configuration issues
kubectl podview -n production --check-config
```

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--namespace` | `-n` | Kubernetes namespace to inspect (default: "default") |
| `--all` | `-a` | Show all pods, including healthy ones |
| `--check-config` | | Check and highlight resource configuration issues |
| `--kubeconfig` | | Path to kubeconfig file |

### Example Output

```
🔗 Connecting to cluster...
📦 Fetching pods in namespace 'test-gatekeeper'...
🔍 Analyzing 5 pods...

NAME                                     STATUS     READY    RESTARTS   AGE        REASON
----------------------------------------------------------------------------------------------------
nginx-deployment-7c79c4bf97-abc12        ✓ Healthy  1/1      0          2d5h       
app-backend-6f8b9d4c5-xyz99              ⚠ Warning  0/1      15         1h30m      CrashLoopBackOff
  └─ Missing resource limits
redis-master-0                           ◷ Pending  0/1      0          5m         Unschedulable: 0/3 nodes...

📊 Summary
----------------------------------------
Total Pods:     5
Healthy:        3
Pending:        1
Warning:        1
Total Restarts: 15
Config Issues:  1

💡 Recommendations
----------------------------------------
  • Container keeps crashing - check application logs and resource limits
  • Check node resources and taints
  • Set resource limits to prevent resource exhaustion
```

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
│   │   └── analyzer.go     # Pod analysis logic
│   └── printer/
│       └── printer.go      # Output formatting
├── go.mod
├── go.sum
└── README.md
```

## How kubectl Plugins Work

kubectl plugins are simply executables that:
1. Are named `kubectl-<plugin-name>`
2. Are available in your `$PATH`

When you run `kubectl podview`, kubectl looks for an executable named `kubectl-podview` in your PATH and runs it.

## Development

### Prerequisites

- Go 1.21+
- Access to a Kubernetes cluster
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
go run . -n default --all
```

## Key Go Concepts Demonstrated

This plugin demonstrates several important Go patterns for K8s development:

1. **client-go usage**: Connecting to clusters, listing resources
2. **Cobra CLI**: Building professional command-line interfaces
3. **Package organization**: Separating concerns into client/analyzer/printer
4. **Error handling**: Propagating errors with context
5. **Struct and interface**: Defining data types and behaviors
6. **Context usage**: Timeout handling for API calls

