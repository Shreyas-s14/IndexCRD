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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	// for watching resources
	"sigs.k8s.io/controller-runtime/pkg/handler"
	// "sigs.k8s.io/controller-runtime/pkg/source"
)

// FOR LOCAL TESTING!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

// IndexReconciler reconciles a Index object
type IndexReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *IndexReconciler) Init(ctx context.Context) error {
	fmt.Println("This happens on inittttttttttt")
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

	// Fetch the Index instance
	// var index corev1api.Index

	// err := r.Get(ctx, req.NamespacedName, &index)
	// if err != nil {
	// 	log.Info("Index resource not found, might have been deleted", "name", req.NamespacedName)
	// 	return ctrl.Result{}, nil
	// }

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
	// log.Info("yay Pod created in namespace", "namespace", &pod.Namespace, "podName", &pod.Name)
	// log.V(1).Info("yay Pod created", "namespace", pod.GetNamespace(), "podName", pod.GetName())
	log.Info(fmt.Sprintf("Yay Pod created in %s called %s", pod.Namespace, pod.Name))

	// Roles:
	// var clusterRoles rbacv1.ClusterRoleList
	// err = r.List(ctx, &clusterRoles)
	// fmt.Println(err)
	// if err != nil {
	// 	fmt.Println("Error listing roles:", err)
	// } else {
	// 	fmt.Println("ClusterRoles found:", len(clusterRoles.Items))
	// 	for _, clusterRole := range clusterRoles.Items {
	// 		if !strings.HasPrefix(clusterRole.Name, "system:") && !strings.HasPrefix(clusterRole.Name, "kubeadm:") {
	// 			fmt.Println("ClusterRole: ", clusterRole.Name)
	// 			// log.Info(clusterRole.)
	// 		}
	// 	}
	// }

	//RoleBindings:
	var RoleBindings rbacv1.RoleBindingList
	err = r.List(ctx, &RoleBindings)
	if err != nil {
		log.Error(err, "Error Listing RoleBindings")
	} else {
		for _, rb := range RoleBindings.Items {
			// fmt.Println("Rolebindings : ", rb.Name, "Namespace: ", rb.Namespace)
			// fmt.Println(rb.Subjects)
			// fmt.Println(rb.RoleRef.Name)
			for _, sa := range rb.Subjects {
				fmt.Println(sa.Kind, " : ", sa.Name)
			}
		}
	}

	var serviceAccounts corev1.ServiceAccountList
	err = r.List(ctx, &serviceAccounts)
	if err != nil {
		log.Error(err, "Failed to list service accounts")
		return ctrl.Result{}, err
	}

	// Filter out ServiceAccounts that are in "kube-system" or "local-path-storage" namespaces
	for _, sa := range serviceAccounts.Items {
		if sa.Namespace != "kube-system" && sa.Namespace != "local-path-storage" {
			// log.Info(fmt.Sprintf("ServiceAccount: %s, Namespace: %s", sa.Name, sa.Namespace))
			fmt.Println("ServiceAcc: ", sa.Name, "Namespace: ", sa.Namespace)
		}
	}
	return ctrl.Result{}, nil
	// TODO(user): your logic here
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := r.Init(context.Background())
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1api.Index{}).
		Watches(
			&corev1.Pod{},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
