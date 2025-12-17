# kubectl multi-context

A kubectl plugin that runs commands against every context in your kubeconfig file in parallel.

## Features

- Run kubectl commands against all contexts simultaneously
- Parallel execution with batching (10 contexts at a time)
- Support for `version` and `get` subcommands
- Flexible output formatting:
  - Default: Adds a CONTEXT column to table output
  - JSON/YAML: Concatenates items with `.metadata.context` field

## Installation

```bash
go build .
```

## Usage

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

- Go 1.21 or later
- kubectl installed and configured
- Valid kubeconfig file (default: `~/.kube/config` or `$KUBECONFIG`)

