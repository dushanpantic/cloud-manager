package iprange

import (
	"context"
	iprange2 "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/iprange"
	"github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/iprange"
	iprange3 "github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/iprange"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/common/actions/focal"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type IPRangeReconciler interface {
	reconcile.Reconciler
}

type ipRangeReconciler struct {
	composedStateFactory composed.StateFactory
	focalStateFactory    focal.StateFactory

	awsStateFactory   iprange2.StateFactory
	azureStateFactory iprange.StateFactory
	gcpStateFactory   iprange3.StateFactory
}

func NewIPRangeReconciler(
	composedStateFactory composed.StateFactory,
	focalStateFactory focal.StateFactory,
	awsStateFactory iprange2.StateFactory,
	azureStateFactory iprange.StateFactory,
	gcpStateFactory iprange3.StateFactory,
) IPRangeReconciler {
	return &ipRangeReconciler{
		composedStateFactory: composedStateFactory,
		focalStateFactory:    focalStateFactory,
		awsStateFactory:      awsStateFactory,
		azureStateFactory:    azureStateFactory,
		gcpStateFactory:      gcpStateFactory,
	}
}

func (r *ipRangeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	state := r.newFocalState(req.NamespacedName)
	action := r.newAction()

	return composed.Handle(action(ctx, state))
}

func (r *ipRangeReconciler) newAction() composed.Action {
	return composed.ComposeActions(
		"main",
		focal.New(),
		func(ctx context.Context, st composed.State) (error, context.Context) {
			return composed.ComposeActions(
				"ipRangeCommon",
				// common IpRange common actions here
				// ... none so far
				// and now branch to provider specific flow
				composed.BuildSwitchAction(
					"providerSwitch",
					nil,
					composed.NewCase(focal.AwsProviderPredicate, iprange2.New(r.awsStateFactory)),
					composed.NewCase(focal.AzureProviderPredicate, iprange.New(r.azureStateFactory)),
					composed.NewCase(focal.GcpProviderPredicate, iprange3.New(r.gcpStateFactory)),
				),
			)(ctx, newState(st.(focal.State)))
		},
	)
}

func (r *ipRangeReconciler) newFocalState(name types.NamespacedName) focal.State {
	return r.focalStateFactory.NewState(
		r.composedStateFactory.NewState(name, &cloudresourcesv1beta1.IpRange{}),
	)
}