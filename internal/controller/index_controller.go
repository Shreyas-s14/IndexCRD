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
	// "encoding/json"
	"fmt"
	// "strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1api "controller/index/api/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	// for watching resources
	"sigs.k8s.io/controller-runtime/pkg/handler"
	// "sigs.k8s.io/controller-runtime/pkg/source"
)

// FOR LOCAL TESTING!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
type ResourceNamespace struct {
	Namespace string              `json:"namespace,omitempty"`
	Resources map[string][]string `json:"resources,omitempty"`
}

type IndexSpec struct {
	ServiceAccount string                       `json:"serviceAccount,omitempty"`
	NamespaceMap   map[string]ResourceNamespace `json:"namespaceMap,omitempty"`
}

// ABOVE FOR LOCAL TESTING: wtihout deploying a cr

// IndexReconciler reconciles a Index object
type IndexReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// NamespaceMap map[string]ResourceNamespace // remove later
	IndexSpecs  map[string]IndexSpec // remove
	Initialized bool                 // remove later
}

func (r *IndexReconciler) Init(ctx context.Context) error {
	fmt.Println("This happens on inittttttttttt")
	log := log.FromContext(ctx)

	// List all RoleBindings that are not in kube-system, kube-public, or local-path-storage namespaces
	r.IndexSpecs = make(map[string]IndexSpec)

	// List all RoleBindings that are not in kube-system, kube-public, or local-path-storage namespaces
	var roleBindings rbacv1.RoleBindingList
	err := r.List(ctx, &roleBindings)
	if err != nil {
		log.Error(err, "Failed to list role bindings")
		return err
	}

	// Process RoleBindings to populate IndexSpec instances
	for _, roleBinding := range roleBindings.Items {
		if roleBinding.Namespace == "kube-system" || roleBinding.Namespace == "kube-public" || roleBinding.Namespace == "local-path-storage" {
			continue
		}

		namespace := roleBinding.Namespace

		for _, subject := range roleBinding.Subjects {
			if subject.Kind == "ServiceAccount" {
				serviceAccountName := subject.Name

				// Check if the service account already has an IndexSpec
				indexSpec, exists := r.IndexSpecs[serviceAccountName]
				if !exists {
					// Initialize a new IndexSpec for the service account
					indexSpec = IndexSpec{
						ServiceAccount: serviceAccountName,
						NamespaceMap:   make(map[string]ResourceNamespace),
					}
				}

				// Initialize the namespace entry in the NamespaceMap if it doesn't exist
				if _, exists := indexSpec.NamespaceMap[namespace]; !exists {
					indexSpec.NamespaceMap[namespace] = ResourceNamespace{
						Namespace: namespace,
						Resources: make(map[string][]string),
					}
				}

				// List Pods in the current namespace
				var pods corev1.PodList
				err := r.List(ctx, &pods, &client.ListOptions{Namespace: namespace})
				if err != nil {
					log.Error(err, fmt.Sprintf("Failed to list pods in namespace: %s", namespace))
					continue
				}

				// Associate the Pods with the namespace in the IndexSpec
				for _, pod := range pods.Items {
					indexSpec.NamespaceMap[namespace].Resources["Pod"] = append(indexSpec.NamespaceMap[namespace].Resources["Pod"], pod.Name)
				}

				// Update the IndexSpec in the map
				r.IndexSpecs[serviceAccountName] = indexSpec
			}
		}
	}

	// Log the initialized IndexSpecs for debugging purposes
	// log.Info("Initialized IndexSpecs", "IndexSpecs", r.IndexSpecs)

	r.Initialized = true
	log.Info(fmt.Sprintf("%d", len(r.IndexSpecs)))
	fmt.Println()
	fmt.Println()
	fmt.Println(r.IndexSpecs)
	fmt.Println()
	fmt.Println()
	return nil
}

// Updating pod data
func (r *IndexReconciler) UpdateServiceAccountResources(ctx context.Context, pod *corev1.Pod) error {
	log := log.FromContext(ctx)
	namespace := pod.Namespace
	podName := pod.Name
	fmt.Println(len(r.IndexSpecs))
	for serviceAccount, indexSpec := range r.IndexSpecs {
		if resourceNamespace, exists := indexSpec.NamespaceMap[namespace]; exists {
			// Check if the pod is already in the list to avoid duplicates
			podList := resourceNamespace.Resources["Pod"]
			podExists := false
			for _, existingPod := range podList {
				if existingPod == podName {
					podExists = true
					break
				}
			}

			if !podExists {
				resourceNamespace.Resources["Pod"] = append(resourceNamespace.Resources["Pod"], podName)
				indexSpec.NamespaceMap[namespace] = resourceNamespace
				r.IndexSpecs[serviceAccount] = indexSpec

				log.Info(fmt.Sprintf("Updated IndexSpec for ServiceAccount: %s with new pod: %s in Namespace: %s", serviceAccount, podName, namespace))
			}
		}
	}

	return nil
}

// +kubebuilder:rbac:groups=core.index.demo,resources=indices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.index.demo,resources=indices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.index.demo,resources=indices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Index object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *IndexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// checking initialization
	if !r.Initialized {
		if err := r.Init(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}
// Fetch the Pod instance
	var pod corev1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)

	if err != nil {
		// If the Pod was not found, it's likely a deletion event
		log.Info(fmt.Sprintf("Welp pod %s deleted in %s", pod.Name, pod.Namespace))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// to check if it's a preexisting pod:
	if pod.Status.Phase == corev1.PodRunning {
		return ctrl.Result{}, nil
	}

	// if the pod is being deleted
	if pod.ObjectMeta.DeletionTimestamp != nil || pod.ObjectMeta.Generation > 1 {
		return ctrl.Result{}, nil
	}
	r.UpdateServiceAccountResources(ctx, &pod)
	log.Info(fmt.Sprintf("Yay Pod created in %s called %s", pod.Namespace, pod.Name))
	

	fmt.Println(r.IndexSpecs)

	//RoleBindings:
	// var RoleBindings rbacv1.RoleBindingList
	// err = r.List(ctx, &RoleBindings)
	// if err != nil {
	// 	log.Error(err, "Error Listing RoleBindings")
	// } else {
	// 	for _, rb := range RoleBindings.Items {
	// 		// fmt.Println("Rolebindings : ", rb.Name, "Namespace: ", rb.Namespace)
	// 		// fmt.Println(rb.Subjects)
	// 		// fmt.Println(rb.RoleRef.Name)
	// 		for _, sa := range rb.Subjects {
	// 			fmt.Println(sa.Kind, " : ", sa.Name)
	// 		}
	// 	}
	// }
	return ctrl.Result{}, nil
	// TODO(user): your logic here
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// err := r.Init(context.Background())
	// if err != nil {
	// 	return err
	// }
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1api.Index{}).
		Watches(
			&corev1.Pod{},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
