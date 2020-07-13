package network

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/locks"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

var SubnetResourceName = "azurerm_subnet"

func resourceArmSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSubnetCreateUpdate,
		Read:   resourceArmSubnetRead,
		Update: resourceArmSubnetCreateUpdate,
		Delete: resourceArmSubnetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"virtual_network_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_prefix": {
				Type:     schema.TypeString,
				Required: true,
			},

			"network_security_group_id": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use the `azurerm_subnet_network_security_group_association` resource instead.",
			},

			"route_table_id": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use the `azurerm_subnet_route_table_association` resource instead.",
			},

			"ip_configurations": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"service_endpoints": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"delegation": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"service_delegation": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											"Microsoft.BareMetal/AzureVMware",
											"Microsoft.BareMetal/CrayServers",
											"Microsoft.Batch/batchAccounts",
											"Microsoft.ContainerInstance/containerGroups",
											"Microsoft.Databricks/workspaces",
											"Microsoft.DBforPostgreSQL/serversv2",
											"Microsoft.HardwareSecurityModules/dedicatedHSMs",
											"Microsoft.Logic/integrationServiceEnvironments",
											"Microsoft.Netapp/volumes",
											"Microsoft.ServiceFabricMesh/networks",
											"Microsoft.Sql/managedInstances",
											"Microsoft.Sql/servers",
											"Microsoft.StreamAnalytics/streamingJobs",
											"Microsoft.Web/hostingEnvironments",
											"Microsoft.Web/serverFarms",
										}, false),
									},
									"actions": {
										Type:       schema.TypeList,
										Optional:   true,
										Computed:   true,
										ConfigMode: schema.SchemaConfigModeAttr,
										Elem: &schema.Schema{
											Type: schema.TypeString,
											ValidateFunc: validation.StringInSlice([]string{
												"Microsoft.Network/networkinterfaces/*",
												"Microsoft.Network/virtualNetworks/subnets/action",
												"Microsoft.Network/virtualNetworks/subnets/join/action",
												"Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action",
												"Microsoft.Network/virtualNetworks/subnets/unprepareNetworkPolicies/action",
											}, false),
										},
									},
								},
							},
						},
					},
				},
			},

			"enforce_private_link_endpoint_network_policies": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"enforce_private_link_service_network_policies": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceArmSubnetCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Azure ARM Subnet creation.")

	name := d.Get("name").(string)
	vnetName := d.Get("virtual_network_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, vnetName, name, "")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Subnet %q (Virtual Network %q / Resource Group %q): %s", name, vnetName, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_subnet", *existing.ID)
		}
	}

	addressPrefix := d.Get("address_prefix").(string)

	locks.ByName(vnetName, VirtualNetworkResourceName)
	defer locks.UnlockByName(vnetName, VirtualNetworkResourceName)

	properties := network.SubnetPropertiesFormat{
		AddressPrefix: &addressPrefix,
	}

	if v, ok := d.GetOk("enforce_private_link_service_network_policies"); ok {
		// To enable private endpoints you must disable the network policies for the
		// subnet because Network policies like network security groups are not
		// supported by private endpoints.
		if v.(bool) {
			properties.PrivateLinkServiceNetworkPolicies = utils.String("Disabled")
		}
	}

	if v, ok := d.GetOk("network_security_group_id"); ok {
		nsgId := v.(string)
		properties.NetworkSecurityGroup = &network.SecurityGroup{
			ID: &nsgId,
		}

		parsedNsgId, err := ParseNetworkSecurityGroupID(nsgId)
		if err != nil {
			return err
		}

		locks.ByName(parsedNsgId.Name, networkSecurityGroupResourceName)
		defer locks.UnlockByName(parsedNsgId.Name, networkSecurityGroupResourceName)
	} else {
		properties.NetworkSecurityGroup = nil
	}

	if v, ok := d.GetOk("route_table_id"); ok {
		rtId := v.(string)
		properties.RouteTable = &network.RouteTable{
			ID: &rtId,
		}

		parsedRouteTableId, err := ParseRouteTableID(rtId)
		if err != nil {
			return err
		}

		locks.ByName(parsedRouteTableId.Name, routeTableResourceName)
		defer locks.UnlockByName(parsedRouteTableId.Name, routeTableResourceName)
	} else {
		properties.RouteTable = nil
	}

	if v, ok := d.GetOk("enforce_private_link_endpoint_network_policies"); ok {
		// This is strange logic, but to get the schema to make sense for the end user
		// I exposed it with the same name that the Azure CLI does to be consistent
		// between the tool sets, which means true == Disabled.
		//
		// To enable private endpoints you must disable the network policies for the
		// subnet because Network policies like network security groups are not
		// supported by private endpoints.
		if v.(bool) {
			properties.PrivateEndpointNetworkPolicies = utils.String("Disabled")
		}
	}

	serviceEndpoints := expandSubnetServiceEndpoints(d)
	properties.ServiceEndpoints = &serviceEndpoints

	delegations := expandSubnetDelegation(d)
	properties.Delegations = &delegations

	subnet := network.Subnet{
		Name:                   &name,
		SubnetPropertiesFormat: &properties,
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, vnetName, name, subnet)
	if err != nil {
		return fmt.Errorf("Error Creating/Updating Subnet %q (Virtual Network %q / Resource Group %q): %+v", name, vnetName, resGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for completion of Subnet %q (Virtual Network %q / Resource Group %q): %+v", name, vnetName, resGroup, err)
	}

	read, err := client.Get(ctx, resGroup, vnetName, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read ID of Subnet %q (Virtual Network %q / Resource Group %q)", vnetName, name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmSubnetRead(d, meta)
}

func resourceArmSubnetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	vnetName := id.Path["virtualNetworks"]
	name := id.Path["subnets"]

	resp, err := client.Get(ctx, resGroup, vnetName, name, "")

	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Subnet %q: %+v", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("virtual_network_name", vnetName)

	if props := resp.SubnetPropertiesFormat; props != nil {
		d.Set("address_prefix", props.AddressPrefix)

		if p := props.PrivateLinkServiceNetworkPolicies; p != nil {
			// To enable private endpoints you must disable the network policies for the
			// subnet because Network policies like network security groups are not
			// supported by private endpoints.

			d.Set("enforce_private_link_service_network_policies", strings.EqualFold("Disabled", *p))
		}

		var securityGroupId *string
		if props.NetworkSecurityGroup != nil {
			securityGroupId = props.NetworkSecurityGroup.ID
		}
		d.Set("network_security_group_id", securityGroupId)

		var routeTableId string
		if props.RouteTable != nil && props.RouteTable.ID != nil {
			routeTableId = *props.RouteTable.ID
		}
		d.Set("route_table_id", routeTableId)

		ips := flattenSubnetIPConfigurations(props.IPConfigurations)
		if err := d.Set("ip_configurations", ips); err != nil {
			return err
		}

		serviceEndpoints := flattenSubnetServiceEndpoints(props.ServiceEndpoints)
		if err := d.Set("service_endpoints", serviceEndpoints); err != nil {
			return err
		}

		// This is strange logic, but to get the schema to make sense for the end user
		// I exposed it with the same name that the Azure CLI does to be consistent
		// between the tool sets, which means true == Disabled.
		//
		// To enable private endpoints you must disable the network policies for the
		// subnet because Network policies like network security groups are not
		// supported by private endpoints.
		if privateEndpointNetworkPolicies := props.PrivateEndpointNetworkPolicies; privateEndpointNetworkPolicies != nil {
			d.Set("enforce_private_link_endpoint_network_policies", *privateEndpointNetworkPolicies == "Disabled")
		}

		delegation := flattenSubnetDelegation(props.Delegations)
		if err := d.Set("delegation", delegation); err != nil {
			return fmt.Errorf("Error flattening `delegation`: %+v", err)
		}
	}

	return nil
}

func resourceArmSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.SubnetsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["subnets"]
	vnetName := id.Path["virtualNetworks"]

	if v, ok := d.GetOk("network_security_group_id"); ok {
		networkSecurityGroupId := v.(string)
		parsedNetworkSecurityGroupId, err2 := ParseNetworkSecurityGroupID(networkSecurityGroupId)
		if err2 != nil {
			return err2
		}

		locks.ByName(parsedNetworkSecurityGroupId.Name, networkSecurityGroupResourceName)
		defer locks.UnlockByName(parsedNetworkSecurityGroupId.Name, networkSecurityGroupResourceName)
	}

	if v, ok := d.GetOk("route_table_id"); ok {
		rtId := v.(string)
		parsedRouteTableId, err2 := ParseRouteTableID(rtId)
		if err2 != nil {
			return err2
		}

		locks.ByName(parsedRouteTableId.Name, routeTableResourceName)
		defer locks.UnlockByName(parsedRouteTableId.Name, routeTableResourceName)
	}

	locks.ByName(vnetName, VirtualNetworkResourceName)
	defer locks.UnlockByName(vnetName, VirtualNetworkResourceName)

	locks.ByName(name, SubnetResourceName)
	defer locks.UnlockByName(name, SubnetResourceName)

	future, err := client.Delete(ctx, resGroup, vnetName, name)
	if err != nil {
		return fmt.Errorf("Error deleting Subnet %q (Virtual Network %q / Resource Group %q): %+v", name, vnetName, resGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for completion for Subnet %q (Virtual Network %q / Resource Group %q): %+v", name, vnetName, resGroup, err)
	}

	return nil
}

