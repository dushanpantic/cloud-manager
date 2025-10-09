package client

// NOTE: This file introduces a client abstraction for Azure Managed Redis.
// It currently mirrors the existing Redis (Cache for Redis) client interface.
// If Azure Managed Redis uses a different SDK package (e.g. armredisenterprise),
// adjust the imports and method parameter types accordingly in a follow-up step.

import (
	"context"

	armredisenterprise "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redisenterprise/armredisenterprise/v2"
)

// ManagedRedisClient defines operations needed by the Managed Redis controller layer.
// Separated from RedisClient to allow evolution / divergence of features.
type ManagedRedisClient interface {
	CreateManagedRedis(ctx context.Context, resourceGroupName, clusterName string, parameters armredisenterprise.Cluster) error
	UpdateManagedRedis(ctx context.Context, resourceGroupName, clusterName string, parameters armredisenterprise.ClusterUpdate) error
	GetManagedRedis(ctx context.Context, resourceGroupName, clusterName string) (*armredisenterprise.Cluster, error)
	DeleteManagedRedis(ctx context.Context, resourceGroupName, clusterName string) error
	// Enterprise SKU often manages access differently; placeholder for future key retrieval if supported.
}

func NewManagedRedisClient(svc *armredisenterprise.Client) ManagedRedisClient {
	return &managedRedisClient{svc: svc}
}

var _ ManagedRedisClient = &managedRedisClient{}

type managedRedisClient struct {
	svc *armredisenterprise.Client
}

func (c *managedRedisClient) CreateManagedRedis(ctx context.Context, resourceGroupName, clusterName string, parameters armredisenterprise.Cluster) error {
	// Enterprise Managed Redis (Redis Enterprise) create operation
	poller, err := c.svc.BeginCreate(ctx, resourceGroupName, clusterName, parameters, nil)
	if err != nil {
		return err
	}
	// Non-blocking: controller action can requeue while long-running op completes; no final Wait here.
	_ = poller // intentionally ignoring for now; future enhancement may poll.
	return nil
}

func (c *managedRedisClient) GetManagedRedis(ctx context.Context, resourceGroupName, clusterName string) (*armredisenterprise.Cluster, error) {
	resp, err := c.svc.Get(ctx, resourceGroupName, clusterName, nil)
	if err != nil {
		return nil, err
	}
	return &resp.Cluster, nil
}

func (c *managedRedisClient) DeleteManagedRedis(ctx context.Context, resourceGroupName, clusterName string) error {
	_, err := c.svc.BeginDelete(ctx, resourceGroupName, clusterName, nil)
	return err
}

func (c *managedRedisClient) UpdateManagedRedis(ctx context.Context, resourceGroupName, clusterName string, parameters armredisenterprise.ClusterUpdate) error {
	poller, err := c.svc.BeginUpdate(ctx, resourceGroupName, clusterName, parameters, nil)
	if err != nil {
		return err
	}
	_ = poller
	return nil
}
