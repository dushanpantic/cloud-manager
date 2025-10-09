package client

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	armredisenterprise "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise/v2"
	azureclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/azure/client"
)

type Client interface {
	azureclient.ManagedRedisClient
	azureclient.PrivateEndPointsClient
	azureclient.PrivateDnsZoneGroupClient
}

func NewClientProvider() azureclient.ClientProvider[Client] {
	return func(ctx context.Context, clientId, clientSecret, subscriptionId, tenantId string, auxiliaryTenants ...string) (Client, error) {

		cred, err := azidentity.NewClientSecretCredential(tenantId, clientId, clientSecret, &azidentity.ClientSecretCredentialOptions{})

		if err != nil {
			return nil, err
		}

		armredisenterpriseClientInstance, err := armredisenterprise.NewClient(subscriptionId, cred, nil)
		if err != nil {
			return nil, err
		}

		privateEndPointsClient, err := armnetwork.NewPrivateEndpointsClient(subscriptionId, cred, nil)
		if err != nil {
			return nil, err
		}

		privateDnsZoneGroupClient, err := armnetwork.NewPrivateDNSZoneGroupsClient(subscriptionId, cred, nil)
		if err != nil {
			return nil, err
		}

		return newClient(
			azureclient.NewManagedRedisClient(armredisenterpriseClientInstance),
			azureclient.NewPrivateEndPointClient(privateEndPointsClient),
			azureclient.NewPrivateDnsZoneGroupClient(privateDnsZoneGroupClient),
		), nil
	}
}

type redisInstanceClient struct {
	azureclient.ManagedRedisClient
	azureclient.PrivateEndPointsClient
	azureclient.PrivateDnsZoneGroupClient
}

func newClient(managedRedisClient azureclient.ManagedRedisClient, privateEndPointsClient azureclient.PrivateEndPointsClient, privateDnsZoneGroupClient azureclient.PrivateDnsZoneGroupClient) Client {
	return &redisInstanceClient{
		ManagedRedisClient:        managedRedisClient,
		PrivateEndPointsClient:    privateEndPointsClient,
		PrivateDnsZoneGroupClient: privateDnsZoneGroupClient,
	}
}
