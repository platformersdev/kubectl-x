# kubectl multi-context

A kubectl plugin that runs commands against every context in your kubeconfig file in parallel.


## Features

- Run kubectl commands against all contexts simultaneously
- Parallel execution with configurable batching (default: 25 contexts at a time)
- Filter contexts by name pattern
- Support for `version` and `get` subcommands
- Flexible output formatting:
  - Default: Adds a CONTEXT column to table output
  - JSON/YAML: Concatenates items with `.metadata.context` field


## Why another project?

This functionality already exists in another projects, [kubectl-foreach](https://github.com/ahmetb/kubectl-foreach). That is a great project and I learned a lot from it. However, it does not output valid JSON or YAML and I wanted that option, as well as some other features.


## Installation

```bash
go build .
```


## Usage

### Batch Size

Control the number of contexts processed in parallel using the `--batch-size` (or `-b`) flag:

```bash
# Use default batch size of 25
kubectl multi-context get pods

# Use custom batch size
kubectl multi-context --batch-size 10 get pods
kubectl multi-context -b 50 get pods
```

### Filtering Contexts

Filter which contexts to run commands against using the `--filter` flag (case-insensitive substring match). You can specify multiple `--filter` flags to match contexts that contain any of the patterns (OR logic):

```bash
# Only run against contexts containing "prod"
kubectl multi-context --filter prod get pods

# Only run against contexts containing "dev"
kubectl multi-context --filter dev version

# Match contexts containing "dev" OR "prod"
kubectl multi-context --filter dev --filter prod get pods

# Combine with batch size
kubectl multi-context --filter staging --batch-size 10 get pods
```

### Version Command

Run `kubectl version` against all contexts:

```bash
kubectl multi-context version
```

### Get Command

Run `kubectl get` against all contexts:

```bash
# Get pods from all contexts
kubectl multi-context get pods

# Get pods with namespace
kubectl multi-context get pods -n default

# Get pods with JSON output
kubectl multi-context get pods -o json

# Get pods with YAML output
kubectl multi-context get pods -o yaml
```

## Output Formats

### Default Output

By default, the tool adds a `CONTEXT` column to the left of the kubectl output:

```
CONTEXT    NAME                    READY   STATUS    RESTARTS   AGE
ctx1       pod-abc                 1/1     Running   0          5m
ctx2       pod-xyz                 1/1     Running   0          3m
```

### JSON/YAML Output

When using `-o json` or `-o yaml`, the tool concatenates all items from all contexts and adds a `metadata.context` field to each item:

```json
{
  "apiVersion": "v1",
  "kind": "List",
  "items": [
    {
      "metadata": {
        "name": "pod-abc",
        "context": "ctx1",
        ...
      }
    },
    {
      "metadata": {
        "name": "pod-xyz",
        "context": "ctx2",
        ...
      }
    }
  ]
}
```


## Requirements

- kubectl installed and configured
- Valid kubeconfig file (default: `~/.kube/config` or `$KUBECONFIG`)
- Go 1.21 or later to build

