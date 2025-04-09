package scope

import (
	"context"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	awsclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/client"
	gcpclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/client"
	scopeclient "github.com/kyma-project/cloud-manager/pkg/kcp/scope/client"
	skrruntime "github.com/kyma-project/cloud-manager/pkg/skr/runtime"
	"github.com/kyma-project/cloud-manager/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ScopeReconciler interface {
	reconcile.Reconciler
}

func New(
	mgr manager.Manager,
	awsStsClientProvider awsclient.GardenClientProvider[scopeclient.AwsStsClient],
	activeSkrCollection skrruntime.ActiveSkrCollection,
	gcpServiceUsageClientProvider gcpclient.ClientProvider[gcpclient.ServiceUsageClient],
) ScopeReconciler {
	return NewScopeReconciler(
		NewStateFactory(
			composed.NewStateFactory(composed.NewStateClusterFromCluster(mgr)),
			activeSkrCollection,
			awsStsClientProvider,
			gcpServiceUsageClientProvider,
		),
	)
}

func NewScopeReconciler(stateFactory StateFactory) ScopeReconciler {
	return &scopeReconciler{
		stateFactory: stateFactory,
	}
}

type scopeReconciler struct {
	stateFactory StateFactory
}

func (r *scopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if Ignore != nil && Ignore.ShouldIgnoreKey(req) {
		return ctrl.Result{}, nil
	}

	state := r.newState(req)
	action := r.newAction()

	// Scope reconciler is triggered very often due to KLM constant changes on watched Kyma
	// HandleWithoutLogging should be used, so no reconciliation outcome is logged since it most cases
	// the reconciler will do nothing since no change regarding CloudManager was done on Kyma
	// so it will just produce an unnecessary log entry "Reconciliation finished without control error - doing stop and forget"
	// To accommodate this non-functional requirement to keep logs tidy and prevent excessive and not so usable log entries
	// in cases when Scope actually did something we have to accept the discomfort of not having this log entry
	return composed.Handling().
		WithMetrics("scope", util.RequestObjToString(req)).
		Handle(action(ctx, state))
}

func (r *scopeReconciler) newState(req ctrl.Request) *State {
	return r.stateFactory.NewState(req)
}

func (r *scopeReconciler) newAction() composed.Action {
	return composed.ComposeActionsNoName(
		composed.LoadObjNoStopIfNotFound, // loads Scope
		providerFromScopeToState,
		gardenerClusterLoad,
		networksLoad,
		gardenerClusterExtractShootName,
		logScope,

		composed.IfElse(
			shouldScopeExist,

			composed.ComposeActionsNoName(
				// scope should EXIST
				composed.If(
					isScopeCreateOrUpdateNeeded,
					gardenerClientCreate,
					shootNameMustHave,
					shootLoad,
					gardenerCredentialsLoad,
					scopeCreate,
					scopeEnsureCommonFields,
					scopeSave,
				),
				networkReferenceKymaCreate,
				apiEnable,
				conditionReady,
			),

			composed.ComposeActionsNoName(
				// scope should NOT exist

				// just in case stop the SKR Looper,
				// but kyma should not exist at this point
				// and should have been already removed
				skrDeactivate,
				networkReferenceKymaDelete,
				scopeDelete,
				nukeCreate,
				composed.StopAndForgetAction,
			),
		),
	)
}

func shouldScopeExist(_ context.Context, st composed.State) bool {
	state := st.(*State)

	if state.gardenerCluster == nil {
		return false
	}
	if composed.IsMarkedForDeletion(state.gardenerCluster) {
		return false
	}

	return true
}

func isScopeCreateOrUpdateNeeded(ctx context.Context, st composed.State) bool {
	state := st.(*State)

	if !composed.IsObjLoaded(ctx, st) {
		// scope does not exist
		return true
	}

	// check if labels from GardenerCluster are copied to Scope
	for _, label := range cloudcontrolv1beta1.ScopeLabels {
		if _, ok := state.ObjAsScope().Labels[label]; !ok {
			return true
		}
	}

	// check if GCP scope needs to be updated with worker info from shoot
	if state.ObjAsScope().Spec.Provider == cloudcontrolv1beta1.ProviderGCP {
		if len(state.ObjAsScope().Spec.Scope.Gcp.Workers) == 0 {
			return true
		}
	}

	// check if Azure scope needs to be updated with nodes info from shoot
	if state.ObjAsScope().Spec.Provider == cloudcontrolv1beta1.ProviderAzure {
		if state.ObjAsScope().Spec.Scope.Azure.Network.Nodes == "" {
			return true
		}
	}

	return false
}
