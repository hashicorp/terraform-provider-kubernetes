package client

import (
	"github.com/Azure/azure-sdk-for-go/services/netapp/mgmt/2019-06-01/netapp"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/common"
)

type Client struct {
	AccountClient  *netapp.AccountsClient
	PoolClient     *netapp.PoolsClient
	VolumeClient   *netapp.VolumesClient
	SnapshotClient *netapp.SnapshotsClient
}

func NewClient(o *common.ClientOptions) *Client {
	accountClient := netapp.NewAccountsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&accountClient.Client, o.ResourceManagerAuthorizer)

	poolClient := netapp.NewPoolsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&poolClient.Client, o.ResourceManagerAuthorizer)

	volumeClient := netapp.NewVolumesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&volumeClient.Client, o.ResourceManagerAuthorizer)

	snapshotClient := netapp.NewSnapshotsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&snapshotClient.Client, o.ResourceManagerAuthorizer)

	return &Client{
		AccountClient:  &accountClient,
		PoolClient:     &poolClient,
		VolumeClient:   &volumeClient,
		SnapshotClient: &snapshotClient,
	}
}
