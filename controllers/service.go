package controllers

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/controllers/helpers"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileService make sure the service exists as it should.
func (r *ServerReconciler) ReconcileService(ctx context.Context, log logr.Logger, server *v1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of service")

	log.V(loglevels.Flow).Info("render service")
	service, err := r.RenderService(log, server)
	if err != nil {
		return errors.Wrap(err, "rendering service")
	}
	log.V(loglevels.Trace).Info("service rendered", "service", *service)
	log.V(loglevels.Flow).Info("rendered service ok")

	log.V(loglevels.Flow).Info("fetching service manifest")
	var servicesList corev1.ServiceList
	if err := r.List(ctx, &servicesList, client.InNamespace(server.Namespace),
		client.MatchingFields{serverOwnerKey: server.Name}); err != nil {
		return errors.Wrap(err, "failed to list services")
	}

	if len(servicesList.Items) == 0 {
		log.V(loglevels.Info).Info("service not found, creating new one")
		err = r.Client.Create(ctx, service)
		if err != nil {
			return errors.Wrap(err, "creating service")
		}
		log.V(loglevels.Flow).Info("created service ok")
		return nil
	}

	log.V(loglevels.Flow).Info("checking for extraneous services")
	if len(servicesList.Items) > 1 {
		log.V(loglevels.Info).Info("found more than one service, deleting the extra one(s)", "maps found", len(servicesList.Items))
		for index, cm := range servicesList.Items {
			if index > 0 {
				log.V(loglevels.Flow).Info("deleting service", "namespace", cm.Namespace, "name", cm.Name)
				log.V(loglevels.Trace).Info("deleting service", "service", cm)
				err = r.Client.Delete(ctx, &cm)
				if err != nil {
					// non-critical error
					log.V(loglevels.Info).Error(err, "failed to delete service", "namespace", cm.Namespace, "name", cm.Name)
				}
			}
		}
	}

	// we don't update the service, as we the user can't change anything in the Server Spec that changes here
	return nil
}

// RenderService renders the service used for the Server's Pod
func (r *ServerReconciler) RenderService(log logr.Logger, server *v1.Server) (*corev1.Service, error) {
	log.V(loglevels.Verbose).Info("rendering service")

	log.V(loglevels.Flow).Info("rendering server.properties")
	err, serverProperties := helpers.RenderTemplate(serverPropertiesTemplate, server.Spec.Properties)
	if err != nil {
		return nil, errors.Wrap(err, "rendering server.properties")
	}
	log.V(loglevels.Trace).Info("rendered server.properties", "result",
		serverProperties, "template", serverPropertiesTemplate, "data", server.Spec.Properties)
	log.V(loglevels.Flow).Info("rendered server.properties ok")

	log.V(loglevels.Flow).Info("rendering init.sh")
	err, initScript := helpers.RenderTemplate(initScriptTemplate, server.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "rendering init.sh")
	}
	log.V(loglevels.Trace).Info("rendered init.sh", "result",
		initScript, "template", initScriptTemplate, "data", server.Spec)
	log.V(loglevels.Flow).Info("rendered init.sh ok")

	log.V(loglevels.Flow).Info("rendering service")
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        server.Name,
			Namespace:   server.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name: "tcp-minecraft",
				Port: 25565,
			}},
			Selector: map[string]string{
				"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
				// FIXME: need more
			},
		},
	}
	log.V(loglevels.Flow).Info("rendered service ok")

	log.V(loglevels.Verbose).Info("setting controller reference for service")
	if err := ctrl.SetControllerReference(server, service, r.Scheme); err != nil {
		log.Info("ERROR failed to set owner reference", "error", err)
		return nil, err
	}
	log.V(loglevels.Flow).Info("set controller reference ok for service")

	return service, nil
}
