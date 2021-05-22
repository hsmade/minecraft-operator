package controllers

import (
	"context"
	_ "embed"
	minecraftv1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/controllers/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

//go:embed assets/eula.txt
var eula string

//go:embed assets/server.properties.tmpl
var serverPropertiesTemplate string

//go:embed assets/init.sh.tmpl
var initScriptTemplate string

// createManifests creates all the needed manifests for the Server instance
func (r *ServerReconciler) createManifests(ctx context.Context, server *minecraftv1.Server) error {
	// TODO:
	// - jar download DONE
	// - mods download
	// - property file -> emptydir + init container for initial values DONE
	// - eula.txt DONE
	// - readyness check
	// - liveness check
	log := r.Log.WithValues("server", server.Name)

	err, serverProperties := helpers.RenderTemplate(serverPropertiesTemplate, server.Spec.Properties)
	if err != nil {
		log.Error(err, "server.properties template failed")
		return err
	}

	err, initScript := helpers.RenderTemplate(initScriptTemplate, server.Spec)
	if err != nil {
		log.Error(err, "init.sh template failed")
		return err
	}

	serverConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        server.Name,
			Namespace:   server.Namespace,
		},
		Data: map[string]string{
			"eula.txt":          eula,
			"server.properties": serverProperties,
			"init.sh":           initScript,
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

	var executeBit int32 = 0o777

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
					Name: "world",
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
			Containers: []corev1.Container{{
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
					},
					{
						Name:      "config",
						MountPath: "/config",
					},
					{
						Name:      "config",
						SubPath:   "init.sh",
						MountPath: "/init.sh",
					},
				},
				Command:    []string{"/init.sh"},
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
