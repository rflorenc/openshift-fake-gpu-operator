package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
)

func MetricsExporterService(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nvidia-dcgm-exporter-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "metrics-exporter"),
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels(cfg, "metrics-exporter"),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "gpu-metrics",
					Port:       9400,
					TargetPort: intstr.FromInt32(9400),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func TopologyServerService(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "topology-server-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "topology-server"),
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels(cfg, "topology-server"),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt32(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
