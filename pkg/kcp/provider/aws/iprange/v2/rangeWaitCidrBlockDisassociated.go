package v2

import (
	"context"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"github.com/kyma-project/cloud-manager/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func rangeWaitCidrBlockDisassociated(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	var theBlock *ec2types.VpcCidrBlockAssociation
	for _, cidrBlock := range state.vpc.CidrBlockAssociationSet {
		if ptr.Deref(cidrBlock.CidrBlock, "") == state.ObjAsIpRange().Status.Cidr {
			theBlock = &cidrBlock
		}
	}

	if theBlock == nil || theBlock.CidrBlockState == nil {
		return nil, nil
	}

	actMap := util.NewDelayActIgnoreBuilder[ec2types.VpcCidrBlockStateCode](util.Ignore).
		Delay(ec2types.VpcCidrBlockStateCodeDisassociating).
		Error(
			ec2types.VpcCidrBlockStateCodeFailing,
			ec2types.VpcCidrBlockStateCodeFailed,
			ec2types.VpcCidrBlockStateCodeAssociated,
			ec2types.VpcCidrBlockStateCodeAssociating,
		).
		Build()

	outcome := actMap.Case(theBlock.CidrBlockState.State)

	if outcome == util.Delay {
		logger.Info("Waiting for VPC Cidr block to get disassociated")
		return composed.StopWithRequeueDelay(util.Timing.T1000ms()), nil
	}

	if outcome == util.Ignore {
		// all fine, it's disassociated
		return nil, nil
	}

	// it's in the failing/failed state, report the error, and stop and forget

	return composed.PatchStatus(state.ObjAsIpRange()).
		SetExclusiveConditions(metav1.Condition{
			Type:    cloudcontrolv1beta1.ConditionTypeError,
			Status:  metav1.ConditionTrue,
			Reason:  cloudcontrolv1beta1.ReasonUnknown,
			Message: "VPC Cidr block is in Failed/Associated state",
		}).
		ErrorLogMessage("Failed patching KCP IpRange status with erroneous state of cidr block after deletion").
		SuccessLogMsg("Forgetting KCP IpRange with erroneous state of cidr block after deletion").
		SuccessError(composed.StopAndForget).
		Run(ctx, st)
}
