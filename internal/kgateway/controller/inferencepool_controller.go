package controller

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	infextv1a2 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/deployer"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
)

type inferencePoolReconciler struct {
	cli            client.Client
	scheme         *runtime.Scheme
	controllerName string
	deployer       *deployer.Deployer
}

func (r *inferencePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("inferencepool", req.NamespacedName)
	log.Info("Reconciling request", "name", req.Name, "namespace", req.Namespace)

	pool := &infextv1a2.InferencePool{}
	err := r.cli.Get(ctx, req.NamespacedName, pool)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The InferencePool is deleted, clean-up the endpoint picker k8s resources.
	if pool.GetDeletionTimestamp() != nil {
		log.Info("Deleting cluster-scoped resources for InferencePool.")
		// Clean up cluster-scoped resources.
		if err := r.deployer.CleanupClusterScopedResources(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
	}

	// List HTTPRoutes referencing this InferencePool.
	var routeList gwv1.HTTPRouteList
	if err := r.cli.List(ctx, &routeList,
		client.InNamespace(pool.Namespace),
		client.MatchingFields{InferencePoolField: pool.Name},
	); err != nil {
		log.Error(err, "failed to list HTTPRoutes referencing InferencePool")
		return ctrl.Result{}, err
	}

	if len(routeList.Items) == 0 {
		log.Info("No HTTPRoutes reference this InferencePool; deleting cluster-scoped resources if needed.")
		if err := r.deployer.CleanupClusterScopedResources(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
		// TODO [danehans]: Remove status for the matching kgateway parentRef.
		return ctrl.Result{}, nil
	}

	// Compute a new single PoolStatus for the controller, if any.
	newStatus, err := r.buildStatus(routeList.Items, pool)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Update pool.Status.Parents only if needed.
	changed, updatedParents := updatePoolStatus(pool.Status.Parents, newStatus)
	if changed {
		pool.Status.Parents = updatedParents
		if err := r.cli.Status().Update(ctx, pool); err != nil {
			log.Error(err, "failed to update InferencePool status")
			return ctrl.Result{}, err
		}
		log.Info("Updated InferencePool status")
	} else {
		log.Info("No status change needed for InferencePool")
	}

	// Deploy the k8s resources for the InferencePool.
	objs, err := r.deployer.GetEndpointPickerObjs(pool)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.deployer.DeployObjs(ctx, objs); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Reconcile complete", "req", req)
	return ctrl.Result{}, nil
}

// buildStatus returns a single PoolStatus if any HTTPRoute is managed by our controllerName
// and references the InferencePool. If no references are found for our controller, nil is returned.
func (r *inferencePoolReconciler) buildStatus(routes []gwv1.HTTPRoute, pool *infextv1a2.InferencePool) (*infextv1a2.PoolStatus, error) {
	for _, route := range routes {
		for _, p := range route.Status.Parents {
			if p.ControllerName == gwv1.GatewayController(r.controllerName) {
				ns := route.Namespace
				if p.ParentRef.Namespace != nil && *p.ParentRef.Namespace != "" {
					ns = string(*p.ParentRef.Namespace)
				}
				gwName := string(p.ParentRef.Name)

				return &infextv1a2.PoolStatus{
					GatewayRef: corev1.ObjectReference{
						APIVersion: gwv1.GroupVersion.String(),
						Kind:       wellknown.GatewayKind,
						Namespace:  ns,
						Name:       gwName,
					},
					Conditions: []metav1.Condition{
						{
							Type:               string(infextv1a2.InferencePoolConditionAccepted),
							Status:             metav1.ConditionTrue,
							Reason:             string(infextv1a2.InferencePoolReasonAccepted),
							Message:            "Referenced by an HTTPRoute accepted by the parentRef Gateway",
							ObservedGeneration: pool.Generation,
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				}, nil
			}
		}
	}

	// No references to our controllerName found.
	return nil, nil
}

// updatePoolStatus compares the newly computed PoolStatus to the existing pool status entries.
// If they differ, current is modified accordingly and returns. If no change is needed, false and
// current (unmodified) are returned.
func updatePoolStatus(current []infextv1a2.PoolStatus, newStatus *infextv1a2.PoolStatus) (bool, []infextv1a2.PoolStatus) {
	if newStatus == nil {
		return false, current
	}

	// Find a matching GatewayRef.
	foundIndex := -1
	for i, ps := range current {
		if ps.GatewayRef.Kind == newStatus.GatewayRef.Kind &&
			ps.GatewayRef.Namespace == newStatus.GatewayRef.Namespace &&
			ps.GatewayRef.Name == newStatus.GatewayRef.Name {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		// No entry for this GatewayRef, so append the newly created one.
		return true, append(current, *newStatus)
	}

	// Fields to ignrore when performing a status condition comparison.
	cmpOptions := []cmp.Option{
		cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime", "ObservedGeneration"),
	}

	// If found, compare them and only replace if they differ (aside from ignored fields).
	if !cmp.Equal(current[foundIndex], *newStatus, cmpOptions...) {
		updated := make([]infextv1a2.PoolStatus, len(current))
		copy(updated, current)
		updated[foundIndex] = *newStatus
		return true, updated
	}

	// Statuses are the same, so no change.
	return false, current
}
