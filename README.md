# k8s-namespace-cleaner

Automated cleanup of stale Kubernetes namespaces based on TTL annotations. Keeps dev and preview clusters tidy without manual intervention.

## Features

- Deletes namespaces past an RFC3339 expiry (`janitor/ttl` annotation)
- Deletes namespaces past a duration from creation (`iqstudio.dev/ttl` annotation)
- `--dry-run` mode to preview deletions
- Works with kubeconfig or in-cluster credentials
- Skips system namespaces (`kube-system`, `default`, etc.)

## Installation

```bash
go install github.com/iftekarqureshi/k8s-namespace-cleaner/cmd/namespace-cleaner@latest
```

Or build locally:

```bash
go build -o namespace-cleaner ./cmd/namespace-cleaner
```

## Usage

```bash
# Preview what would be deleted
namespace-cleaner --dry-run

# Delete expired namespaces
namespace-cleaner
```

## Annotations

**Absolute expiry (RFC3339):**

```yaml
metadata:
  annotations:
    janitor/ttl: "2026-06-30T23:59:59Z"
```

**Duration from namespace creation:**

```yaml
metadata:
  annotations:
    iqstudio.dev/ttl: "24h"
```

Supported duration units: `ns`, `us`, `ms`, `s`, `m`, `h` (Go `time.ParseDuration`).

## RBAC

Minimum permissions for the service account:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: namespace-cleaner
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["list", "delete"]
```

## CronJob example

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: namespace-cleaner
  namespace: kube-system
spec:
  schedule: "0 */6 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: namespace-cleaner
          containers:
            - name: cleaner
              image: ghcr.io/iftekarqureshi/k8s-namespace-cleaner:latest
              args: ["--dry-run"]
          restartPolicy: OnFailure
```

## Development

```bash
go test ./...
go build -o namespace-cleaner ./cmd/namespace-cleaner
```

## License

MIT — see [LICENSE](LICENSE).
