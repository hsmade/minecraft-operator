package controllers

import (
	"context"
	"github.com/go-logr/logr"
	minecraftv1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/loglevels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	Config struct {
		ModJarsPVC         *corev1.PersistentVolumeClaim
		ServerJarsPVC      *corev1.PersistentVolumeClaim
		ServerPV           *corev1.PersistentVolume
		InitContainerImage string
	}
)

// OperatorConfigReconciler reconciles a OperatorConfig object
type OperatorConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=OperatorConfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=OperatorConfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=minecraft.hsmade.com,resources=OperatorConfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
//
// This reconciler finds the referenced objects and stores them in a global var
func (r *OperatorConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("OperatorConfig", req.NamespacedName)
	log.V(loglevels.Verbose).Info("start reconciling loop")

	log.V(loglevels.Flow).Info("fetching OperatorConfig manifest")
	var OperatorConfig minecraftv1.OperatorConfig
	if err := r.Get(ctx, req.NamespacedName, &OperatorConfig); err != nil {
		log.Error(err, "ERROR unable to fetch OperatorConfig, ending reconcile loop")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(loglevels.Flow).Info("fetched OperatorConfig manifest ok")
	log.V(loglevels.Trace).Info("got OperatorConfig manifest", "OperatorConfig", OperatorConfig)

	var serverJarsPVC corev1.PersistentVolumeClaim
	log.V(loglevels.Flow).Info("Looking for Server Jar PVC", "pvc-name", OperatorConfig.Spec.ServerJarsPVC)
	if err := r.Get(ctx, client.ObjectKey{Name: OperatorConfig.Spec.ServerJarsPVC, Namespace: OperatorConfig.Namespace}, &serverJarsPVC); err != nil {
		log.V(loglevels.Flow).Error(err, "failed to find Server Jar PVC", "name", OperatorConfig.Spec.ServerJarsPVC, "namespace", OperatorConfig.Namespace)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	Config.ServerJarsPVC = &serverJarsPVC
	log.V(loglevels.Verbose).Info("found Server Jar PVC")

	var modJarsPVC corev1.PersistentVolumeClaim
	log.V(loglevels.Flow).Info("Looking for Mod Jars PVC", "pvc-name", OperatorConfig.Spec.ModJarsPVC)
	if err := r.Get(ctx, client.ObjectKey{Name: OperatorConfig.Spec.ModJarsPVC, Namespace: OperatorConfig.Namespace}, &modJarsPVC); err != nil {
		log.V(loglevels.Flow).Error(err, "failed to find Mod Jars PVC", "pvc-name", OperatorConfig.Spec.ModJarsPVC)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	Config.ModJarsPVC = &modJarsPVC
	log.V(loglevels.Verbose).Info("found Mod Jars PVC")

	Config.ServerPV = OperatorConfig.Spec.ServersPV

	Config.InitContainerImage = OperatorConfig.Spec.InitContainerImage
	if Config.InitContainerImage == "" {
		Config.InitContainerImage = "busybox"
	}
	log.V(loglevels.Verbose).Info("init container image set to " + Config.InitContainerImage)

	// return for requeue
	log.V(loglevels.Flow).Info("Reconcile done")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&minecraftv1.OperatorConfig{}).
		Complete(r)
}
