package resources

import (
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
)

func NvidiaRuntimeClass(cfg *gpuv1alpha1.FakeGPUConfig) *nodev1.RuntimeClass {
	return &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "nvidia",
			Labels: commonLabels(cfg, "runtime"),
		},
		Handler: "nvidia",
	}
}
