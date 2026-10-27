# forgeai-instancetype-operator

A Kubernetes operator that turns the GPU nodes of a cluster into `InstanceType`s, and
into [Kueue](https://kueue.sigs.k8s.io) `ResourceFlavor`s, one per GPU class. It is the
instance type controller of [forgeai](https://github.com/wutz/forgeai), split out so that
it can be installed and versioned on its own.

## How it works

Nodes are grouped into GPU classes by the `nvidia.com/gpu.product` label, as set by
NVIDIA GPU feature discovery (`NVIDIA-GeForce-RTX-5090` becomes `rtx-5090`). For each
class the operator:

- creates one cluster-scoped `InstanceType` per size in `instanceTypes.sizes`, up to the
  GPUs per node, named `<class>.x<gpus>` (for example `rtx-5090.x2`);
- sizes CPU and memory per GPU as `(allocatable - reserved) / GPUs` of the smallest node
  in the class, rounded down to whole cores and GiB;
- reports in `status` how many instances fit in total and right now, per node, computed
  from the pods bound to the nodes;
- applies a `ResourceFlavor` named `forgeai-<class>` that selects the class's nodes by
  `nvidia.com/gpu.product`.

The operator recomputes everything when nodes, scheduled pods, the platform config or
its own `InstanceType`s change. Hand edits to an `InstanceType` are reverted, and a deleted
one is recreated. `ResourceFlavor`s are kept when their class disappears, because queues
may still reference them.

```sh
kubectl get fit
```

## Install

The operator is published as a container image and a Helm chart:

```sh
helm install forgeai-instancetype-operator oci://ghcr.io/wutz/charts/forgeai-instancetype-operator \
  --version 0.1.0 \
  --namespace forgeai-system --create-namespace
```

The default image is `ghcr.io/wutz/forgeai-instancetype-operator:<appVersion>`.

### Values

| Value | Default | Description |
|---|---|---|
| `platform.namespace` | `forgeai-system` | Namespace of the forgeai platform ConfigMap. Set as `FORGEAI_NAMESPACE` in the manager. |
| `manager.image.repository` | `ghcr.io/wutz/forgeai-instancetype-operator` | Manager image. |
| `manager.image.tag` | chart `appVersion` | Manager image tag. |
| `rbac.namespaced` | `false` | Use a Role instead of a ClusterRole. The operator works on cluster-scoped objects, so keep the default. |
| `crd.keep` | `true` | Keep the CRD (and its InstanceTypes) when the release is uninstalled. |

See [`charts/chart/values.yaml`](charts/chart/values.yaml) for the rest.

## Platform config

The operator reads its settings from the same ConfigMap as forgeai: `forgeai-system/forgeai-platform`,
key `config.yaml`. It reads only the `instanceTypes` section and ignores the rest:

```yaml
instanceTypes:
  # Kept back on every node for the system before the GPUs are sliced.
  reserved:
    cpu: "2"        # default
    memory: 16Gi    # default
  # GPU counts offered; capped at the GPUs per node. Default: [1, 2, 4, 8].
  sizes: [1, 2, 4, 8]
```

The ConfigMap is re-read at most every 15 seconds, and immediately after it changes.

## Requirements

- Kubernetes with the Kueue `ResourceFlavor` CRD (`kueue.x-k8s.io/v1beta2`) to get flavors.
  Without it the operator still creates the `InstanceType`s and skips the flavors.
- Nodes labelled by NVIDIA GPU feature discovery (`nvidia.com/gpu.product`,
  `nvidia.com/gpu.memory`).

## Contract with forgeai

forgeai selects instance types and ResourceFlavors by the names and labels in
[`pkg/naming`](pkg/naming/naming.go). The operator applies ResourceFlavors with the
server-side field manager `forgeai`, so they keep the owner they had inside forgeai.
Changing a name or a label breaks existing clusters.

## Development

```sh
make test              # unit tests against envtest (Kubernetes 1.37)
make lint              # golangci-lint
make manifests generate  # after changing the API types or RBAC markers
make helm-lint         # lint and render the chart
```

Tests need the envtest binaries, which `make test` downloads into `bin/`.

## Releases

Pushing a `v<major>.<minor>.<patch>` tag runs the `Helm release` workflow. It publishes:

- `ghcr.io/wutz/forgeai-instancetype-operator:<version>`
- `oci://ghcr.io/wutz/charts/forgeai-instancetype-operator:<version>`

The first release needs the two GHCR packages set to public once, under the package settings.

## License

Apache License 2.0. See [LICENSE](LICENSE).
