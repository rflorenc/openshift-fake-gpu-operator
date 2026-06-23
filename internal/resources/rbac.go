package resources

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
)

func ServiceAccount(cfg *gpuv1alpha1.FakeGPUConfig, component, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      component + "-" + cfg.Name,
			Namespace: namespace,
			Labels:    commonLabels(cfg, component),
		},
	}
}

func DevicePluginClusterRole(cfg *gpuv1alpha1.FakeGPUConfig) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "fake-gpu-device-plugin-" + cfg.Name,
			Labels: commonLabels(cfg, "device-plugin"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func StatusUpdaterClusterRole(cfg *gpuv1alpha1.FakeGPUConfig) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "fake-gpu-status-updater-" + cfg.Name,
			Labels: commonLabels(cfg, "status-updater"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "patch"},
			},
		},
	}
}

func MetricsExporterClusterRole(cfg *gpuv1alpha1.FakeGPUConfig) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "fake-gpu-metrics-exporter-" + cfg.Name,
			Labels: commonLabels(cfg, "metrics-exporter"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func SCCClusterRole(cfg *gpuv1alpha1.FakeGPUConfig) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "fake-gpu-scc-" + cfg.Name,
			Labels: commonLabels(cfg, "scc"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{"privileged"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func SCCClusterRoleBinding(cfg *gpuv1alpha1.FakeGPUConfig, namespace string) *rbacv1.ClusterRoleBinding {
	subjects := []rbacv1.Subject{}
	for _, comp := range []string{"device-plugin", "metrics-exporter", "status-updater"} {
		subjects = append(subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      comp + "-" + cfg.Name,
			Namespace: namespace,
		})
	}

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "fake-gpu-scc-" + cfg.Name,
			Labels: commonLabels(cfg, "scc"),
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fake-gpu-scc-" + cfg.Name,
		},
	}
}

func ClusterRoleBinding(cfg *gpuv1alpha1.FakeGPUConfig, component, namespace string, clusterRole *rbacv1.ClusterRole) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRole.Name,
			Labels: commonLabels(cfg, component),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      component + "-" + cfg.Name,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}
}
