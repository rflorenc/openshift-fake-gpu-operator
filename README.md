# OpenShift Fake GPU Operator

An OpenShift operator that wraps [run-ai/fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator) functionality into a CRD driven operator. The upstream project uses Helm and requires manual node labeling and SCC grants on OpenShift. This operator replaces that with `FakeGPUConfig` custom resource that handles node labeling, component lifecycle, RBAC, and SCC management, amongst others.

Simulates NVIDIA GPU resources on CPU only OpenShift nodes, enabling development and testing of GPU dependent workloads without physical hardware.

## Features

- Declarative CRD driven configuration via `FakeGPUConfig` CR
- Simulates full GPUs and MIG partitions on any worker node
- Supports 7 GPU profiles: A100, H100, H200, B200, GB200, L40S, T4
- Registers `nvidia.com/gpu` and MIG slice resources with kubelet
- Exports prometheus DCGM metrics
- Automatic node labeling and topology management

## Use Cases

- Testing Kueue GPU quota and scheduling without GPUs
- Developing RHOAI workbenches and pipelines in non-GPU environments
- CI/CD pipeline validation for GPU workloads
- Training and workshop environments

## Quick Start

### Install from OperatorHub

Search for **OpenShift Fake GPU Operator** in the OpenShift console under Operators > OperatorHub and install it.

### Install from CLI

```bash
operator-sdk run bundle quay.io/rlourencc/openshift-fake-gpu-operator-bundle:v0.1.0
```

### Create a FakeGPUConfig

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
oc apply -f fakegpuconfig.yaml
```

GPU resources appear on matching nodes automatically. Label more nodes at any time and the device plugin schedules on them without any operator or CR changes.

### Verify

```bash
oc get fakegpuconfig
```

```
NAME            PROFILE   GPUS   READY   TOTAL   AGE
training-pool   h200      8      3       3       2m
```

```bash
oc get node -o custom-columns=NAME:.metadata.name,GPU:.status.capacity.nvidia\\.com/gpu
```

```
NAME       GPU
worker-0   8
worker-1   8
worker-2   8
```

More sample CRs available at `config/samples`.  


## No Manual Node Labeling Required

Unlike the Helm-based fake-gpu-operator, there's no need to manually label nodes. The `FakeGPUConfig` CR specifies a `nodeSelector`:

1. You create the `FakeGPUConfig` CR with a `nodeSelector`
2. The operator finds matching nodes and labels them with `run.ai/simulated-gpu-node-pool`, `nvidia.com/gpu.deploy.device-plugin`, and `nvidia.com/gpu.deploy.dcgm-exporter`
3. The DaemonSets for the diferent components are rolled out to labeled nodes
4. GPUs appear on the nodes

Adding more worker nodes to the cluster at any time triggers the operator to label them and schedule the device plugin with no CR changes needed.

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

Custom profiles are also supported:

```yaml
spec:
  custom:
    gpuProduct: "NVIDIA-Custom-GPU"
    gpuMemory: 81920
  gpuCount: 4
  nodeSelector:
    node-role.kubernetes.io/worker: ""
```

## MIG Configuration

For MIG-capable profiles, configure slice types:

```yaml
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

## Components

The operator reconciles these components from the `FakeGPUConfig` CR:

| Component | Type | Purpose |
|-----------|------|---------|
| device-plugin | DaemonSet | Registers `nvidia.com/gpu` with kubelet |
| status-updater | Deployment | Manages node labels and topology ConfigMaps |
| metrics-exporter | DaemonSet | DCGM-compatible Prometheus metrics on port 9400 |
| topology-server | Deployment | HTTP API serving GPU topology data |

Components can be individually enabled or disabled:

```yaml
spec:
  components:
    devicePlugin: true
    statusUpdater: true
    metricsExporter: true
    topologyServer: false
```

## Feature Comparison with Helm Chart

| Feature | Helm chart | Operator |
|---------|-----------|----------|
| Device plugin DaemonSet | Yes | Yes |
| nvidia-smi injection | Yes | Yes |
| Status updater Deployment | Yes | Yes |
| DCGM metrics exporter DaemonSet | Yes | Yes |
| Metrics exporter Service | Yes | Yes |
| Topology server Deployment | Yes | Yes |
| Topology server Service | Yes | Yes |
| Topology ConfigMap | Yes | Yes |
| Per-node topology ConfigMaps | Yes | Yes |
| GPU profile ConfigMaps | All 7 profiles | Active profile |
| RuntimeClass `nvidia` | Yes | Yes |
| RBAC | Yes | Yes |
| MIG faker DaemonSet | Yes | Yes |
| DRA plugin DaemonSet | Yes | Yes |
| ComputeDomain DRA DaemonSet | Yes | Yes |
| ComputeDomain controller Deployment | Yes | Yes |
| Node labeling | Manual pre-step | Automated by operator |
| MIG node labels + annotations | Manual | Automated when MIG enabled |
| DRA node labels | Manual | Automated when DRA enabled |
| Helm conflict detection | N/A | Yes |
| OLM / OperatorHub | No | Yes |
| Multiple node pools | Multiple pools in one topology CM | One CR per pool |
| KWOK device plugin | Yes | No |
| KWOK status exporter | Yes | No |
| KWOK DRA plugin | Yes | No |
| NodeResourceTopology / NUMA | Yes | No |
| ServiceMonitor for Prometheus | No | No |

## Development

### Prerequisites

- Go 1.24+
- operator-sdk v1.42+
- podman or docker
- Access to an OpenShift cluster or CRC

### Build and run locally

```bash
make install    # Install CRD
make run        # Run controller locally
```

### Build container image

```bash
make docker-build IMG=quay.io/rlourencc/openshift-fake-gpu-operator:dev CONTAINER_TOOL=podman
```

### Generate OLM bundle

```bash
make bundle IMG=quay.io/rlourencc/openshift-fake-gpu-operator:v0.1.0 VERSION=0.1.0
```

### Test OLM install

```bash
operator-sdk run bundle quay.io/rlourencc/openshift-fake-gpu-operator-bundle:v0.1.0
```

## Related Projects

- [fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator) - Upstream GPU simulation
- [Kueue](https://github.com/kubernetes-sigs/kueue) - Kubernetes job queueing and quota management

## License

Apache License 2.0
