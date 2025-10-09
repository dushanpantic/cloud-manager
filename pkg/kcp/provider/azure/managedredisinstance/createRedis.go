package managedredisinstance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armredisenterprise "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise/v2"
	"github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"github.com/kyma-project/cloud-manager/pkg/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createRedis(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	if state.azureRedisInstance != nil {
		return nil, ctx
	}

	logger.Info("Creating Azure Redis")
	resourceGroupName := state.resourceGroupName

	redisInstanceName := state.ObjAsManagedRedisInstance().Name
	err := state.client.CreateManagedRedis(
		ctx,
		resourceGroupName,
		redisInstanceName,
		getCreateParams(state),
	)

	if err != nil {
		logger.Error(err, "Error creating Azure Redis")
		meta.SetStatusCondition(state.ObjAsManagedRedisInstance().Conditions(), metav1.Condition{
			Type:    v1beta1.ConditionTypeError,
			Status:  "True",
			Reason:  v1beta1.ReasonFailedCreatingFileSystem,
			Message: fmt.Sprintf("Failed creating AzureRedis: %s", err),
		})
		err = state.UpdateObjStatus(ctx)
		if err != nil {
			return composed.LogErrorAndReturn(err,
				"Error updating RedisInstance status due failed azure redis creation",
				composed.StopWithRequeueDelay(util.Timing.T10000ms()),
				ctx,
			)
		}

		return composed.StopWithRequeueDelay(util.Timing.T10000ms()), nil
	}

	return composed.StopWithRequeueDelay(util.Timing.T60000ms()), nil
}

func getCreateParams(state *State) armredisenterprise.Cluster {
	skuName := getSKUName(state)
	createProperties := &armredisenterprise.ClusterProperties{
		// Add basic properties here - we'll investigate what fields are actually available
	}

	cluster := armredisenterprise.Cluster{
		Location:   to.Ptr(state.Scope().Spec.Region),
		Properties: createProperties,
		SKU: &armredisenterprise.SKU{
			Name:     skuName,
			Capacity: to.Ptr[int32](int32(state.ObjAsManagedRedisInstance().Spec.SKU.Capacity)),
		},
	}
	return cluster
}

func getSKUName(state *State) *armredisenterprise.SKUName {
	// For Redis Enterprise, we'll map based on the family
	if state.ObjAsManagedRedisInstance().Spec.SKU.Family == "E" {
		return to.Ptr(armredisenterprise.SKUNameEnterpriseE10)
	}
	// Default to Enterprise Flash if not specified
	return to.Ptr(armredisenterprise.SKUNameEnterpriseFlashF300)
}
