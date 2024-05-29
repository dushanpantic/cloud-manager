package redisinstance

import (
	"context"
	"fmt"

	"github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/client"
	"github.com/kyma-project/cloud-manager/pkg/kcp/redisinstance/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/common/actions"
	"github.com/kyma-project/cloud-manager/pkg/composed"
)

func New(stateFactory StateFactory) composed.Action {
	return func(ctx context.Context, st composed.State) (error, context.Context) {

		state, err := stateFactory.NewState(ctx, st.(types.State))
		if err != nil {
			redisInstance := st.Obj().(*v1beta1.RedisInstance)
			return composed.UpdateStatus(redisInstance).
				SetExclusiveConditions(metav1.Condition{
					Type:    v1beta1.ConditionTypeError,
					Status:  metav1.ConditionTrue,
					Reason:  v1beta1.ReasonGcpError,
					Message: err.Error(),
				}).
				SuccessError(composed.StopAndForget).
				SuccessLogMsg(fmt.Sprintf("Error creating new GCP RedisInstance state: %s", err)).
				Run(ctx, st)
		}
		return composed.ComposeActions(
			"gcsRedisInstance",
			actions.AddFinalizer,
			createRedisInstance,
			composed.BuildBranchingAction("RunFinalizer", StatePredicate(client.Deleted, ctx, state),
				actions.RemoveFinalizer, nil),
		)(ctx, state)
	}
}

func StatePredicate(status v1beta1.StatusState, ctx context.Context, state *State) composed.Predicate {
	return func(ctx context.Context, st composed.State) bool {
		return status == state.curState
	}
}
