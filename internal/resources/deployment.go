package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
)

const (
	StatusUpdaterImage           = "ghcr.io/run-ai/fake-gpu-operator/status-updater:0.1.0"
	TopologyServerImage          = "ghcr.io/run-ai/fake-gpu-operator/topology-server:0.1.0"
	ComputeDomainControllerImage = "ghcr.io/run-ai/fake-gpu-operator/compute-domain-controller:0.1.0"
)

func StatusUpdaterDeployment(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-updater-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "status-updater"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "status-updater"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "status-updater"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "status-updater-" + cfg.Name,
					Containers: []corev1.Container{
						{
							Name:  "status-updater",
							Image: StatusUpdaterImage,
							Env: []corev1.EnvVar{
								{Name: "TOPOLOGY_CM_NAMESPACE", Value: namespace},
								{Name: "TOPOLOGY_CM_NAME", Value: "topology"},
								{Name: "FAKE_GPU_OPERATOR_NAMESPACE", Value: namespace},
								{Name: "RESOURCE_RESERVATION_NAMESPACE", Value: "runai-reservation"},
								{Name: "DISABLE_NODE_LABELING", Value: "true"},
								{Name: "RUNAI_INTEGRATION_ENABLED", Value: "false"},
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func ComputeDomainControllerDeployment(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compute-domain-controller-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "compute-domain-controller"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "compute-domain-controller"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "compute-domain-controller"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "device-plugin-" + cfg.Name,
					Containers: []corev1.Container{
						{
							Name:  "compute-domain-controller",
							Image: ComputeDomainControllerImage,
							Env: []corev1.EnvVar{
								{Name: "METRICS_BIND_ADDRESS", Value: ":8080"},
								{Name: "HEALTH_PROBE_BIND_ADDRESS", Value: ":8081"},
								{Name: "LEADER_ELECT", Value: "false"},
							},
						},
					},
				},
			},
		},
	}
}

func TopologyServerDeployment(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "topology-server-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "topology-server"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "topology-server"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "topology-server"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "topology-server-" + cfg.Name,
					Containers: []corev1.Container{
						{
							Name:  "topology-server",
							Image: TopologyServerImage,
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
							},
							Env: []corev1.EnvVar{
								{Name: "TOPOLOGY_CM_NAMESPACE", Value: namespace},
								{Name: "TOPOLOGY_CM_NAME", Value: "topology"},
							},
						},
					},
				},
			},
		},
	}
}
