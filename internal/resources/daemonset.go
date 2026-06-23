package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
)

const (
	DevicePluginImage     = "ghcr.io/run-ai/fake-gpu-operator/device-plugin:0.1.0"
	MetricsExporterImage  = "ghcr.io/run-ai/fake-gpu-operator/status-exporter:0.1.0"
	MIGFakerImage         = "ghcr.io/run-ai/fake-gpu-operator/mig-faker:0.1.0"
	DRAPluginImage        = "ghcr.io/run-ai/fake-gpu-operator/dra-plugin-gpu:0.1.0"
	ComputeDomainDRAImage = "ghcr.io/run-ai/fake-gpu-operator/compute-domain-dra-plugin:0.1.0"
)

func DevicePluginDaemonSet(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.DaemonSet {
	privileged := true
	hostPathDir := corev1.HostPathDirectory
	hostPathDirOrCreate := corev1.HostPathDirectoryOrCreate

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "device-plugin-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "device-plugin"),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "device-plugin"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "device-plugin"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "device-plugin-" + cfg.Name,
					NodeSelector: map[string]string{
						"nvidia.com/gpu.deploy.device-plugin": "true",
					},
					Containers: []corev1.Container{
						{
							Name:  "nvidia-device-plugin-ctr",
							Image: DevicePluginImage,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{Name: "TOPOLOGY_CM_NAMESPACE", Value: namespace},
								{Name: "TOPOLOGY_CM_NAME", Value: "topology"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "device-plugin", MountPath: "/var/lib/kubelet/device-plugins"},
								{Name: "runai-bin-directory", MountPath: "/runai/bin"},
								{Name: "runai-shared-directory", MountPath: "/runai/shared"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "device-plugin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/device-plugins",
									Type: &hostPathDir,
								},
							},
						},
						{
							Name: "runai-bin-directory",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/runai/bin",
									Type: &hostPathDirOrCreate,
								},
							},
						},
						{
							Name: "runai-shared-directory",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/runai/shared",
									Type: &hostPathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
}

func MetricsExporterDaemonSet(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nvidia-dcgm-exporter-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "metrics-exporter"),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "metrics-exporter"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "metrics-exporter"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "metrics-exporter-" + cfg.Name,
					NodeSelector: map[string]string{
						"nvidia.com/gpu.deploy.dcgm-exporter": "true",
					},
					Containers: []corev1.Container{
						{
							Name:  "nvidia-dcgm-exporter",
							Image: MetricsExporterImage,
							Ports: []corev1.ContainerPort{
								{Name: "metrics", ContainerPort: 9400, Protocol: corev1.ProtocolTCP},
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
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

func MIGFakerDaemonSet(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.DaemonSet {
	privileged := true
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mig-faker-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "mig-faker"),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "mig-faker"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "mig-faker"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "device-plugin-" + cfg.Name,
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/runai-dynamic-mig": "true",
					},
					Containers: []corev1.Container{
						{
							Name:  "mig-faker",
							Image: MIGFakerImage,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
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

func DRAPluginDaemonSet(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.DaemonSet {
	privileged := true
	hostPathDirOrCreate := corev1.HostPathDirectoryOrCreate

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dra-plugin-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "dra-plugin"),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "dra-plugin"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "dra-plugin"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "device-plugin-" + cfg.Name,
					NodeSelector: map[string]string{
						"nvidia.com/gpu.deploy.dra-plugin-gpu": "true",
					},
					Containers: []corev1.Container{
						{
							Name:    "dra-plugin",
							Image:   DRAPluginImage,
							Command: []string{"dra-plugin-gpu"},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{Name: "CDI_ROOT", Value: "/var/run/cdi"},
								{Name: "KUBELET_REGISTRAR_DIRECTORY_PATH", Value: "/var/lib/kubelet/plugins_registry"},
								{Name: "KUBELET_PLUGINS_DIRECTORY_PATH", Value: "/var/lib/kubelet/plugins"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "plugins-registry", MountPath: "/var/lib/kubelet/plugins_registry"},
								{Name: "plugins", MountPath: "/var/lib/kubelet/plugins"},
								{Name: "cdi", MountPath: "/var/run/cdi"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "plugins-registry",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins_registry",
									Type: &hostPathDirOrCreate,
								},
							},
						},
						{
							Name: "plugins",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins",
									Type: &hostPathDirOrCreate,
								},
							},
						},
						{
							Name: "cdi",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/cdi",
									Type: &hostPathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
}

func ComputeDomainDRADaemonSet(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *appsv1.DaemonSet {
	privileged := true
	hostPathDirOrCreate := corev1.HostPathDirectoryOrCreate

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compute-domain-dra-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, "compute-domain-dra"),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(cfg, "compute-domain-dra"),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels(cfg, "compute-domain-dra"),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "device-plugin-" + cfg.Name,
					NodeSelector: map[string]string{
						"nvidia.com/gpu.deploy.compute-domain-dra-plugin": "true",
					},
					Containers: []corev1.Container{
						{
							Name:  "compute-domain-dra",
							Image: ComputeDomainDRAImage,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{Name: "CDI_ROOT", Value: "/etc/cdi"},
								{Name: "KUBELET_REGISTRAR_DIRECTORY_PATH", Value: "/var/lib/kubelet/plugins_registry"},
								{Name: "KUBELET_PLUGINS_DIRECTORY_PATH", Value: "/var/lib/kubelet/plugins"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "plugins-registry", MountPath: "/var/lib/kubelet/plugins_registry"},
								{Name: "plugins", MountPath: "/var/lib/kubelet/plugins"},
								{Name: "cdi", MountPath: "/etc/cdi"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "plugins-registry",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins_registry",
									Type: &hostPathDirOrCreate,
								},
							},
						},
						{
							Name: "plugins",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins",
									Type: &hostPathDirOrCreate,
								},
							},
						},
						{
							Name: "cdi",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/cdi",
									Type: &hostPathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
}

func commonLabels(cfg *gpuv1alpha1.FakeGPUConfig, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "fake-gpu-operator",
		"app.kubernetes.io/instance":   cfg.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/managed-by": "openshift-fake-gpu-operator",
	}
}

func selectorLabels(cfg *gpuv1alpha1.FakeGPUConfig, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "fake-gpu-operator",
		"app.kubernetes.io/instance":  cfg.Name,
		"app.kubernetes.io/component": component,
	}
}
