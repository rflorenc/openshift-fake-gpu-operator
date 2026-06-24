# OpenShift Fake GPU Operator

An OpenShift operator that wraps [run-ai/fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator) functionality.  
The reference "upstream" project from NVIDIA RunAI uses Helm and requires manual node labeling and SCC grants on OpenShift.  
This project replaces that with a `FakeGPUConfig` custom resource that handles node labeling, component lifecycle, RBAC, and SCC management.


It simulates NVIDIA GPU resources on CPU only OpenShift nodes, enabling development and testing of GPU dependent workloads without physical hardware.

## Features

- Declarative CRD driven configuration via `FakeGPUConfig` CR
- 7 built-in GPU profiles: A100, H100, H200, B200, GB200, L40S, T4
- Full GPU and MIG partition simulation with `nvidia.com/gpu` and `nvidia.com/mig-*` resources
- DRA and ComputeDomain support for dynamic GPU allocation
- Automatic node labeling, SCC grants, and topology management
- Prometheus DCGM metrics export
- Helm conflict detection
- Works on both OpenShift and plain Kubernetes

## Usage

### Install

```bash
kubectl apply -f https://raw.githubusercontent.com/rflorenc/openshift-fake-gpu-operator/main/dist/install.yaml
```

### Uninstall

```bash
kubectl delete -f https://raw.githubusercontent.com/rflorenc/openshift-fake-gpu-operator/main/dist/install.yaml
```

## Usage

### 1. Create a FakeGPUConfig

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: training-pool
spec:
  gpuProfile: h200
  gpuCount: 8
  nodeSelector:
    node-role.kubernetes.io/worker: ""
```

```bash
kubectl apply -f fakegpuconfig.yaml
```

### 2. Verify

```bash
kubectl get fakegpuconfig
```

```
NAME            PROFILE   GPUS   READY   TOTAL   AGE
training-pool   h200      8      3       3       2m
```

```bash
kubectl get node -o custom-columns=NAME:.metadata.name,GPU:.status.capacity.nvidia\\.com/gpu
```

```
NAME       GPU
worker-0   8
worker-1   8
worker-2   8
```

### 3. Remove

```bash
kubectl delete fakegpuconfig training-pool
```

All child resources, node labels, and GPU capacity are cleaned up automatically.

## Examples

All examples are in [`config/samples/`](config/samples/).

### Basic — full GPUs on all workers

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: training-pool
spec:
  gpuProfile: h200
  gpuCount: 8
  nodeSelector:
    node-role.kubernetes.io/worker: ""
```

### MIG — multi-tenant GPU sharing

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: mig-pool
spec:
  gpuProfile: h200
  gpuCount: 8
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  mig:
    enabled: true
    strategy: mixed
    devices:
      - name: 1g.18gb
        count: 16
      - name: 2g.35gb
        count: 8
      - name: 3g.71gb
        count: 8
```

### Custom profile — user-defined GPU

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: custom-pool
spec:
  custom:
    gpuProduct: "NVIDIA-Custom-GPU"
    gpuMemory: 81920
  gpuCount: 4
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  components:
    devicePlugin: true
    statusUpdater: true
    metricsExporter: false
    topologyServer: false
```

### DRA plus ComputeDomain

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: dra-pool
spec:
  gpuProfile: a100
  gpuCount: 8
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  mig:
    enabled: true
    strategy: mixed
    devices:
      - name: 1g.5gb
        count: 7
      - name: 2g.10gb
        count: 3
      - name: 3g.20gb
        count: 2
  components:
    devicePlugin: true
    statusUpdater: true
    metricsExporter: true
    topologyServer: true
    dra: true
    computeDomainController: true
```

### Manual node labeling — Helm-compatible workflow

Omit `nodeSelector` and label nodes yourself. The CR name must match the label value.

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: default
spec:
  gpuProfile: h100
  gpuCount: 4
```

```bash
kubectl label node <node-name> run.ai/simulated-gpu-node-pool=default
```

### Image overrides — test a new upstream version

```yaml
apiVersion: gpu.openshift.io/v1alpha1
kind: FakeGPUConfig
metadata:
  name: test-pool
spec:
  gpuProfile: h200
  gpuCount: 8
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  images:
    tag: "0.2.0"
    # registry: my-mirror.example.com/fake-gpu
    # overrides:
    #   device-plugin: my-registry.example.com/custom-device-plugin:test
```

