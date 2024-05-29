package client

import (
	"context"
	"fmt"

	"github.com/kyma-project/cloud-manager/pkg/composed"

	redis "cloud.google.com/go/redis/apiv1"
	redispb "cloud.google.com/go/redis/apiv1/redispb"
	"github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/client"
	"google.golang.org/api/option"
)

type CreateRedisInstanceOptions struct {
	VPCNetworkName string
	IPRangeName    string
}

type MemorystoreClient interface {
	CreateRedisInstance(ctx context.Context, projectId, locationId, instanceId string, options CreateRedisInstanceOptions) (*redis.CreateInstanceOperation, error)
}

func NewMemorystoreClientProvider() client.ClientProvider[MemorystoreClient] {
	return client.NewCachedClientProvider(
		func(ctx context.Context, saJsonKeyPath string) (MemorystoreClient, error) {
			httpClient, err := client.GetCachedGcpClient(ctx, saJsonKeyPath)
			if err != nil {
				return nil, err
			}

			memorystoreClient, err := redis.NewCloudRedisClient(ctx, option.WithHTTPClient(httpClient))

			if err != nil {
				return nil, fmt.Errorf("error obtaining GCP Memorystore Client: [%w]", err)
			}
			return NewMemorystoreClient(memorystoreClient), nil
		},
	)
}

func NewMemorystoreClient(cloudRedisClient *redis.CloudRedisClient) MemorystoreClient {
	return &memorystoreClient{cloudRedisClient: cloudRedisClient}
}

type memorystoreClient struct {
	cloudRedisClient *redis.CloudRedisClient
}

func (memorystoreClient *memorystoreClient) CreateRedisInstance(ctx context.Context, projectId, locationId, instanceId string, options CreateRedisInstanceOptions) (*redis.CreateInstanceOperation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", projectId, locationId)
	req := &redispb.CreateInstanceRequest{
		Parent:     parent,
		InstanceId: instanceId,
		Instance: &redispb.Instance{
			Name:              fmt.Sprintf("%s/%s", parent, instanceId),
			LocationId:        locationId,
			MemorySizeGb:      4,
			Tier:              redispb.Instance_BASIC,
			ConnectMode:       redispb.Instance_PRIVATE_SERVICE_ACCESS, // always
			AuthorizedNetwork: options.VPCNetworkName,
			ReservedIpRange:   options.IPRangeName,
		},
	}

	operation, err := memorystoreClient.cloudRedisClient.CreateInstance(ctx, req)
	if err != nil {
		logger := composed.LoggerFromCtx(ctx)
		logger.Error(err, "CreateRedisInstance", "projectId", projectId, "locationId", locationId, "instanceId", instanceId)
		return nil, err
	}

	return operation, nil
}
