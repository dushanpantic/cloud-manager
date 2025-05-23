package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/kyma-project/cloud-manager/pkg/composed"
)

type VaultClient interface {
	CreateVault(ctx context.Context, resourceGroupName string, vaultName string, location string) (*string, error)
	DeleteVault(ctx context.Context, resourceGroupName string, vaultName string) error
	ListVaults(ctx context.Context) ([]*armrecoveryservices.Vault, error)
}

type vaultClient struct {
	azureClient *armrecoveryservices.VaultsClient
}

type CreateVaultResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func NewVaultClient(subscriptionId string, cred *azidentity.ClientSecretCredential) (VaultClient, error) {

	vc, err := armrecoveryservices.NewVaultsClient(subscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}

	return vaultClient{vc}, nil
}

// Returns operationId used to check the status
func (c vaultClient) CreateVault(ctx context.Context, resourceGroupName string, vaultName string, location string) (*string, error) {
	logger := composed.LoggerFromCtx(ctx).WithName("vaultClient - CreateVault")

	poller, err := c.azureClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		vaultName,
		armrecoveryservices.Vault{
			Location: to.Ptr(location),
			Properties: to.Ptr(armrecoveryservices.VaultProperties{
				PublicNetworkAccess: to.Ptr(armrecoveryservices.PublicNetworkAccessEnabled),
			}),
			SKU: to.Ptr(armrecoveryservices.SKU{
				Name: to.Ptr(armrecoveryservices.SKUNameStandard),
			}),
			Tags: map[string]*string{"cloud-manager": to.Ptr("rwxVolumeBackup")},
		},
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create vault: %w", err)
	}

	resp, err := poller.Poll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to poll the create operation: %w", err)
	}

	// Read resp body
	if resp == nil || resp.Body == nil {
		return nil, errors.New("response body is nil")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %w", err)
	}
	var data CreateVaultResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the response body: %w", err)
	}

	logger.Info("CreateVault response", "jobId", data.Id)

	return &data.Id, nil

}

func (c vaultClient) DeleteVault(ctx context.Context, resourceGroupName string, vaultName string) error {

	_, err := c.azureClient.Delete(
		ctx,
		resourceGroupName,
		vaultName,
		to.Ptr(armrecoveryservices.VaultsClientDeleteOptions{}),
	)

	if err != nil {
		return err
	}

	return nil
}

func (c vaultClient) ListVaults(ctx context.Context) ([]*armrecoveryservices.Vault, error) {

	pager := c.azureClient.NewListBySubscriptionIDPager(
		&armrecoveryservices.VaultsClientListBySubscriptionIDOptions{},
	)

	var vaults []*armrecoveryservices.Vault
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return vaults, err
		}

		vaults = append(vaults, page.Value...)

	}
	return vaults, nil

}
