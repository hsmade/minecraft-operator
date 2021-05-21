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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Server object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
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

	defer r.Status().Update(ctx, &server)

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

// createManifests creates all the needed manifests for the Server instance
func (r *ServerReconciler) createManifests(ctx context.Context, server *minecraftv1.Server) error {
	// TODO:
	// - jar download in initcontainer
	// - mods download in initcontainer
	// - property file -> emptydir + init container for initial values
	// - eula.txt DONEq
	// - readyness check
	// - liveness check
	log := r.Log.WithValues("server", server.Name)

	serverConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        server.Name,
			Namespace:   server.Namespace,
		},
		Data: map[string]string{
			"eula.txt": "eula=true",
			"server.properties": `max-tick-time=60000
generator-settings=
force-gamemode=false
allow-nether=true
gamemode=0
enable-query=false
player-idle-timeout=0
difficulty=1
spawn-monsters=true
op-permission-level=4
pvp=true
snooper-enabled=true
level-type=DEFAULT
hardcore=false
enable-command-block=true
max-players=20
network-compression-threshold=256
resource-pack-sha1=
max-world-size=29999984
server-port=25571
server-ip=
spawn-npcs=true
allow-flight=true
level-name=world
view-distance=10
resource-pack=
spawn-animals=true
white-list=false
generate-structures=true
online-mode=true
max-build-height=256
level-seed=
prevent-proxy-connections=false
use-native-transport=true
motd=TNT
enable-rcon=false`,
		},
	}

	if err := ctrl.SetControllerReference(server, &serverConfigMap, r.Scheme); err != nil {
		log.Error(err, "failed to set owner reference")
		return err
	}

	if err := r.Create(ctx, &serverConfigMap); err != nil {
		log.Error(err, "unable to create configMap for minecraft Server", "configMap", serverConfigMap)
		return err // FIXME: ignore exists
	}

	dirType := corev1.HostPathDirectory
	serverPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        server.Name,
			Namespace:   server.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: server.Spec.HostPath,
							Type: &dirType,
						},
					},
				},
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: server.Name,
							},
						},
					},
				},
			},
			Containers: []corev1.Container{{
				Name:  "minecraft",
				Image: "adoptopenjdk:8-jre-hotspot",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
					},
					{
						Name:      "config",
						SubPath:   "eula.txt",
						MountPath: "/data/eula.txt",
					},
					{
						Name:      "config",
						SubPath:   "server.properties",
						MountPath: "/data/server.properties",
					},
				},
				Command:    []string{"java", "-jar", "server.jar"},
				WorkingDir: "/data",
			}},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	if err := ctrl.SetControllerReference(server, &serverPod, r.Scheme); err != nil {
		log.Error(err, "failed to set owner reference")
		return err
	}

	if err := r.Create(ctx, &serverPod); err != nil {
		log.Error(err, "unable to create Pod for minecraft Server", "pod", serverPod)
		return err
	}
	log.Info("created pod for minecraft server", "pod", serverPod)

	return nil
}
