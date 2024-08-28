/*
Copyright 2024.

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
	// "strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1api "controller/index/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// for watching resources
	"sigs.k8s.io/controller-runtime/pkg/handler"
	// "sigs.k8s.io/controller-runtime/pkg/source"
)

// IndexReconciler reconciles a Index object
type IndexReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *IndexReconciler) Init(ctx context.Context) error {
	fmt.Println("This happens on init")
	log := log.FromContext(ctx)

	var roleBindings rbacv1.RoleBindingList
	err := r.List(ctx, &roleBindings)
	if err != nil {
		log.Error(err, "Failed to list role bindings")
		return err
	}
	//kube-system,   kube-public,  local-path-storage
	for _, roleBinding := range roleBindings.Items {
		if roleBinding.Namespace == "kube-system" || roleBinding.Namespace == "kube-public" || roleBinding.Namespace == "local-path-storage" {
			continue
		}
		for _, subject := range roleBinding.Subjects {
			if subject.Kind == "ServiceAccount" {
				serviceAccountName := subject.Name
				namespace := subject.Namespace
				indexName := fmt.Sprintf("index-%s", serviceAccountName)

				// Check if the ServiceAccount exists in the subject's namespace
				var serviceAccount corev1.ServiceAccount
				err := r.Get(ctx, client.ObjectKey{Name: serviceAccountName, Namespace: namespace}, &serviceAccount)
				if err != nil && client.IgnoreNotFound(err) == nil {
					var existingIndex corev1api.Index
					err = r.Get(ctx, client.ObjectKey{Name: indexName, Namespace: "default"}, &existingIndex)
					if err == nil {
						err = r.Delete(ctx, &existingIndex)
						if err != nil {
							log.Error(err, fmt.Sprintf("Failed to delete Index CR: %s", indexName))
						} else {
							log.Info(fmt.Sprintf("Deleted Index CR: %s for ServiceAccount: %s", indexName, serviceAccountName))
						}
					}
					continue
				}
				var existingIndex corev1api.Index
				err = r.Get(ctx, client.ObjectKey{Name: indexName, Namespace: "default"}, &existingIndex)
				if err != nil && client.IgnoreNotFound(err) == nil {
					newIndex := &corev1api.Index{
						ObjectMeta: metav1.ObjectMeta{
							Name:      indexName,
							Namespace: "default",
						},
						Spec: corev1api.IndexSpec{
							ServiceAccount: serviceAccountName,
							NamespaceMap:   make(map[string]corev1api.ResourceNamespace),
						},
						Status: corev1api.IndexStatus{
							LastUpdated: metav1.Now(),
						},
					}
					err = r.Create(ctx, newIndex)
					if err != nil {
						log.Error(err, fmt.Sprintf("Failed to create Index CR for ServiceAccount: %s", serviceAccountName))
						continue
					}
					log.Info(fmt.Sprintf("Created new Index CR: %s for ServiceAccount: %s", indexName, serviceAccountName))
					existingIndex = *newIndex
				} else if err != nil {
					log.Error(err, fmt.Sprintf("Failed to get Index CR: %s", indexName))
					continue
				}
				if existingIndex.Spec.NamespaceMap == nil {
					existingIndex.Spec.NamespaceMap = make(map[string]corev1api.ResourceNamespace) // NamespaceMap is <nil> fix
				}
				namespace = roleBinding.Namespace
				nsResource, exists := existingIndex.Spec.NamespaceMap[namespace]
				if !exists || nsResource.Resources == nil {
					nsResource = corev1api.ResourceNamespace{
						Namespace: namespace,
						Resources: make(map[string][]string),
					}
				}

				var pods corev1.PodList
				err = r.List(ctx, &pods, &client.ListOptions{Namespace: namespace})
				if err != nil {
					log.Error(err, fmt.Sprintf("Failed to list pods in namespace: %s", namespace))
					continue
				}

				activePods := make([]string, 0)
				for _, pod := range pods.Items {
					activePods = append(activePods, pod.Name)
				}
				existingPods := nsResource.Resources["Pod"]
				if len(existingPods) != len(activePods) {
					nsResource.Resources["Pod"] = activePods
				}

				var deployments appsv1.DeploymentList
				if err = r.List(ctx, &deployments, &client.ListOptions{Namespace: namespace}); err != nil {
					log.Error(err, "Failed to list deployments")
					continue
				}
				activeDeployments := make([]string, 0)
				for _, deployment := range deployments.Items {
					activeDeployments = append(activeDeployments, deployment.Name)
				}
				existingDeployments := nsResource.Resources["Deployment"]
				if len(existingDeployments) != len(activeDeployments) {
					nsResource.Resources["Deployment"] = activeDeployments
				}

				var services corev1.ServiceList
				err = r.List(ctx, &services, &client.ListOptions{Namespace: namespace})
				if err != nil {
					log.Error(err, fmt.Sprintf("Failed to list services in namespace: %s", namespace))
					continue
				}
				activeServices := make([]string, 0)
				for _, service := range services.Items {
					activeServices = append(activeServices, service.Name)
				}
				existingServices := nsResource.Resources["Service"]
				if len(existingServices) != len(activeServices) {
					nsResource.Resources["Service"] = activeServices
				}

				existingIndex.Spec.NamespaceMap[namespace] = nsResource
				existingIndex.Status.LastUpdated = metav1.Now()

				err = r.Update(ctx, &existingIndex)
				if err != nil {
					log.Error(err, "Error Updating the CR")
				}
			}
		}
	}

	return nil
}

// +kubebuilder:rbac:groups=core.index.demo,resources=indices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.index.demo,resources=indices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.index.demo,resources=indices/finalizers,verbs=update

func (r *IndexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	err := r.Init(ctx)
	if err != nil {
		log.Error(err, "Failed to initialize Index resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1api.Index{}).
		Watches(
			&corev1.Pod{},
			&handler.EnqueueRequestForObject{},
		).
		Watches(
			&appsv1.Deployment{},
			&handler.EnqueueRequestForObject{},
		).
		Watches(
			&corev1.Service{},
			&handler.EnqueueRequestForObject{},
		).
		Watches(
			&rbacv1.RoleBinding{},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
