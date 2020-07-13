package client

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/common"
)

type Client struct {
	ApplicationGatewaysClient            *network.ApplicationGatewaysClient
	ApplicationSecurityGroupsClient      *network.ApplicationSecurityGroupsClient
	AzureFirewallsClient                 *network.AzureFirewallsClient
	BastionHostsClient                   *network.BastionHostsClient
	ConnectionMonitorsClient             *network.ConnectionMonitorsClient
	DDOSProtectionPlansClient            *network.DdosProtectionPlansClient
	ExpressRouteAuthsClient              *network.ExpressRouteCircuitAuthorizationsClient
	ExpressRouteCircuitsClient           *network.ExpressRouteCircuitsClient
	ExpressRoutePeeringsClient           *network.ExpressRouteCircuitPeeringsClient
	InterfacesClient                     *network.InterfacesClient
	LoadBalancersClient                  *network.LoadBalancersClient
	LocalNetworkGatewaysClient           *network.LocalNetworkGatewaysClient
	PointToSiteVpnGatewaysClient         *network.P2sVpnGatewaysClient
	ProfileClient                        *network.ProfilesClient
	PacketCapturesClient                 *network.PacketCapturesClient
	PrivateEndpointClient                *network.PrivateEndpointsClient
	PublicIPsClient                      *network.PublicIPAddressesClient
	PublicIPPrefixesClient               *network.PublicIPPrefixesClient
	RoutesClient                         *network.RoutesClient
	RouteTablesClient                    *network.RouteTablesClient
	SecurityGroupClient                  *network.SecurityGroupsClient
	SecurityRuleClient                   *network.SecurityRulesClient
	SubnetsClient                        *network.SubnetsClient
	NatGatewayClient                     *network.NatGatewaysClient
	VnetGatewayConnectionsClient         *network.VirtualNetworkGatewayConnectionsClient
	VnetGatewayClient                    *network.VirtualNetworkGatewaysClient
	VnetClient                           *network.VirtualNetworksClient
	VnetPeeringsClient                   *network.VirtualNetworkPeeringsClient
	VirtualWanClient                     *network.VirtualWansClient
	VirtualHubClient                     *network.VirtualHubsClient
	VpnGatewaysClient                    *network.VpnGatewaysClient
	VpnServerConfigurationsClient        *network.VpnServerConfigurationsClient
	WatcherClient                        *network.WatchersClient
	WebApplicationFirewallPoliciesClient *network.WebApplicationFirewallPoliciesClient
	PrivateLinkServiceClient             *network.PrivateLinkServicesClient
}

