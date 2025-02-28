package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	infextv1a1 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/deployer"
)

type inferencePoolReconciler struct {
	cli      client.Client
	scheme   *runtime.Scheme
	deployer *deployer.Deployer
}

func (r *inferencePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("inferencepool", req.NamespacedName)
	log.V(1).Info("reconciling request", "request", req)

	pool := new(infextv1a1.InferencePool)
	if err := r.cli.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pool.GetDeletionTimestamp() != nil {
		// no need to do anything as we have owner refs, so children will be deleted
		log.Info("inferencepool deleted, no need for reconciling")
		return ctrl.Result{}, nil
	}

	// Use the registered index to list HTTPRoutes that reference this pool.
	var routeList gwv1.HTTPRouteList
	if err := r.cli.List(ctx, &routeList,
		client.InNamespace(pool.Namespace),
		client.MatchingFields{InferencePoolField: pool.Name},
	); err != nil {
		log.Error(err, "failed to list HTTPRoutes referencing InferencePool", "name", pool.Name, "namespace", pool.Namespace)
		return ctrl.Result{}, err
	}

	// If no HTTPRoutes reference the pool, skip reconciliation.
	if len(routeList.Items) == 0 {
		log.Info("No HTTPRoutes reference this InferencePool; skipping reconcile", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	objs, err := r.deployer.GetEndpointPickerObjs(pool)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO [danehans]: Manage inferencepool status conditions.

	// Deploy the endpoint picker resources.
	log.Info("Deploying endpoint picker for InferencePool", "name", pool.Name, "namespace", pool.Namespace)
	err = r.deployer.DeployObjs(ctx, objs)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("reconciled request", "request", req)

	return ctrl.Result{}, nil
}
