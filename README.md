# kubectl x

A kubectl plugin that runs commands against every context in your kubeconfig file in parallel.


## Features

- Run kubectl commands against all contexts simultaneously
- Parallel execution with configurable batching (default: 25 contexts at a time)
- Include/exclude contexts by name pattern
- Support for `version`, `get`, `logs`, `wait`, `top`, `events`, `api-resources`, and `api-versions` subcommands
- Streaming log output with `-f` flag across all contexts
- Watch mode with `-w`/`--watch` flag on `get` and `events` subcommands
- Flexible output formatting:
  - Default: Adds a CONTEXT column to table output
  - JSON/YAML: Concatenates items list, adds `.metadata.context` field. This is useful for manipulation with tools like [`jq`](https://jqlang.org/) or [`yq`](https://github.com/mikefarah/yq).


## Why another project?

This functionality already exists in some other projects, including [kubectl-foreach](https://github.com/ahmetb/kubectl-foreach), [kubectl-cluster-group](https://github.com/u2takey/kubectl-clusters), and [kubectl-allctx](https://github.com/onatm/kubectl-allctx). These are all great projects and we learned a lot from them. However, none of them output valid JSON or YAML and we wanted those options for complex querying capabilities.


## Limitations

### Unstable API

This is a v0.x project - the interface may change at any time and should not be relied upon for programmatic use.

### Read-only operations

The authors of this project do not intend to support write operations (`apply`, `delete`, etc.) via the tool. If you desire to do that, we encourage you to come up with your own tooling, fork the project, or look elsewhere.


## Installation

### Download a release

Download the latest binary for your platform from the [releases page](https://github.com/platformersdev/kubectl-x/releases/latest), then place it in your `$PATH`:

```bash
# Example for macOS (Apple Silicon)
curl -L https://github.com/platformersdev/kubectl-x/releases/latest/download/kubectl-x-darwin-arm64 -o kubectl-x
chmod +x kubectl-x
sudo mv kubectl-x /usr/local/bin/
```

### Install with Go

```bash
go install github.com/platformersdev/kubectl-x@latest
```

### Build from source

```bash
git clone https://github.com/platformersdev/kubectl-x.git
cd kubectl-x
go build .
```

kubectl discovers plugins by looking for executables named `kubectl-<plugin>` on your `$PATH`. As long as `kubectl-x` is in your `$PATH`, you can invoke it as `kubectl x`.


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

### Including Contexts

Filter which contexts to run commands against using the `--include` flag with regex patterns (case-insensitive). You can specify multiple `--include` flags to match contexts that match any of the patterns (OR logic):

```bash
# Match contexts containing "prod"
kubectl x --include prod get pods

# Match contexts starting with "dev"
kubectl x --include "^dev" version

# Match contexts containing "dev" OR "prod"
kubectl x --include dev --include prod get pods

# Match contexts ending with "-prod" or "-staging"
kubectl x --include "-prod$" --include "-staging$" get pods

# Match contexts with "prod" or "production" (using regex alternation)
kubectl x --include "prod(uction)?" get pods

# Combine with batch size
kubectl x --include staging --batch-size 10 get pods
```

### Excluding Contexts

Exclude contexts using the `--exclude` flag with regex patterns (case-insensitive). Multiple `--exclude` flags are OR'd together. When both `--include` and `--exclude` are used, include filters are applied first, then exclude filters remove from that set:

```bash
# Exclude contexts containing "dev"
kubectl x --exclude dev get pods

# Exclude multiple patterns
kubectl x --exclude dev --exclude staging get pods

# Include "prod" contexts but exclude US West
kubectl x --include prod --exclude "us-west" get pods
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

### API Resources Command

Run `kubectl api-resources` against all contexts:

```bash
# List all API resources from all contexts
kubectl x api-resources

# List namespaced resources only
kubectl x api-resources --namespaced=true

# List resources for a specific API group
kubectl x api-resources --api-group=apps
```

### API Versions Command

Run `kubectl api-versions` against all contexts:

```bash
# List all API versions from all contexts
kubectl x api-versions
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
- Go 1.25 or later to build

