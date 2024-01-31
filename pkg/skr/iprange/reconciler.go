package iprange

import (
	"context"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	skrruntime "github.com/kyma-project/cloud-manager/pkg/skr/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewReconcilerFactory() skrruntime.ReconcilerFactory {
	return &reconcilerFactory{}
}

type reconcilerFactory struct {
}

func (f *reconcilerFactory) New(kymaRef klog.ObjectRef, kcpCluster cluster.Cluster, skrCluster cluster.Cluster) reconcile.Reconciler {
	return &reconciler{
		factory: newStateFactory(
			composed.NewStateFactory(composed.NewStateClusterFromCluster(skrCluster)),
			kymaRef,
			composed.NewStateClusterFromCluster(kcpCluster),
		),
	}
}

type reconciler struct {
	factory *stateFactory
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	state := r.factory.NewState(req)
	action := r.newAction()

	return composed.Handle(action(ctx, state))
}

func (r *reconciler) newState() *State {
	return nil
}

func (r *reconciler) newAction() composed.Action {
	return composed.ComposeActions(
		"crIpRangeMain",
		composed.LoadObj,
		validateCidr,
		copyCidrToStatus,
		preventCidrChange,
		addFinalizer,
		loadKcpIpRange,
		createKcpIpRange,
		deleteKcpIpRange,
		removeFinalizer,
		updateStatus,
		composed.StopAndForgetAction,
	)
}