func expandSubnetServiceEndpoints(d *schema.ResourceData) []network.ServiceEndpointPropertiesFormat {
	serviceEndpoints := d.Get("service_endpoints").([]interface{})
	endpoints := make([]network.ServiceEndpointPropertiesFormat, 0)

	for _, svcEndpointRaw := range serviceEndpoints {
		if svc, ok := svcEndpointRaw.(string); ok {
			endpoint := network.ServiceEndpointPropertiesFormat{
				Service: &svc,
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

func flattenSubnetServiceEndpoints(serviceEndpoints *[]network.ServiceEndpointPropertiesFormat) []string {
	endpoints := make([]string, 0)

	if serviceEndpoints == nil {
		return endpoints
	}

	for _, endpoint := range *serviceEndpoints {
		if endpoint.Service != nil {
			endpoints = append(endpoints, *endpoint.Service)
		}
	}

	return endpoints
}

func flattenSubnetIPConfigurations(ipConfigurations *[]network.IPConfiguration) []string {
	ips := make([]string, 0)

	if ipConfigurations != nil {
		for _, ip := range *ipConfigurations {
			ips = append(ips, *ip.ID)
		}
	}

	return ips
}

func expandSubnetDelegation(d *schema.ResourceData) []network.Delegation {
	delegations := d.Get("delegation").([]interface{})
	retDelegations := make([]network.Delegation, 0)

	for _, deleValue := range delegations {
		deleData := deleValue.(map[string]interface{})
		deleName := deleData["name"].(string)
		srvDelegations := deleData["service_delegation"].([]interface{})
		srvDelegation := srvDelegations[0].(map[string]interface{})
		srvName := srvDelegation["name"].(string)
		srvActions := srvDelegation["actions"].([]interface{})

		retSrvActions := make([]string, 0)
		for _, srvAction := range srvActions {
			srvActionData := srvAction.(string)
			retSrvActions = append(retSrvActions, srvActionData)
		}

		retDelegation := network.Delegation{
			Name: &deleName,
			ServiceDelegationPropertiesFormat: &network.ServiceDelegationPropertiesFormat{
				ServiceName: &srvName,
				Actions:     &retSrvActions,
			},
		}

		retDelegations = append(retDelegations, retDelegation)
	}

	return retDelegations
}

func flattenSubnetDelegation(delegations *[]network.Delegation) []interface{} {
	if delegations == nil {
		return []interface{}{}
	}

	retDeles := make([]interface{}, 0)

	for _, dele := range *delegations {
		retDele := make(map[string]interface{})
		if v := dele.Name; v != nil {
			retDele["name"] = *v
		}

		svcDeles := make([]interface{}, 0)
		svcDele := make(map[string]interface{})
		if props := dele.ServiceDelegationPropertiesFormat; props != nil {
			if v := props.ServiceName; v != nil {
				svcDele["name"] = *v
			}

			if v := props.Actions; v != nil {
				svcDele["actions"] = *v
			}
		}

		svcDeles = append(svcDeles, svcDele)

		retDele["service_delegation"] = svcDeles

		retDeles = append(retDeles, retDele)
	}

	return retDeles
}
