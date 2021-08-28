package controllers

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/go-logr/logr"
	minecraftv1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileDeployment make sure the deployment exists as it should.
// TODO:
// - readiness check
// - liveliness check
func (r *ServerReconciler) ReconcileDeployment(ctx context.Context, log logr.Logger, server *minecraftv1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of Deployment")

	log.V(loglevels.Flow).Info("render Deployment")
	deployment, err := r.RenderDeployment(log, server)
	if err != nil {
		return errors.Wrap(err, "rendering Deployment")
	}
	log.V(loglevels.Trace).Info("Deployment rendered", "Deployment", *deployment)
	log.V(loglevels.Flow).Info("rendered Deployment ok")

	log.V(loglevels.Flow).Info("fetching Deployment manifest")
	var DeploymentList appsv1.DeploymentList
	if err := r.List(ctx, &DeploymentList, client.InNamespace(server.Namespace),
		client.MatchingFields{serverOwnerKey: server.Name}); err != nil {
		return errors.Wrap(err, "failed to list Deployments")
	}

	if len(DeploymentList.Items) == 0 {
		log.V(loglevels.Info).Info("Deployment not found, creating new one")
		err = r.Client.Create(ctx, deployment)
		if err != nil {
			return errors.Wrap(err, "creating Deployment")
		}
		log.V(loglevels.Flow).Info("created Deployment ok")
		return nil
	}

	log.V(loglevels.Flow).Info("checking for extraneous Deployments")
	if len(DeploymentList.Items) > 1 {
		log.V(loglevels.Info).Info("found more than one Deployment, deleting the extra one(s)", "maps found", len(DeploymentList.Items))
		for index, p := range DeploymentList.Items {
			if index > 0 {
				log.V(loglevels.Flow).Info("deleting Deployment", "namespace", p.Namespace, "name", p.Name)
				log.V(loglevels.Trace).Info("deleting Deployment", "Deployment", p)
				err = r.Client.Delete(ctx, &p)
				if err != nil {
					// non-critical error
					log.V(loglevels.Info).Error(err, "failed to delete Deployment", "namespace", p.Namespace, "name", p.Name)
				}
			}
		}
	}

	// patch the Deployment, if needed
	// only check for things we can influence
	log.V(loglevels.Flow).Info("comparing Deployment data with the rendered data")
	if !reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Ports[0].HostPort, DeploymentList.Items[0].Spec.Template.Spec.Containers[0].Ports[0].HostPort) ||
		!reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Image, DeploymentList.Items[0].Spec.Template.Spec.Containers[0].Image) ||
		!reflect.DeepEqual(deployment.Spec.Replicas, DeploymentList.Items[0].Spec.Replicas) ||
		!reflect.DeepEqual(deployment.ObjectMeta.Annotations["configHash"], DeploymentList.Items[0].ObjectMeta.Annotations["configHash"]) ||
		!reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Command, DeploymentList.Items[0].Spec.Template.Spec.Containers[0].Command) {
		log.V(loglevels.Info).Info("replacing Deployment")
		log.V(loglevels.Trace).Info("replacing Deployment", "rendered", deployment.Spec, "found", DeploymentList.Items[0].Spec)
		DeploymentList.Items[0].Spec = deployment.Spec
		DeploymentList.Items[0].ObjectMeta.Annotations["configHash"] = deployment.ObjectMeta.Annotations["configHash"]
		err = r.Client.Update(ctx, &DeploymentList.Items[0])
		if err != nil {
			return errors.Wrap(err, "replacing Deployment")
		}
	}
	log.V(loglevels.Flow).Info("Deployment is already up to date")

	return nil
}

// RenderDeployment renders the Deployment used for the Server
func (r *ServerReconciler) RenderDeployment(log logr.Logger, server *minecraftv1.Server) (*appsv1.Deployment, error) {
	log.V(loglevels.Verbose).Info("rendering Deployment")

	if Config.ServerPV == nil || Config.ModJarsPVC == nil || Config.ServerJarsPVC == nil {
		return nil, errors.New("Operator config isn't initialised (yet)") // FIXME
	}

	var executeBit int32 = 0o777
	var replicas int32 = 0
	if server.Spec.Enabled {
		replicas = 1
	}

	log.V(loglevels.Flow).Info("generating hash of spec")
	configHash, err := hashstructure.Hash(server.Spec, hashstructure.FormatV2, nil)
	if err != nil {
		log.V(loglevels.Info).Info("failed to generate hash from spec", "error", err)
		configHash = 0
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
				// FIXME: need more
			},
			Annotations: make(map[string]string),
			Name:        server.Name,
			Namespace:   server.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
					// FIXME: need more
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
						// FIXME: need more
					},
					Annotations: map[string]string{
						"checksum/config": fmt.Sprintf("%d", configHash),
					},
					Name:      server.Name,
					Namespace: server.Namespace,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "world",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: server.Name,
								},
							},
						},
						{
							Name: "server-jars",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: Config.ServerJarsPVC.Name,
								},
							},
						},
						{
							Name: "mod-jars",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: Config.ModJarsPVC.Name,
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
									DefaultMode: &executeBit,
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "init",
							Image:   Config.InitContainerImage,
							Command: []string{"/init.sh"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "config",
									MountPath: "/config",
								},
								{
									Name:      "server-jars",
									MountPath: "/jars/server",
								},
								{
									Name:      "mod-jars",
									MountPath: "/jars/mods",
								},
								{
									Name:      "config",
									SubPath:   "init.sh",
									MountPath: "/init.sh",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "minecraft",
							Image: server.Spec.Image,
							Ports: []corev1.ContainerPort{{
								Name:          "tcp-minecraft",
								ContainerPort: 25565,
								HostPort:      server.Spec.HostPort,
							}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "world",
									MountPath: "/data/world",
									SubPath:   server.Name,
								},
							},
							Command: []string{
								"java",
								fmt.Sprintf("-Xmx%dM", server.Spec.MaxMemory),
								fmt.Sprintf("-Xms%dM", server.Spec.InitMemory),
								"-jar", "server.jar",
							},
							WorkingDir: "/data",
						},
					},
				},
			},
		},
	}

	log.V(loglevels.Flow).Info("rendered Deployment ok")

	log.V(loglevels.Verbose).Info("setting controller reference for Deployment")
	if err := ctrl.SetControllerReference(server, deployment, r.Scheme); err != nil {
		log.Info("ERROR failed to set owner reference", "error", err)
		return nil, err
	}
	log.V(loglevels.Flow).Info("set controller reference ok for Deployment")

	return deployment, nil
}
