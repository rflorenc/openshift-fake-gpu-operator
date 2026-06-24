/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gpuv1alpha1 "github.com/rflorenc/openshift-fake-gpu-operator/api/v1alpha1"
	"github.com/rflorenc/openshift-fake-gpu-operator/internal/profiles"
	"github.com/rflorenc/openshift-fake-gpu-operator/internal/resources"
)

const (
	finalizerName   = "gpu.openshift.io/finalizer"
	nodePoolLabel   = "run.ai/simulated-gpu-node-pool"
	deployDPLabel   = "nvidia.com/gpu.deploy.device-plugin"
	deployDCGMLabel = "nvidia.com/gpu.deploy.dcgm-exporter"
)

type FakeGPUConfigReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Namespace   string
	isOpenShift *bool
}

// +kubebuilder:rbac:groups=gpu.openshift.io,resources=fakegpuconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gpu.openshift.io,resources=fakegpuconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gpu.openshift.io,resources=fakegpuconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=list
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=node.k8s.io,resources=runtimeclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=privileged,verbs=use

func (r *FakeGPUConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	cfg := &gpuv1alpha1.FakeGPUConfig{}
	if err := r.Get(ctx, req.NamespacedName, cfg); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !cfg.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cfg)
	}

	if !controllerutil.ContainsFinalizer(cfg, finalizerName) {
		controllerutil.AddFinalizer(cfg, finalizerName)
		if err := r.Update(ctx, cfg); err != nil {
			return ctrl.Result{}, err
		}
	}

	if conflict, ns := r.detectHelmConflict(ctx); conflict {
		log.Error(nil, "Detected existing Helm-installed fake-gpu-operator", "namespace", ns)
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "HelmConflict",
			Message: fmt.Sprintf("Helm-installed fake-gpu-operator detected in namespace %s. Uninstall it with 'helm uninstall gpu-operator -n %s' before using this operator.", ns, ns),
		})
		_ = r.Status().Update(ctx, cfg)
		return ctrl.Result{}, nil
	}

	profile, err := r.resolveProfile(cfg)
	if err != nil {
		log.Error(err, "Failed to resolve GPU profile")
		meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidProfile",
			Message: err.Error(),
		})
		_ = r.Status().Update(ctx, cfg)
		return ctrl.Result{}, err
	}

	if err := r.reconcileRBAC(ctx, cfg); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling RBAC: %w", err)
	}

	if err := r.reconcileTopologyConfigMap(ctx, cfg, profile); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling topology ConfigMap: %w", err)
	}

	if err := r.reconcileGPUProfileConfigMap(ctx, cfg, profile); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling GPU profile ConfigMap: %w", err)
	}

	if err := r.reconcileDevicePlugin(ctx, cfg); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling device plugin: %w", err)
	}

	components := cfg.Spec.Components
	if components == nil || components.StatusUpdater {
		if err := r.reconcileStatusUpdater(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling status updater: %w", err)
		}
	}

	if components == nil || components.MetricsExporter {
		if err := r.reconcileMetricsExporter(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling metrics exporter: %w", err)
		}
	}

	if components == nil || components.TopologyServer {
		if err := r.reconcileTopologyServer(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling topology server: %w", err)
		}
	}

	if err := r.reconcileServices(ctx, cfg); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling services: %w", err)
	}

	if cfg.Spec.MIG != nil && cfg.Spec.MIG.Enabled {
		if err := r.reconcileMIGFaker(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling MIG faker: %w", err)
		}
	}

	if components != nil && components.DRA {
		if err := r.reconcileDRAPlugin(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling DRA plugin: %w", err)
		}
	}

	if components != nil && components.ComputeDomainController {
		if err := r.reconcileComputeDomainController(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling compute domain controller: %w", err)
		}
		if err := r.reconcileComputeDomainDRA(ctx, cfg); err != nil {
			return ctrl.Result{}, fmt.Errorf("reconciling compute domain DRA: %w", err)
		}
	}

	if err := r.reconcileRuntimeClass(ctx, cfg); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling RuntimeClass: %w", err)
	}

	if err := r.reconcileNodeLabels(ctx, cfg); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling node labels: %w", err)
	}

	if err := r.reconcileNodeTopologyConfigMaps(ctx, cfg, profile); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling node topology ConfigMaps: %w", err)
	}

	return r.updateStatus(ctx, cfg, profile)
}