## Node Selection

Two modes for selecting which nodes get fake GPUs:

**Automatic** — set `nodeSelector` in the CR. The operator finds matching nodes and labels them. New nodes matching the selector are picked up automatically.

**Manual** — omit `nodeSelector`. Label nodes with `run.ai/simulated-gpu-node-pool=<cr-name>` and the operator reconciles them. Compatible with the upstream Helm chart workflow.

## Image Configuration

The operator deploys upstream [run-ai/fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator) container images. Image versions can be controlled without rebuilding the operator:

| Method | Scope | Use case |
|--------|-------|----------|
| `spec.images.overrides.<component>` | Single component | Pin one component to a test image |
| `spec.images.tag` | All components in one CR | Test a new upstream release |
| `spec.images.registry` | All components in one CR | Use a private mirror |
| `RELATED_IMAGE_TAG` env var | Cluster-wide default | Set default version for all CRs |
| `RELATED_IMAGE_REGISTRY` env var | Cluster-wide default | Set default registry for all CRs |

Priority: per-component override > CR tag/registry > env var > compiled default.

Valid component keys for overrides: `device-plugin`, `status-updater`, `metrics-exporter`, `topology-server`, `mig-faker`, `dra-plugin`, `compute-domain-controller`, `compute-domain-dra`.

## GPU Profiles

| Profile | GPU | Memory | MIG | Architecture |
|---------|-----|--------|-----|--------------|
| a100 | NVIDIA A100-SXM4-40GB | 40 GiB | Yes | Ampere |
| h100 | NVIDIA H100 80GB HBM3 | 80 GiB | Yes | Hopper |
| h200 | NVIDIA H200 | 141 GiB | Yes | Hopper |
| b200 | NVIDIA B200 | 192 GiB | Yes | Blackwell |
| gb200 | NVIDIA GB200 NVL | 192 GiB | Yes | Blackwell |
| l40s | NVIDIA L40S | 48 GiB | No | Ada Lovelace |
| t4 | Tesla T4 | 16 GiB | No | Turing |

## Components

| Component | Type | Purpose |
|-----------|------|---------|
| device-plugin | DaemonSet | Registers `nvidia.com/gpu` with kubelet |
| status-updater | Deployment | Manages node labels and topology ConfigMaps |
| metrics-exporter | DaemonSet | DCGM Prometheus metrics on port 9400 |
| topology-server | Deployment | HTTP API serving GPU topology data |
| mig-faker | DaemonSet | MIG partition simulation |
| dra-plugin | DaemonSet | Dynamic Resource Allocation for GPUs |
| compute-domain-controller | Deployment | ComputeDomain CRD reconciler |
| compute-domain-dra | DaemonSet | ComputeDomain DRA plugin |

## Feature Comparison with Helm Chart

| Feature | Helm chart | Operator |
|---------|-----------|----------|
| Device plugin, status updater, metrics, topology | Yes | Yes |
| nvidia-smi injection | Yes | Yes |
| MIG faker | Yes | Yes |
| DRA + ComputeDomain | Yes | Yes |
| RBAC, RuntimeClass, Services | Yes | Yes |
| Node labeling | Manual | Automated or manual |
| SCC grants | Manual | Automated |
| OLM / OperatorHub | No | Yes |
| Helm conflict detection | N/A | Yes |
| Plain Kubernetes support | Yes | Yes |
| KWOK support | Yes | No |
| NodeResourceTopology / NUMA | Yes | No |

## Development

```bash
make install                    # Install CRD
make run                        # Run controller locally
make release VERSION=0.2.0      # Build, push, release with docker
make release VERSION=0.2.0 CONTAINER_TOOL=podman # ...
```

Requires Go 1.24+, operator-sdk v1.42+, podman or docker, and access to an OpenShift cluster or any Kubernetes cluster.

## Related Projects

- [fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator) — upstream GPU simulation
- [fake-gpu-booking](https://github.com/rflorenc/fake-gpu-booking) — GPU booking system with Kueue, RHOAI hardware profiles, and OpenShift console plugin
- [Kueue](https://github.com/kubernetes-sigs/kueue) — Kubernetes job queueing and quota management

## License

Apache License 2.0
