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
	"github.com/hsmade/minecraft-operator/loglevels"
	v1 "k8s.io/api/apps/v1"
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
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("server", req.NamespacedName)
	log.V(loglevels.Verbose).Info("start reconciling loop")

	var server minecraftv1.Server
	log.V(loglevels.Flow).Info("fetching Server manifest")
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		log.Error(err, "ERROR unable to fetch Server, ending reconcile loop")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(loglevels.Flow).Info("fetched Server manifest ok")
	log.V(loglevels.Trace).Info("got server manifest", "server", server)

	err := r.ReconcileConfigMap(ctx, log, &server)
	if err != nil {
		log.V(loglevels.Error).Error(err, "failed to reconcile configMap, retrying in 30s")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	err = r.ReconcileDeployment(ctx, log, &server)
	if err != nil {
		log.V(loglevels.Error).Error(err, "failed to reconcile Pod, retrying in 30s")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	err = r.ReconcileService(ctx, log, &server)
	if err != nil {
		log.V(loglevels.Error).Error(err, "failed to reconcile Service, retrying in 30s")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	err = r.UpdateStatus(ctx, log, &server) // triggers requeue
	if err != nil {
		log.V(loglevels.Error).Error(err, "failed to update Server status, retrying in 30s")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if server.Spec.Enabled && server.Spec.IdleTimeoutSeconds > 0 {
		log.V(loglevels.Flow).Info("checking for idle timeout")
		log.V(loglevels.Trace).Info("checking for idle timeout", "idleTime",
			server.Status.IdleTime, "IdleTimeoutSeconds", server.Spec.IdleTimeoutSeconds)
		if server.Status.IdleTime > 0 && time.Now().Unix()-server.Status.IdleTime > server.Spec.IdleTimeoutSeconds {
			log.V(loglevels.Info).Info("server idle timeout reached, shutting down Pod")
			log.V(loglevels.Verbose).Info("setting server enable to false")
			server.Spec.Enabled = false
			err = r.Client.Update(ctx, &server)
			if err != nil {
				log.V(loglevels.Error).Error(err, "failed to update Server, retrying in 30s")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
		}
	}

	// return for requeue
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err

	// ------ old code below

	//defer func() {
	//	log.Info("storing status in server manifest")
	//	err := r.Status().Update(ctx, &server)
	//	if err != nil {
	//		log.Info("ERROR failed to update server status field", "error", err)
	//	}
	//	log.Info("reconciliation done")
	//}()
	//
	//var pods corev1.PodList
	//if err := r.List(ctx, &pods, client.InNamespace(req.Namespace), client.MatchingFields{serverOwnerKey: req.Name}); err != nil {
	//	log.Info("ERROR unable to list Pods for Server", "error", err)
	//	return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	//}
	//
	//log.Info(fmt.Sprintf("found %d pods for server", len(pods.Items)))
	//
	//// delete extraneous pods, if they exist
	//// if server is enabled, skip the first pod
	//maxPods := 0
	//if server.Spec.Enabled {
	//	log.Info("Server enabled, expecting one pod")
	//	maxPods = 1
	//}
	//// maxPods = 0; podIndex = 0 -> close
	//// maxPods = 1; podIndex = 0 -> ok
	//// maxPods = 1; podIndex = 1 -> close
	//for podIndex, pod := range pods.Items {
	//	if maxPods > podIndex || maxPods == 0 {
	//		log.V(0).Info("deleting extraneous pod", "pod", pod.Name)
	//		if err := r.Delete(ctx, &pod, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
	//			log.Info("ERROR unable to delete extraneous pod", "pod", pod, "error", err)
	//		}
	//	}
	//}
	//
	//if !server.Spec.Enabled {
	//	return ctrl.Result{}, nil
	//}
	//
	//server.Status.Running = false // default
	//
	//if server.Spec.Enabled && len(pods.Items) == 0 {
	//	log.Info("missing pod for server, creating manifests...")
	//	err := r.createManifests(ctx, &server)
	//	if err != nil {
	//		log.Info("ERROR failed to create manifests for server", "error", err)
	//		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	//	}
	//
	//	log.Info("created manifests for minecraft server")
	//	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	//}
	//
	//// update the pod's status
	//serverPod := pods.Items[0]
	//for _, condition := range serverPod.Status.Conditions {
	//	if condition.Type == corev1.PodReady {
	//		log.Info("updating server status with pod ready status", "status", condition.Status)
	//		server.Status.Running = condition.Status == corev1.ConditionTrue
	//	}
	//}
	//
	//// the rest needs the server to be running
	//if !server.Status.Running {
	//	log.Info("Server not running, stopping reconcile...")
	//	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	//}
	//
	//log.Info("connecting to server to get info...")
	//conn, err := net.Dial("tcp", fmt.Sprintf("%s:25565", serverPod.Status.PodIP)) // FIXME: ? hard coded port
	//if err != nil {
	//	log.Info("ERROR unable to connect to the Server", "error", err)
	//}
	//
	//status, _, err := mcping.PingAndListConn(conn, 578)
	//if err != nil {
	//	log.Info("ERROR unable to ping the Server", "error", err)
	//	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	//}
	//log.Info("got pong status for server", "status", status)
	//
	//// update the last pong timestamp
	//server.Status.LastPong = time.Now().Unix()
	//
	//server.Status.Players = []string{}
	//for _, player := range status.Players.Sample {
	//	server.Status.Players = append(server.Status.Players, player.Name)
	//}
	//log.Info("players set", "players", server.Status.Players)
	//
	//icon, err := status.Favicon.ToPNG()
	//if err != nil {
	//	log.Info("ERROR unable to get thumbnail from the Server")
	//	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	//}
	//server.Status.Thumbnail = base64.StdEncoding.EncodeToString([]byte(icon))
	//
	//return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Deployment{}, serverOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		deployment := rawObj.(*v1.Deployment)
		owner := metav1.GetControllerOf(deployment)
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

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.ConfigMap{}, serverOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		cm := rawObj.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(cm)
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

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, serverOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		svc := rawObj.(*corev1.Service)
		owner := metav1.GetControllerOf(svc)
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