func (r *FakeGPUConfigReconciler) detectHelmConflict(ctx context.Context) (bool, string) {
	secretList := &corev1.SecretList{}
	if err := r.List(ctx, secretList, client.MatchingLabels{
		"owner": "helm",
		"name":  "gpu-operator",
	}); err != nil {
		return false, ""
	}
	for _, s := range secretList.Items {
		return true, s.Namespace
	}

	cmList := &corev1.ConfigMapList{}
	if err := r.List(ctx, cmList, client.MatchingLabels{
		"app.kubernetes.io/managed-by": "Helm",
	}); err != nil {
		return false, ""
	}
	for _, cm := range cmList.Items {
		if cm.Name == "topology" {
			return true, cm.Namespace
		}
	}

	return false, ""
}

func (r *FakeGPUConfigReconciler) resolveProfile(cfg *gpuv1alpha1.FakeGPUConfig) (profiles.GPUProfile, error) {
	if cfg.Spec.Custom != nil {
		return profiles.GPUProfile{
			Name:    "custom",
			Product: cfg.Spec.Custom.GPUProduct,
			Memory:  cfg.Spec.Custom.GPUMemory,
		}, nil
	}
	if cfg.Spec.GPUProfile == "" {
		return profiles.GPUProfile{}, fmt.Errorf("either gpuProfile or custom must be specified")
	}
	p, ok := profiles.Get(cfg.Spec.GPUProfile)
	if !ok {
		return profiles.GPUProfile{}, fmt.Errorf("unknown GPU profile: %s", cfg.Spec.GPUProfile)
	}
	return p, nil
}

func (r *FakeGPUConfigReconciler) reconcileRBAC(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	components := []string{"device-plugin", "status-updater", "metrics-exporter", "topology-server"}
	for _, comp := range components {
		sa := resources.ServiceAccount(cfg, comp, r.Namespace)
		if err := r.createOrUpdate(ctx, sa, cfg); err != nil {
			return err
		}
	}

	dpRole := resources.DevicePluginClusterRole(cfg)
	if err := r.createOrUpdateClusterScoped(ctx, dpRole); err != nil {
		return err
	}
	dpBinding := resources.ClusterRoleBinding(cfg, "device-plugin", r.Namespace, dpRole)
	if err := r.createOrUpdateClusterScoped(ctx, dpBinding); err != nil {
		return err
	}

	suRole := resources.StatusUpdaterClusterRole(cfg)
	if err := r.createOrUpdateClusterScoped(ctx, suRole); err != nil {
		return err
	}
	suBinding := resources.ClusterRoleBinding(cfg, "status-updater", r.Namespace, suRole)
	if err := r.createOrUpdateClusterScoped(ctx, suBinding); err != nil {
		return err
	}

	meRole := resources.MetricsExporterClusterRole(cfg)
	if err := r.createOrUpdateClusterScoped(ctx, meRole); err != nil {
		return err
	}
	meBinding := resources.ClusterRoleBinding(cfg, "metrics-exporter", r.Namespace, meRole)
	if err := r.createOrUpdateClusterScoped(ctx, meBinding); err != nil {
		return err
	}

	if r.detectOpenShift() {
		sccRole := resources.SCCClusterRole(cfg)
		if err := r.createOrUpdateClusterScoped(ctx, sccRole); err != nil {
			return err
		}
		sccBinding := resources.SCCClusterRoleBinding(cfg, r.Namespace)
		if err := r.createOrUpdateClusterScoped(ctx, sccBinding); err != nil {
			return err
		}
	}

	return nil
}

