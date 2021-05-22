/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	minecraftv1 "github.com/hsmade/minecraft-operator/api/v1"
)

// ServerReconciler reconciles a Server object
type ServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	serverOwnerKey = ".meta.owner.name"
	apiGVStr       = minecraftv1.GroupVersion.String()
)

//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=servers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=servers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
// TODO:
// - manifest generator per object
// - if we're disabled, delete and ignore not found, or find and delete if found -> return with requeue of 30s
// - always generate the manifests
// - find the live ones
// - if it's missing, submit -> requeue 10s
// - if it's there, compare
// - if it's different, patch -> requeue 10s
// - get liveness -> update status
// - get thumbnail -> update status
// - return with requeue of 10s

func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("server", req.NamespacedName)

	log.Info("reconciling...")

	var server minecraftv1.Server
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		log.Error(err, "unable to fetch Server")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("got server", "server", server.Name)

	defer func() {
		err := r.Status().Update(ctx, &server)
		if err != nil {
			log.Error(err, "failed to update server status field")
		}
	}()

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace), client.MatchingFields{serverOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list Pods for Server")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	log.Info(fmt.Sprintf("Got pods: %+v", pods))

	// delete extraneous pods, if they exist
	// if server is enabled, skip the first pod
	maxPods := 0
	if server.Spec.Enabled {
		maxPods = 1
	}
	for podIndex, pod := range pods.Items {
		if podIndex >= maxPods {
			if err := r.Delete(ctx, &pod, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete extraneous pod", "pod", pod)
			} else {
				log.V(0).Info("deleted extraneous pod", "pod", pod)
			}
		}
	}

	if !server.Spec.Enabled {
		return ctrl.Result{}, nil
	}

	server.Status.Running = false // default

	if server.Spec.Enabled && len(pods.Items) == 0 {
		log.Info("missing pod for server, creating manifests...")
		err := r.createManifests(ctx, &server)
		if err != nil {
			log.Error(err, "failed to create manifests for server")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}

		log.Info("created manifests for minecraft server")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// update the pod's status
	serverPod := pods.Items[0]
	for _, condition := range serverPod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			log.Info("updating server status with pod ready status", "status", condition.Status)
			server.Status.Running = condition.Status == corev1.ConditionTrue
		}
	}

	// TODO: update thumbnail
	// TODO: get players

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, serverOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Pod...
		if owner.APIVersion != apiGVStr || owner.Kind != "Server" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&minecraftv1.Server{}).
		Complete(r)
}
