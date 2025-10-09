package managedredisinstance

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	armredisenterprise "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise/v2"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"

	"github.com/kyma-project/cloud-manager/pkg/common/actions/focal"
	azureclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/client"
	azurecommon "github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/common"
	azureconfig "github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/config"
	azuremanagedredisclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/managedredisinstance/client"
)

type State struct {
	focal.State

	client         azuremanagedredisclient.Client
	provider       azureclient.ClientProvider[azuremanagedredisclient.Client]
	clientId       string
	clientSecret   string
	subscriptionId string
	tenantId       string

	ipRange *cloudcontrolv1beta1.IpRange

	resourceGroupName   string
	privateEndPoint     *armnetwork.PrivateEndpoint
	privateDnsZoneGroup *armnetwork.PrivateDNSZoneGroup
	azureRedisInstance  *armredisenterprise.Cluster
}

type StateFactory interface {
	NewState(ctx context.Context, focalState focal.State) (*State, error)
}

type stateFactory struct {
	clientProvider azureclient.ClientProvider[azuremanagedredisclient.Client]
}

func NewStateFactory(clientProvider azureclient.ClientProvider[azuremanagedredisclient.Client]) StateFactory {
	return &stateFactory{
		clientProvider: clientProvider,
	}
}

func (f *stateFactory) NewState(ctx context.Context, focalState focal.State) (*State, error) {

	clientId := azureconfig.AzureConfig.DefaultCreds.ClientId
	clientSecret := azureconfig.AzureConfig.DefaultCreds.ClientSecret
	subscriptionId := focalState.Scope().Spec.Scope.Azure.SubscriptionId
	tenantId := focalState.Scope().Spec.Scope.Azure.TenantId

	c, err := f.clientProvider(ctx, clientId, clientSecret, subscriptionId, tenantId)

	if err != nil {
		return nil, err
	}

	return newState(focalState, c, f.clientProvider, clientId, clientSecret, subscriptionId, tenantId), nil
}

func newState(focalState focal.State,
	client azuremanagedredisclient.Client,
	provider azureclient.ClientProvider[azuremanagedredisclient.Client],
	clientId string,
	clientSecret string,
	subscriptionId string,
	tenantId string) *State {
	return &State{
		State:          focalState,
		client:         client,
		provider:       provider,
		clientId:       clientId,
		clientSecret:   clientSecret,
		subscriptionId: subscriptionId,
		tenantId:       tenantId,

		resourceGroupName: azurecommon.AzureCloudManagerResourceGroupName(focalState.Scope().Spec.Scope.Azure.VpcNetwork),
	}
}

func (s *State) ObjAsManagedRedisInstance() *cloudcontrolv1beta1.AzureManagedRedisInstance {
	return s.Obj().(*cloudcontrolv1beta1.AzureManagedRedisInstance)
}

func (s *State) IpRange() *cloudcontrolv1beta1.IpRange {
	return s.ipRange
}

func (s *State) SetIpRange(r *cloudcontrolv1beta1.IpRange) {
	s.ipRange = r
}
