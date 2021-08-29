package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FIXME: we can't update an existing PVC

// ReconcilePersistentVolumeClaim make sure the PVC exists as it should.
func (r *ServerReconciler) ReconcilePersistentVolumeClaim(ctx context.Context, log logr.Logger, server *v1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of PersistentVolumeClaim")

	log.V(loglevels.Flow).Info("render PersistentVolumeClaim")
	PersistentVolumeClaim, err := r.RenderPersistentVolumeClaim(log, server)
	if err != nil {
		return errors.Wrap(err, "rendering PersistentVolumeClaim")
	}
	log.V(loglevels.Trace).Info("PersistentVolumeClaim rendered", "PersistentVolumeClaim", *PersistentVolumeClaim)
	log.V(loglevels.Flow).Info("rendered PersistentVolumeClaim ok")

	log.V(loglevels.Flow).Info("fetching PersistentVolumeClaim manifest")
	var PersistentVolumeClaimsList corev1.PersistentVolumeClaimList
	if err := r.List(ctx, &PersistentVolumeClaimsList, client.InNamespace(server.Namespace),
		client.MatchingFields{serverOwnerKey: server.Name}); err != nil {
		return errors.Wrap(err, "failed to list PersistentVolumeClaims")
	}

	if len(PersistentVolumeClaimsList.Items) == 0 {
		log.V(loglevels.Info).Info("PersistentVolumeClaim not found, creating new one")
		log.V(loglevels.Trace).Info("List returned", "err", err, "PersistentVolumeClaimsList", PersistentVolumeClaimsList.Items)
		err = r.Client.Create(ctx, PersistentVolumeClaim)
		if err != nil {
			return errors.Wrap(err, "creating PersistentVolumeClaim")
		}
		log.V(loglevels.Flow).Info("created PersistentVolumeClaim ok")
		return nil
	}

	log.V(loglevels.Flow).Info("checking for extraneous PersistentVolumeClaims")
	if len(PersistentVolumeClaimsList.Items) > 1 {
		log.V(loglevels.Info).Info("found more than one PersistentVolumeClaim, deleting the extra one(s)", "maps found", len(PersistentVolumeClaimsList.Items))
		for index, cm := range PersistentVolumeClaimsList.Items {
			if index > 0 {
				log.V(loglevels.Flow).Info("deleting PersistentVolumeClaim", "namespace", cm.Namespace, "name", cm.Name)
				log.V(loglevels.Trace).Info("deleting PersistentVolumeClaim", "PersistentVolumeClaim", cm)
				err = r.Client.Delete(ctx, &cm)
				if err != nil {
					// non-critical error
					log.V(loglevels.Info).Error(err, "failed to delete PersistentVolumeClaim", "namespace", cm.Namespace, "name", cm.Name)
				}
			}
		}
	}

	// patch the PersistentVolumeClaim, if needed
	log.V(loglevels.Flow).Info("comparing PersistentVolumeClaim data with the rendered data")
	if !reflect.DeepEqual(PersistentVolumeClaim.Spec, PersistentVolumeClaimsList.Items[0].Spec) {
		log.V(loglevels.Info).Info("replacing PersistentVolumeClaim")
		log.V(loglevels.Trace).Info("replacing PersistentVolumeClaim", "rendered", PersistentVolumeClaim.Spec, "found", PersistentVolumeClaimsList.Items[0].Spec)
		err = r.Client.Update(ctx, PersistentVolumeClaim)
		if err != nil {
			return errors.Wrap(err, "replacing PersistentVolumeClaim")
		}
	}
	log.V(loglevels.Flow).Info("PersistentVolumeClaim is already up to date")

	return nil
}

// RenderPersistentVolumeClaim renders the PersistentVolumeClaim used for the Server's Pod
func (r *ServerReconciler) RenderPersistentVolumeClaim(log logr.Logger, server *v1.Server) (*corev1.PersistentVolumeClaim, error) {
	log.V(loglevels.Verbose).Info("rendering PersistentVolumeClaim")

	if Config.ServerPV == nil || Config.ModJarsPVC == nil || Config.ServerJarsPVC == nil {
		return nil, errors.New("Operator config isn't initialised (yet)") // FIXME
	}

	log.V(loglevels.Trace).Info("checking for Config.ServersPV", "value", Config.ServerPV)
	if Config.ServerPV == nil {
		return nil, errors.New("ServersPV is not set")
	}

	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: server.Namespace,
			Name:      server.Name,
			Labels: map[string]string{
				"app": fmt.Sprintf("minecraft-operator-server-%s", server.Name),
				// FIXME: need more
			},
			Annotations: make(map[string]string),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"), // FIXME no hard coding
				},
			},
			VolumeName:       server.Namespace + "-" + server.Name,
			StorageClassName: &Config.ServerPV.Spec.StorageClassName,
			VolumeMode:       Config.ServerPV.Spec.VolumeMode,
		},
	}

	log.V(loglevels.Trace).Info("rendered pvc", "pvc", pvc)
	log.V(loglevels.Flow).Info("rendered PersistentVolumeClaim ok")

	log.V(loglevels.Verbose).Info("setting controller reference for PersistentVolumeClaim")
	if err := ctrl.SetControllerReference(server, &pvc, r.Scheme); err != nil {
		log.Info("ERROR failed to set owner reference", "error", err)
		return nil, err
	}
	log.V(loglevels.Flow).Info("set controller reference ok for PersistentVolumeClaim")

	return &pvc, nil
}
