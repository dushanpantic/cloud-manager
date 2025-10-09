package managedredisinstance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armredisenterprise "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise/v2"
	"github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"github.com/kyma-project/cloud-manager/pkg/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func modifyRedis(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	requestedAzureRedisInstance := state.ObjAsManagedRedisInstance()

	if !meta.IsStatusConditionTrue(requestedAzureRedisInstance.Status.Conditions, cloudresourcesv1beta1.ConditionTypeReady) {
		return nil, ctx
	}

	if state.azureRedisInstance == nil {
		return nil, ctx
	}

	updateParams, capacityChanged := getUpdateParams(state)

	if !capacityChanged {
		return nil, ctx
	}

	resourceGroupName := state.resourceGroupName
	logger.Info("Detected modified Redis configuration")
	err := state.client.UpdateManagedRedis(
		ctx,
		resourceGroupName,
		requestedAzureRedisInstance.Name,
		updateParams,
	)

	if err != nil {
		logger.Error(err, "Error updating Azure Redis")
		meta.SetStatusCondition(state.ObjAsManagedRedisInstance().Conditions(), metav1.Condition{
			Type:    v1beta1.ConditionTypeError,
			Status:  "True",
			Reason:  v1beta1.ConditionTypeError,
			Message: fmt.Sprintf("Failed to modify AzureRedis: %s", err),
		})
		err = state.UpdateObjStatus(ctx)
		if err != nil {
			return composed.LogErrorAndReturn(err,
				"Error updating RedisInstance status due failed azure redis update",
				composed.StopWithRequeueDelay(util.Timing.T10000ms()),
				ctx,
			)
		}

		return composed.StopWithRequeueDelay(util.Timing.T10000ms()), nil
	}

	return composed.StopWithRequeueDelay(util.Timing.T1000ms()), nil
}

func getUpdateParams(state *State) (armredisenterprise.ClusterUpdate, bool) {

	requestedAzureRedisInstance := state.ObjAsManagedRedisInstance()
	capacityChanged := int(*state.azureRedisInstance.SKU.Capacity) != requestedAzureRedisInstance.Spec.SKU.Capacity
	updateParameters := armredisenterprise.ClusterUpdate{}

	if !capacityChanged {
		return updateParameters, false
	}

	updateProperties := armredisenterprise.ClusterUpdate{
		SKU: &armredisenterprise.SKU{
			Capacity: to.Ptr[int32](int32(state.ObjAsManagedRedisInstance().Spec.SKU.Capacity)),
		},
	}

	return updateProperties, capacityChanged
}
