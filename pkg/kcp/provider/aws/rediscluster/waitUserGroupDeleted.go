package rediscluster

import (
	"context"
	"errors"
	"fmt"

	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	awsmeta "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/meta"
	"github.com/kyma-project/cloud-manager/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func waitUserGroupDeleted(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	if state.userGroup == nil {
		return nil, nil
	}

	cacheState := ptr.Deref(state.userGroup.Status, "")

	if cacheState != awsmeta.ElastiCache_UserGroup_DELETING {
		errorMsg := fmt.Sprintf("Error: unexpected aws elasticache user group state: %s", cacheState)
		logger.Error(errors.New(errorMsg), errorMsg)
		redisInstance := st.Obj().(*cloudcontrolv1beta1.RedisCluster)
		redisInstance.Status.State = cloudcontrolv1beta1.StateError
		return composed.UpdateStatus(redisInstance).
			SetExclusiveConditions(metav1.Condition{
				Type:    cloudcontrolv1beta1.ConditionTypeError,
				Status:  metav1.ConditionTrue,
				Reason:  cloudcontrolv1beta1.ReasonUnknown,
				Message: errorMsg,
			}).
			SuccessError(composed.StopAndForget).
			SuccessLogMsg(errorMsg).
			Run(ctx, st)
	}

	logger.Info("User group is still being deleted, requeueing with delay")
	return composed.StopWithRequeueDelay(util.Timing.T60000ms()), nil
}
