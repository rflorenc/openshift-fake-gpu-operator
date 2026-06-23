package resources

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
	"github.com/rflorenc/openshift-fake-gpu-operator/internal/profiles"
)

func TopologyConfigMap(cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile, namespace string) *corev1.ConfigMap {
	migStrategy := "none"
	if cfg.Spec.MIG != nil && cfg.Spec.MIG.Enabled {
		migStrategy = cfg.Spec.MIG.Strategy
	}

	gpuProduct := profile.Product
	gpuMemory := profile.Memory
	if cfg.Spec.Custom != nil {
		gpuProduct = cfg.Spec.Custom.GPUProduct
		gpuMemory = cfg.Spec.Custom.GPUMemory
	}

	topology := fmt.Sprintf(`migStrategy: %s
nodePoolLabelKey: run.ai/simulated-gpu-node-pool
nodePools:
  %s:
    gpuCount: %d
    gpuMemory: %d
    gpuProduct: %s`,
		migStrategy, cfg.Name, cfg.Spec.GPUCount, gpuMemory, gpuProduct)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "topology",
			Namespace: namespace,
		},
		Data: map[string]string{
			"topology.yml": topology,
		},
	}
}

func GPUProfileConfigMap(profile profiles.GPUProfile, namespace string) *corev1.ConfigMap {
	profileYAML := fmt.Sprintf(`device_count: 8
device_defaults:
  name: %q
  architecture: %q
  memory:
    total_bytes: %d`,
		profile.Product, profile.Architecture, int64(profile.Memory)*1024*1024)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpu-profile-" + profile.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"fake-gpu-operator/gpu-profile": "true",
			},
		},
		Data: map[string]string{
			"profile.yaml": profileYAML,
		},
	}
}

func NodeTopologyConfigMap(nodeName string, cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile, namespace string) *corev1.ConfigMap {
	gpuProduct := profile.Product
	gpuMemory := profile.Memory
	if cfg.Spec.Custom != nil {
		gpuProduct = cfg.Spec.Custom.GPUProduct
		gpuMemory = cfg.Spec.Custom.GPUMemory
	}

	migStrategy := "none"
	if cfg.Spec.MIG != nil && cfg.Spec.MIG.Enabled {
		migStrategy = cfg.Spec.MIG.Strategy
	}

	var gpuEntries []string
	for i := int32(0); i < cfg.Spec.GPUCount; i++ {
		gpuEntries = append(gpuEntries, fmt.Sprintf(`    - id: GPU-%s-%d
      status:
        allocatedBy:
          namespace: ""
          pod: ""
          container: ""
        podGpuUsageStatus: {}`, nodeName, i))
	}

	topology := fmt.Sprintf(`gpuMemory: %d
gpuProduct: %s
gpus:
%s
migStrategy: %s`, gpuMemory, gpuProduct, strings.Join(gpuEntries, "\n"), migStrategy)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "topology-" + nodeName,
			Namespace: namespace,
			Labels: map[string]string{
				"node-topology": "true",
				"node-name":     nodeName,
			},
		},
		Data: map[string]string{
			"topology.yml": topology,
		},
	}
}
