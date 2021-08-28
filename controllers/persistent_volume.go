package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilePersistentVolume make sure the PV exists as it should.
func (r *ServerReconciler) ReconcilePersistentVolume(ctx context.Context, log logr.Logger, server *v1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of PersistentVolume")

	log.V(loglevels.Flow).Info("render PersistentVolume")
	PersistentVolume, err := r.RenderPersistentVolume(log, server)
	if err != nil {
		return errors.Wrap(err, "rendering PersistentVolume")
	}
	log.V(loglevels.Trace).Info("PersistentVolume rendered", "PersistentVolume", *PersistentVolume)
	log.V(loglevels.Flow).Info("rendered PersistentVolume ok")

	log.V(loglevels.Flow).Info("fetching existing PersistentVolume manifest")
	var existingPersistentVolume corev1.PersistentVolume
	if err := r.Get(ctx, client.ObjectKey{Name: server.Namespace + "-" + server.Name}, &existingPersistentVolume); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to list PersistentVolumes")
	}
	log.V(loglevels.Trace).Info("Existing PersistentVolume", "pv", existingPersistentVolume)

	if existingPersistentVolume.Name == "" {
		log.V(loglevels.Info).Info("PersistentVolume not found, creating new one")
		err = r.Client.Create(ctx, PersistentVolume)
		if err != nil {
			return errors.Wrap(err, "creating PersistentVolume")
		}
		log.V(loglevels.Flow).Info("created PersistentVolume ok")
		return nil
	}

	// patch the PersistentVolume, if needed
	log.V(loglevels.Flow).Info("comparing PersistentVolume data with the rendered data")
	if !reflect.DeepEqual(PersistentVolume.Spec, existingPersistentVolume.Spec) {
		log.V(loglevels.Info).Info("replacing PersistentVolume")
		log.V(loglevels.Trace).Info("replacing PersistentVolume", "rendered", PersistentVolume.Spec, "found", existingPersistentVolume.Spec)
		err = r.Client.Update(ctx, PersistentVolume)
		if err != nil {
			return errors.Wrap(err, "replacing PersistentVolume")
		}
	}
	log.V(loglevels.Flow).Info("PersistentVolume is already up to date")

	return nil
}

// RenderPersistentVolume renders the PersistentVolume used for the Server's Pod
func (r *ServerReconciler) RenderPersistentVolume(log logr.Logger, server *v1.Server) (*corev1.PersistentVolume, error) {
	log.V(loglevels.Verbose).Info("rendering PersistentVolume")

	if Config.ServerPV == nil {
		return nil, errors.New("Operator config isn't initialised (yet)") // FIXME
	}

	log.V(loglevels.Trace).Info("checking for Config.ServersPV", "value", Config.ServerPV)
	if Config.ServerPV == nil {
		return nil, errors.New("ServersPV is not set")
	}

	pv := Config.ServerPV.DeepCopy()
	pv.Name = server.Namespace + "-" + server.Name
	pv.Labels = map[string]string{
		"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
		// FIXME: need more
	}

	log.V(loglevels.Trace).Info("rendered pv", "pv", *pv)
	log.V(loglevels.Flow).Info("rendered PersistentVolume ok")

	// cluster-scoped resource must not have a namespace-scoped owner, owner's namespace minecraft
	//log.V(loglevels.Verbose).Info("setting controller reference for PersistentVolume")
	//if err := ctrl.SetControllerReference(server, pv, r.Scheme); err != nil {
	//	log.Info("ERROR failed to set owner reference", "error", err)
	//	return nil, err
	//}
	//log.V(loglevels.Flow).Info("set controller reference ok for PersistentVolume")

	return pv, nil
}