func (r *FakeGPUConfigReconciler) reconcileTopologyConfigMap(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile) error {
	cm := resources.TopologyConfigMap(cfg, profile, r.Namespace)
	return r.createOrUpdate(ctx, cm, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileGPUProfileConfigMap(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile) error {
	cm := resources.GPUProfileConfigMap(cfg, profile, r.Namespace)
	return r.createOrUpdate(ctx, cm, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileDevicePlugin(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	ds := resources.DevicePluginDaemonSet(cfg, r.Namespace)
	return r.createOrUpdate(ctx, ds, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileStatusUpdater(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	dep := resources.StatusUpdaterDeployment(cfg, r.Namespace)
	return r.createOrUpdate(ctx, dep, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileMetricsExporter(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	ds := resources.MetricsExporterDaemonSet(cfg, r.Namespace)
	return r.createOrUpdate(ctx, ds, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileTopologyServer(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	dep := resources.TopologyServerDeployment(cfg, r.Namespace)
	return r.createOrUpdate(ctx, dep, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileServices(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	metricsSvc := resources.MetricsExporterService(cfg, r.Namespace)
	if err := r.createOrUpdate(ctx, metricsSvc, cfg); err != nil {
		return err
	}
	topoSvc := resources.TopologyServerService(cfg, r.Namespace)
	return r.createOrUpdate(ctx, topoSvc, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileMIGFaker(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	ds := resources.MIGFakerDaemonSet(cfg, r.Namespace)
	return r.createOrUpdate(ctx, ds, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileDRAPlugin(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	ds := resources.DRAPluginDaemonSet(cfg, r.Namespace)
	return r.createOrUpdate(ctx, ds, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileComputeDomainController(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	dep := resources.ComputeDomainControllerDeployment(cfg, r.Namespace)
	return r.createOrUpdate(ctx, dep, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileComputeDomainDRA(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	ds := resources.ComputeDomainDRADaemonSet(cfg, r.Namespace)
	return r.createOrUpdate(ctx, ds, cfg)
}

func (r *FakeGPUConfigReconciler) reconcileRuntimeClass(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	rc := resources.NvidiaRuntimeClass(cfg)
	return r.createOrUpdateClusterScoped(ctx, rc)
}

func (r *FakeGPUConfigReconciler) detectOpenShift() bool {
	if r.isOpenShift != nil {
		return *r.isOpenShift
	}
	_, err := r.Client.RESTMapper().ResourcesFor(
		schema.GroupVersionResource{Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraints"},
	)
	result := err == nil
	r.isOpenShift = &result
	return result
}

func (r *FakeGPUConfigReconciler) nodeListOption(cfg *gpuv1alpha1.FakeGPUConfig) client.MatchingLabels {
	if len(cfg.Spec.NodeSelector) > 0 {
		return client.MatchingLabels(cfg.Spec.NodeSelector)
	}
	return client.MatchingLabels{nodePoolLabel: cfg.Name}
}

func (r *FakeGPUConfigReconciler) reconcileNodeLabels(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) error {
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList, r.nodeListOption(cfg)); err != nil {
		return err
	}

	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		needsUpdate := false

		labels := map[string]string{
			nodePoolLabel:   cfg.Name,
			deployDPLabel:   "true",
			deployDCGMLabel: "true",
		}

		if cfg.Spec.MIG != nil && cfg.Spec.MIG.Enabled {
			labels["node-role.kubernetes.io/runai-dynamic-mig"] = "true"
		}

		if cfg.Spec.Components != nil && cfg.Spec.Components.DRA {
			labels["nvidia.com/gpu.deploy.dra-plugin-gpu"] = "true"
		}

		if cfg.Spec.Components != nil && cfg.Spec.Components.ComputeDomainController {
			labels["nvidia.com/gpu.deploy.compute-domain-dra-plugin"] = "true"
		}

		for k, v := range labels {
			if node.Labels[k] != v {
				if node.Labels == nil {
					node.Labels = make(map[string]string)
				}
				node.Labels[k] = v
				needsUpdate = true
			}
		}

		if needsUpdate {
			if err := r.Update(ctx, node); err != nil {
				return fmt.Errorf("labeling node %s: %w", node.Name, err)
			}
		}
	}

	return nil
}

func (r *FakeGPUConfigReconciler) reconcileNodeTopologyConfigMaps(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile) error {
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList, client.MatchingLabels{nodePoolLabel: cfg.Name}); err != nil {
		return err
	}

	for i := range nodeList.Items {
		cm := resources.NodeTopologyConfigMap(nodeList.Items[i].Name, cfg, profile, r.Namespace)
		if err := r.createOrUpdate(ctx, cm, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (r *FakeGPUConfigReconciler) updateStatus(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig, profile profiles.GPUProfile) (ctrl.Result, error) {
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList, r.nodeListOption(cfg)); err != nil {
		return ctrl.Result{}, err
	}
	cfg.Status.TotalNodes = int32(len(nodeList.Items))

	readyNodes := int32(0)
	for _, node := range nodeList.Items {
		if node.Labels[deployDPLabel] == "true" {
			readyNodes++
		}
	}
	cfg.Status.ReadyNodes = readyNodes
	cfg.Status.GPUProfile = profile.Name
	cfg.Status.GPUCountPerNode = cfg.Spec.GPUCount

	meta.SetStatusCondition(&cfg.Status.Conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: fmt.Sprintf("Fake GPU resources deployed for %d nodes", readyNodes),
	})

	if err := r.Status().Update(ctx, cfg); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FakeGPUConfigReconciler) handleDeletion(ctx context.Context, cfg *gpuv1alpha1.FakeGPUConfig) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	instanceLabels := client.MatchingLabels{"app.kubernetes.io/instance": cfg.Name}
	ns := client.InNamespace(r.Namespace)

	dsList := &appsv1.DaemonSetList{}
	if err := r.List(ctx, dsList, instanceLabels, ns); err == nil {
		for i := range dsList.Items {
			if err := r.Delete(ctx, &dsList.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete DaemonSet", "name", dsList.Items[i].Name)
			}
		}
	}

	deployList := &appsv1.DeploymentList{}
	if err := r.List(ctx, deployList, instanceLabels, ns); err == nil {
		for i := range deployList.Items {
			if err := r.Delete(ctx, &deployList.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete Deployment", "name", deployList.Items[i].Name)
			}
		}
	}

	svcList := &corev1.ServiceList{}
	if err := r.List(ctx, svcList, instanceLabels, ns); err == nil {
		for i := range svcList.Items {
			if err := r.Delete(ctx, &svcList.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete Service", "name", svcList.Items[i].Name)
			}
		}
	}

	saList := &corev1.ServiceAccountList{}
	if err := r.List(ctx, saList, instanceLabels, ns); err == nil {
		for i := range saList.Items {
			if err := r.Delete(ctx, &saList.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete ServiceAccount", "name", saList.Items[i].Name)
			}
		}
	}

	cmList := &corev1.ConfigMapList{}
	if err := r.List(ctx, cmList, instanceLabels, ns); err == nil {
		for i := range cmList.Items {
			if err := r.Delete(ctx, &cmList.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete ConfigMap", "name", cmList.Items[i].Name)
			}
		}
	}

	nodeTopoCMs := &corev1.ConfigMapList{}
	if err := r.List(ctx, nodeTopoCMs, client.MatchingLabels{"node-topology": "true"}, ns); err == nil {
		for i := range nodeTopoCMs.Items {
			if err := r.Delete(ctx, &nodeTopoCMs.Items[i]); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete node topology ConfigMap", "name", nodeTopoCMs.Items[i].Name)
			}
		}
	}

	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList, client.MatchingLabels{nodePoolLabel: cfg.Name}); err != nil {
		return ctrl.Result{}, err
	}

	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		delete(node.Labels, nodePoolLabel)
		delete(node.Labels, deployDPLabel)
		delete(node.Labels, deployDCGMLabel)
		delete(node.Labels, "node-role.kubernetes.io/runai-dynamic-mig")
		delete(node.Labels, "nvidia.com/gpu.deploy.dra-plugin-gpu")
		delete(node.Labels, "nvidia.com/gpu.deploy.compute-domain-dra-plugin")
		if err := r.Update(ctx, node); err != nil {
			log.Error(err, "Failed to remove labels from node", "node", node.Name)
		}
	}

	clusterScopedResources := []client.Object{
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-device-plugin-" + cfg.Name}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-device-plugin-" + cfg.Name}},
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-status-updater-" + cfg.Name}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-status-updater-" + cfg.Name}},
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-metrics-exporter-" + cfg.Name}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-metrics-exporter-" + cfg.Name}},
		&nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "nvidia"}},
	}
	if r.detectOpenShift() {
		clusterScopedResources = append(clusterScopedResources,
			&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-scc-" + cfg.Name}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "fake-gpu-scc-" + cfg.Name}},
		)
	}
	for _, obj := range clusterScopedResources {
		if err := r.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete cluster-scoped resource", "resource", obj.GetName())
		}
	}

	controllerutil.RemoveFinalizer(cfg, finalizerName)
	return ctrl.Result{}, r.Update(ctx, cfg)
}

func (r *FakeGPUConfigReconciler) createOrUpdate(ctx context.Context, obj client.Object, owner *gpuv1alpha1.FakeGPUConfig) error {
	existing := obj.DeepCopyObject().(client.Object)
	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, obj)
	}
	if err != nil {
		return err
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, obj)
}

func (r *FakeGPUConfigReconciler) createOrUpdateClusterScoped(ctx context.Context, obj client.Object) error {
	existing := obj.DeepCopyObject().(client.Object)
	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName()}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, obj)
	}
	if err != nil {
		return err
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, obj)
}

func (r *FakeGPUConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpuv1alpha1.FakeGPUConfig{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Watches(&corev1.Node{}, handler.EnqueueRequestsFromMapFunc(r.nodeToFakeGPUConfig)).
		Named("fakegpuconfig").
		Complete(r)
}

func (r *FakeGPUConfigReconciler) nodeToFakeGPUConfig(ctx context.Context, obj client.Object) []reconcile.Request {
	cfgList := &gpuv1alpha1.FakeGPUConfigList{}
	if err := r.List(ctx, cfgList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	node := obj.(*corev1.Node)
	for _, cfg := range cfgList.Items {
		if r.nodeMatchesCfg(node, &cfg) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: cfg.Name},
			})
		}
	}
	return requests
}

func (r *FakeGPUConfigReconciler) nodeMatchesCfg(node *corev1.Node, cfg *gpuv1alpha1.FakeGPUConfig) bool {
	if len(cfg.Spec.NodeSelector) > 0 {
		for k, v := range cfg.Spec.NodeSelector {
			if node.Labels[k] != v {
				return false
			}
		}
		return true
	}
	return node.Labels[nodePoolLabel] == cfg.Name
}