func NewClient(o *common.ClientOptions) *Client {
	ApplicationGatewaysClient := network.NewApplicationGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ApplicationGatewaysClient.Client, o.ResourceManagerAuthorizer)

	ApplicationSecurityGroupsClient := network.NewApplicationSecurityGroupsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ApplicationSecurityGroupsClient.Client, o.ResourceManagerAuthorizer)

	AzureFirewallsClient := network.NewAzureFirewallsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&AzureFirewallsClient.Client, o.ResourceManagerAuthorizer)

	BastionHostsClient := network.NewBastionHostsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&BastionHostsClient.Client, o.ResourceManagerAuthorizer)

	ConnectionMonitorsClient := network.NewConnectionMonitorsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ConnectionMonitorsClient.Client, o.ResourceManagerAuthorizer)

	DDOSProtectionPlansClient := network.NewDdosProtectionPlansClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&DDOSProtectionPlansClient.Client, o.ResourceManagerAuthorizer)

	ExpressRouteAuthsClient := network.NewExpressRouteCircuitAuthorizationsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ExpressRouteAuthsClient.Client, o.ResourceManagerAuthorizer)

	ExpressRouteCircuitsClient := network.NewExpressRouteCircuitsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ExpressRouteCircuitsClient.Client, o.ResourceManagerAuthorizer)

	ExpressRoutePeeringsClient := network.NewExpressRouteCircuitPeeringsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ExpressRoutePeeringsClient.Client, o.ResourceManagerAuthorizer)

	InterfacesClient := network.NewInterfacesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&InterfacesClient.Client, o.ResourceManagerAuthorizer)

	LoadBalancersClient := network.NewLoadBalancersClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&LoadBalancersClient.Client, o.ResourceManagerAuthorizer)

	LocalNetworkGatewaysClient := network.NewLocalNetworkGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&LocalNetworkGatewaysClient.Client, o.ResourceManagerAuthorizer)

	pointToSiteVpnGatewaysClient := network.NewP2sVpnGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&pointToSiteVpnGatewaysClient.Client, o.ResourceManagerAuthorizer)

	vpnServerConfigurationsClient := network.NewVpnServerConfigurationsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&vpnServerConfigurationsClient.Client, o.ResourceManagerAuthorizer)

	ProfileClient := network.NewProfilesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ProfileClient.Client, o.ResourceManagerAuthorizer)

	VnetClient := network.NewVirtualNetworksClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VnetClient.Client, o.ResourceManagerAuthorizer)

	PacketCapturesClient := network.NewPacketCapturesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&PacketCapturesClient.Client, o.ResourceManagerAuthorizer)

	PrivateEndpointClient := network.NewPrivateEndpointsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&PrivateEndpointClient.Client, o.ResourceManagerAuthorizer)

	VnetPeeringsClient := network.NewVirtualNetworkPeeringsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VnetPeeringsClient.Client, o.ResourceManagerAuthorizer)

	PublicIPsClient := network.NewPublicIPAddressesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&PublicIPsClient.Client, o.ResourceManagerAuthorizer)

	PublicIPPrefixesClient := network.NewPublicIPPrefixesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&PublicIPPrefixesClient.Client, o.ResourceManagerAuthorizer)

	PrivateLinkServiceClient := network.NewPrivateLinkServicesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&PrivateLinkServiceClient.Client, o.ResourceManagerAuthorizer)

	RoutesClient := network.NewRoutesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&RoutesClient.Client, o.ResourceManagerAuthorizer)

	RouteTablesClient := network.NewRouteTablesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&RouteTablesClient.Client, o.ResourceManagerAuthorizer)

	SecurityGroupClient := network.NewSecurityGroupsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&SecurityGroupClient.Client, o.ResourceManagerAuthorizer)

	SecurityRuleClient := network.NewSecurityRulesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&SecurityRuleClient.Client, o.ResourceManagerAuthorizer)

	SubnetsClient := network.NewSubnetsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&SubnetsClient.Client, o.ResourceManagerAuthorizer)

	VnetGatewayClient := network.NewVirtualNetworkGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VnetGatewayClient.Client, o.ResourceManagerAuthorizer)

	NatGatewayClient := network.NewNatGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&NatGatewayClient.Client, o.ResourceManagerAuthorizer)

	VnetGatewayConnectionsClient := network.NewVirtualNetworkGatewayConnectionsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VnetGatewayConnectionsClient.Client, o.ResourceManagerAuthorizer)

	VirtualWanClient := network.NewVirtualWansClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VirtualWanClient.Client, o.ResourceManagerAuthorizer)

	VirtualHubClient := network.NewVirtualHubsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&VirtualHubClient.Client, o.ResourceManagerAuthorizer)

	vpnGatewaysClient := network.NewVpnGatewaysClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&vpnGatewaysClient.Client, o.ResourceManagerAuthorizer)

	WatcherClient := network.NewWatchersClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&WatcherClient.Client, o.ResourceManagerAuthorizer)

	WebApplicationFirewallPoliciesClient := network.NewWebApplicationFirewallPoliciesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&WebApplicationFirewallPoliciesClient.Client, o.ResourceManagerAuthorizer)

	return &Client{
		ApplicationGatewaysClient:            &ApplicationGatewaysClient,
		ApplicationSecurityGroupsClient:      &ApplicationSecurityGroupsClient,
		AzureFirewallsClient:                 &AzureFirewallsClient,
		BastionHostsClient:                   &BastionHostsClient,
		ConnectionMonitorsClient:             &ConnectionMonitorsClient,
		DDOSProtectionPlansClient:            &DDOSProtectionPlansClient,
		ExpressRouteAuthsClient:              &ExpressRouteAuthsClient,
		ExpressRouteCircuitsClient:           &ExpressRouteCircuitsClient,
		ExpressRoutePeeringsClient:           &ExpressRoutePeeringsClient,
		InterfacesClient:                     &InterfacesClient,
		LoadBalancersClient:                  &LoadBalancersClient,
		LocalNetworkGatewaysClient:           &LocalNetworkGatewaysClient,
		PointToSiteVpnGatewaysClient:         &pointToSiteVpnGatewaysClient,
		ProfileClient:                        &ProfileClient,
		PacketCapturesClient:                 &PacketCapturesClient,
		PrivateEndpointClient:                &PrivateEndpointClient,
		PublicIPsClient:                      &PublicIPsClient,
		PublicIPPrefixesClient:               &PublicIPPrefixesClient,
		RoutesClient:                         &RoutesClient,
		RouteTablesClient:                    &RouteTablesClient,
		SecurityGroupClient:                  &SecurityGroupClient,
		SecurityRuleClient:                   &SecurityRuleClient,
		SubnetsClient:                        &SubnetsClient,
		NatGatewayClient:                     &NatGatewayClient,
		VnetGatewayConnectionsClient:         &VnetGatewayConnectionsClient,
		VnetGatewayClient:                    &VnetGatewayClient,
		VnetClient:                           &VnetClient,
		VnetPeeringsClient:                   &VnetPeeringsClient,
		VirtualWanClient:                     &VirtualWanClient,
		VirtualHubClient:                     &VirtualHubClient,
		VpnGatewaysClient:                    &vpnGatewaysClient,
		VpnServerConfigurationsClient:        &vpnServerConfigurationsClient,
		WatcherClient:                        &WatcherClient,
		WebApplicationFirewallPoliciesClient: &WebApplicationFirewallPoliciesClient,
		PrivateLinkServiceClient:             &PrivateLinkServiceClient,
	}
}
