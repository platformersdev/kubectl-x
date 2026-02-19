# kubectl x

A kubectl plugin that runs commands against every context in your kubeconfig file in parallel.


## Features

- Run kubectl commands against all contexts simultaneously
- Parallel execution with configurable batching (default: 25 contexts at a time)
- Filter contexts by name pattern
- Support for `version`, `get`, `logs`, `wait`, `top`, and `events` subcommands
- Streaming log output with `-f` flag across all contexts
- Watch mode with `-w`/`--watch` flag on `get` and `events` subcommands
- Flexible output formatting:
  - Default: Adds a CONTEXT column to table output
  - JSON/YAML: Concatenates items list, adds `.metadata.context` field


## Why another project?

This functionality already exists in another project, [kubectl-foreach](https://github.com/ahmetb/kubectl-foreach). That is a great project and I learned a lot from it. However, it does not output valid JSON or YAML and I wanted that option, as well as some other features.


## Limitations

### Unstable API

This is a v0.x project - the interface may change at any time and should not be relied upon for programmatic use.

### Read-only operations

The author of this project does not intend to support write operations via the tool. If you desire to do that, we encourage you to come up with your own tooling or look elsewhere.


## Installation

```bash
go build .
```


## Usage

### Batch Size

Control the number of contexts processed in parallel using the `--batch-size` (or `-b`) flag:

```bash
# Use default batch size of 25
kubectl x get pods

# Use custom batch size
kubectl x --batch-size 10 get pods
kubectl x -b 50 get pods
```

### Filtering Contexts

Filter which contexts to run commands against using the `--filter` flag with regex patterns (case-insensitive). You can specify multiple `--filter` flags to match contexts that match any of the patterns (OR logic):

```bash
# Match contexts containing "prod"
kubectl x --filter prod get pods

# Match contexts starting with "dev"
kubectl x --filter "^dev" version

# Match contexts containing "dev" OR "prod"
kubectl x --filter dev --filter prod get pods

# Match contexts ending with "-prod" or "-staging"
kubectl x --filter "-prod$" --filter "-staging$" get pods

# Match contexts with "prod" or "production" (using regex alternation)
kubectl x --filter "prod(uction)?" get pods

# Combine with batch size
kubectl x --filter staging --batch-size 10 get pods
```

### Version Command

Run `kubectl version` against all contexts:

```bash
kubectl x version
```

### Get Command

Run `kubectl get` against all contexts:

```bash
# Get pods from all contexts
kubectl x get pods

# Get pods with namespace
kubectl x get pods -n default

# Get pods with JSON output
kubectl x get pods -o json

# Get pods with YAML output
kubectl x get pods -o yaml

# Watch pods across all contexts
kubectl x get pods -w

# Watch with namespace
kubectl x get pods -n default --watch

# Watch-only (skip initial listing)
kubectl x get pods --watch-only
```

### Wait Command

Run `kubectl wait` against all contexts:

```bash
# Wait for a pod to be ready across all contexts
kubectl x wait --for=condition=ready pod/my-pod

# Wait with a timeout
kubectl x wait --for=condition=ready pod/my-pod --timeout=60s

# Wait for all pods with a label selector
kubectl x wait --for=condition=ready pods -l app=myapp -n default

# Wait for a deployment rollout
kubectl x wait --for=condition=available deployment/my-deploy
```

### Logs Command

Run `kubectl logs` against all contexts:

```bash
# Get logs for a pod across all contexts
kubectl x logs my-pod

# Get logs with namespace
kubectl x logs my-pod -n default

# Stream logs in real-time across all contexts
kubectl x logs my-pod -f

# Stream logs with additional flags
kubectl x logs my-pod -f --tail=100 -n default
```

### Events Command

Run `kubectl events` against all contexts:

```bash
# Get events from all contexts
kubectl x events

# Get events in a specific namespace
kubectl x events -n default

# Watch events across all contexts in real-time
kubectl x events -w

# Watch events in a specific namespace
kubectl x events -n default --watch

# Watch-only (skip initial listing)
kubectl x events --watch-only
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

