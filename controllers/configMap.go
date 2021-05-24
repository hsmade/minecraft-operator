package controllers

import (
	"context"
	_ "embed"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/controllers/helpers"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed assets/eula.txt
var eula string

//go:embed assets/server.properties.tmpl
var serverPropertiesTemplate string

//go:embed assets/init.sh.tmpl
var initScriptTemplate string

// ReconcileConfigMap make sure the config map exists as it should.
func (r *ServerReconciler) ReconcileConfigMap(ctx context.Context, log logr.Logger, server *v1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of configMap")

	log.V(loglevels.Flow).Info("render configMap")
	configMap, err := r.RenderConfigMap(log, server)
	if err != nil {
		return errors.Wrap(err, "rendering configMap")
	}
	log.V(loglevels.Trace).Info("configMap rendered", "configMap", *configMap)
	log.V(loglevels.Flow).Info("rendered configMap ok")

	log.V(loglevels.Flow).Info("fetching configMap manifest")
	var configMapsList corev1.ConfigMapList
	if err := r.List(ctx, &configMapsList, client.InNamespace(server.Namespace),
		client.MatchingFields{serverOwnerKey: server.Name}); err != nil {
		return errors.Wrap(err, "failed to list configMaps")
	}

	if len(configMapsList.Items) == 0 {
		log.V(loglevels.Info).Info("configMap not found, creating new one")
		log.V(loglevels.Trace).Info("List returned", "err", err, "configMapsList", configMapsList.Items)
		err = r.Client.Create(ctx, configMap)
		if err != nil {
			return errors.Wrap(err, "creating configMap")
		}
		log.V(loglevels.Flow).Info("created configMap ok")
		return nil
	}

	log.V(loglevels.Flow).Info("checking for extraneous configMaps")
	if len(configMapsList.Items) > 1 {
		log.V(loglevels.Info).Info("found more than one configMap, deleting the extra one(s)", "maps found", len(configMapsList.Items))
		for index, cm := range configMapsList.Items {
			if index > 0 {
				log.V(loglevels.Flow).Info("deleting configMap", "namespace", cm.Namespace, "name", cm.Name)
				log.V(loglevels.Trace).Info("deleting configMap", "configMap", cm)
				err = r.Client.Delete(ctx, &cm)
				if err != nil {
					// non-critical error
					log.V(loglevels.Info).Error(err, "failed to delete configMap", "namespace", cm.Namespace, "name", cm.Name)
				}
			}
		}
	}

	// patch the configMap, if needed
	log.V(loglevels.Flow).Info("comparing configMap data with the rendered data")
	if !reflect.DeepEqual(configMap.Data, configMapsList.Items[0].Data) {
		log.V(loglevels.Info).Info("replacing configMap")
		log.V(loglevels.Trace).Info("replacing configMap", "rendered", configMap.Data, "found", configMapsList.Items[0].Data)
		err = r.Client.Update(ctx, configMap)
		if err != nil {
			return errors.Wrap(err, "replacing configMap")
		}
	}
	log.V(loglevels.Flow).Info("configMap is already up to date")

	return nil
}

// RenderConfigMap renders the configMap used for the Server's Pod
func (r *ServerReconciler) RenderConfigMap(log logr.Logger, server *v1.Server) (*corev1.ConfigMap, error) {
	log.V(loglevels.Verbose).Info("rendering configMap")

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

	log.V(loglevels.Flow).Info("rendering configMap")
	configMap := &corev1.ConfigMap{
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
	log.V(loglevels.Flow).Info("rendered configMap ok")

	log.V(loglevels.Verbose).Info("setting controller reference for configMap")
	if err := ctrl.SetControllerReference(server, configMap, r.Scheme); err != nil {
		log.Info("ERROR failed to set owner reference", "error", err)
		return nil, err
	}
	log.V(loglevels.Flow).Info("set controller reference ok for configMap")

	return configMap, nil
}
