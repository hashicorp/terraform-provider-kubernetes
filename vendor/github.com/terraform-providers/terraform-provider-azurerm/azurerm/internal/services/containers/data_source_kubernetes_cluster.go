package containers

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-10-01/containerservice"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/kubernetes"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func dataSourceArmKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmKubernetesClusterRead,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": azure.SchemaResourceGroupNameForDataSource(),

			"location": azure.SchemaLocationForDataSource(),

			"addon_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"http_application_routing": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"http_application_routing_zone_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},

						"oms_agent": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"log_analytics_workspace_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},

						"kube_dashboard": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},

						"azure_policy": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"agent_pool_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"count": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"max_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"min_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"enable_auto_scaling": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"availability_zones": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						// TODO: remove this in a future version
						"dns_prefix": {
							Type:       schema.TypeString,
							Computed:   true,
							Deprecated: "This field is no longer returned from the Azure API",
						},

						"vm_size": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"os_disk_size_gb": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"vnet_subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"os_type": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"max_pods": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"node_taints": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"enable_node_public_ip": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"dns_prefix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"api_server_authorized_ip_ranges": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"private_link_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"private_fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"kubernetes_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"kube_admin_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"password": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"client_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"client_key": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"cluster_ca_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"kube_admin_config_raw": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"kube_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"password": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"client_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"client_key": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"cluster_ca_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"kube_config_raw": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"linux_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ssh_key": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key_data": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"windows_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"network_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_plugin": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"network_policy": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"service_cidr": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"dns_service_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"docker_bridge_cidr": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"pod_cidr": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"load_balancer_sku": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"node_resource_group": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"role_based_access_control": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"azure_active_directory": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"client_app_id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"server_app_id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"tenant_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"service_principal": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tags.SchemaDataSource(),
		},
	}
}

func dataSourceArmKubernetesClusterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Containers.KubernetesClustersClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	resp, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Error: Managed Kubernetes Cluster %q was not found in Resource Group %q", name, resourceGroup)
		}

		return fmt.Errorf("Error retrieving Managed Kubernetes Cluster %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	profile, err := client.GetAccessProfile(ctx, resourceGroup, name, "clusterUser")
	if err != nil {
		return fmt.Errorf("Error retrieving Access Profile for Managed Kubernetes Cluster %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.SetId(*resp.ID)

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if props := resp.ManagedClusterProperties; props != nil {
		d.Set("dns_prefix", props.DNSPrefix)
		d.Set("fqdn", props.Fqdn)
		d.Set("private_fqdn", props.PrivateFQDN)
		d.Set("kubernetes_version", props.KubernetesVersion)
		d.Set("node_resource_group", props.NodeResourceGroup)

		// TODO: 2.0 we should introduce a access_profile block to match the new API design,
		if accessProfile := props.APIServerAccessProfile; accessProfile != nil {
			apiServerAuthorizedIPRanges := utils.FlattenStringSlice(accessProfile.AuthorizedIPRanges)
			if err := d.Set("api_server_authorized_ip_ranges", apiServerAuthorizedIPRanges); err != nil {
				return fmt.Errorf("Error setting `api_server_authorized_ip_ranges`: %+v", err)
			}

			d.Set("private_link_enabled", accessProfile.EnablePrivateCluster)
		}

		addonProfiles := flattenKubernetesClusterDataSourceAddonProfiles(props.AddonProfiles)
		if err := d.Set("addon_profile", addonProfiles); err != nil {
			return fmt.Errorf("Error setting `addon_profile`: %+v", err)
		}

		agentPoolProfiles := flattenKubernetesClusterDataSourceAgentPoolProfiles(props.AgentPoolProfiles)
		if err := d.Set("agent_pool_profile", agentPoolProfiles); err != nil {
			return fmt.Errorf("Error setting `agent_pool_profile`: %+v", err)
		}

		linuxProfile := flattenKubernetesClusterDataSourceLinuxProfile(props.LinuxProfile)
		if err := d.Set("linux_profile", linuxProfile); err != nil {
			return fmt.Errorf("Error setting `linux_profile`: %+v", err)
		}

		windowsProfile := flattenKubernetesClusterDataSourceWindowsProfile(props.WindowsProfile)
		if err := d.Set("windows_profile", windowsProfile); err != nil {
			return fmt.Errorf("Error setting `windows_profile`: %+v", err)
		}

		networkProfile := flattenKubernetesClusterDataSourceNetworkProfile(props.NetworkProfile)
		if err := d.Set("network_profile", networkProfile); err != nil {
			return fmt.Errorf("Error setting `network_profile`: %+v", err)
		}

		roleBasedAccessControl := flattenKubernetesClusterDataSourceRoleBasedAccessControl(props)
		if err := d.Set("role_based_access_control", roleBasedAccessControl); err != nil {
			return fmt.Errorf("Error setting `role_based_access_control`: %+v", err)
		}

		servicePrincipal := flattenKubernetesClusterDataSourceServicePrincipalProfile(props.ServicePrincipalProfile)
		if err := d.Set("service_principal", servicePrincipal); err != nil {
			return fmt.Errorf("Error setting `service_principal`: %+v", err)
		}

		// adminProfile is only available for RBAC enabled clusters with AAD
		if props.AadProfile != nil {
			adminProfile, err := client.GetAccessProfile(ctx, resourceGroup, name, "clusterAdmin")
			if err != nil {
				return fmt.Errorf("Error retrieving Admin Access Profile for Managed Kubernetes Cluster %q (Resource Group %q): %+v", name, resourceGroup, err)
			}

			adminKubeConfigRaw, adminKubeConfig := flattenKubernetesClusterAccessProfile(adminProfile)
			d.Set("kube_admin_config_raw", adminKubeConfigRaw)
			if err := d.Set("kube_admin_config", adminKubeConfig); err != nil {
				return fmt.Errorf("Error setting `kube_admin_config`: %+v", err)
			}
		} else {
			d.Set("kube_admin_config_raw", "")
			d.Set("kube_admin_config", []interface{}{})
		}
	}

	kubeConfigRaw, kubeConfig := flattenKubernetesClusterDataSourceAccessProfile(profile)
	d.Set("kube_config_raw", kubeConfigRaw)
	if err := d.Set("kube_config", kubeConfig); err != nil {
		return fmt.Errorf("Error setting `kube_config`: %+v", err)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func flattenKubernetesClusterDataSourceRoleBasedAccessControl(input *containerservice.ManagedClusterProperties) []interface{} {
	rbacEnabled := false
	if input.EnableRBAC != nil {
		rbacEnabled = *input.EnableRBAC
	}

	results := make([]interface{}, 0)
	if profile := input.AadProfile; profile != nil {
		output := make(map[string]interface{})

		if profile.ClientAppID != nil {
			output["client_app_id"] = *profile.ClientAppID
		}

		if profile.ServerAppID != nil {
			output["server_app_id"] = *profile.ServerAppID
		}

		if profile.TenantID != nil {
			output["tenant_id"] = *profile.TenantID
		}

		results = append(results, output)
	}

	return []interface{}{
		map[string]interface{}{
			"enabled":                rbacEnabled,
			"azure_active_directory": results,
		},
	}
}

func flattenKubernetesClusterDataSourceAccessProfile(profile containerservice.ManagedClusterAccessProfile) (*string, []interface{}) {
	if profile.AccessProfile == nil {
		return nil, []interface{}{}
	}

	if kubeConfigRaw := profile.AccessProfile.KubeConfig; kubeConfigRaw != nil {
		rawConfig := string(*kubeConfigRaw)
		var flattenedKubeConfig []interface{}

		if strings.Contains(rawConfig, "apiserver-id:") {
			kubeConfigAAD, err := kubernetes.ParseKubeConfigAAD(rawConfig)

			if err != nil {
				return utils.String(rawConfig), []interface{}{}
			}

			flattenedKubeConfig = flattenKubernetesClusterDataSourceKubeConfigAAD(*kubeConfigAAD)
		} else {
			kubeConfig, err := kubernetes.ParseKubeConfig(rawConfig)

			if err != nil {
				return utils.String(rawConfig), []interface{}{}
			}

			flattenedKubeConfig = flattenKubernetesClusterDataSourceKubeConfig(*kubeConfig)
		}

		return utils.String(rawConfig), flattenedKubeConfig
	}

	return nil, []interface{}{}
}

func flattenKubernetesClusterDataSourceAddonProfiles(profile map[string]*containerservice.ManagedClusterAddonProfile) interface{} {
	values := make(map[string]interface{})

	routes := make([]interface{}, 0)
	if httpApplicationRouting := profile["httpApplicationRouting"]; httpApplicationRouting != nil {
		enabled := false
		if enabledVal := httpApplicationRouting.Enabled; enabledVal != nil {
			enabled = *enabledVal
		}

		zoneName := ""
		if v := httpApplicationRouting.Config["HTTPApplicationRoutingZoneName"]; v != nil {
			zoneName = *v
		}

		output := map[string]interface{}{
			"enabled":                            enabled,
			"http_application_routing_zone_name": zoneName,
		}
		routes = append(routes, output)
	}
	values["http_application_routing"] = routes

	agents := make([]interface{}, 0)
	if omsAgent := profile["omsagent"]; omsAgent != nil {
		enabled := false
		if enabledVal := omsAgent.Enabled; enabledVal != nil {
			enabled = *enabledVal
		}

		workspaceId := ""
		if workspaceResourceID := omsAgent.Config["logAnalyticsWorkspaceResourceID"]; workspaceResourceID != nil {
			workspaceId = *workspaceResourceID
		}

		output := map[string]interface{}{
			"enabled":                    enabled,
			"log_analytics_workspace_id": workspaceId,
		}
		agents = append(agents, output)
	}
	values["oms_agent"] = agents

	kubeDashboards := make([]interface{}, 0)
	if kubeDashboard := profile["kubeDashboard"]; kubeDashboard != nil {
		enabled := false
		if enabledVal := kubeDashboard.Enabled; enabledVal != nil {
			enabled = *enabledVal
		}

		output := map[string]interface{}{
			"enabled": enabled,
		}
		kubeDashboards = append(kubeDashboards, output)
	}
	values["kube_dashboard"] = kubeDashboards

	azurePolicies := make([]interface{}, 0)
	if azurePolicy := profile["azurepolicy"]; azurePolicy != nil {
		enabled := false
		if enabledVal := azurePolicy.Enabled; enabledVal != nil {
			enabled = *enabledVal
		}

		output := map[string]interface{}{
			"enabled": enabled,
		}
		azurePolicies = append(azurePolicies, output)
	}
	values["azure_policy"] = azurePolicies

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceAgentPoolProfiles(input *[]containerservice.ManagedClusterAgentPoolProfile) []interface{} {
	agentPoolProfiles := make([]interface{}, 0)

	if input == nil {
		return agentPoolProfiles
	}

	for _, profile := range *input {
		agentPoolProfile := make(map[string]interface{})

		if profile.Type != "" {
			agentPoolProfile["type"] = string(profile.Type)
		}

		if profile.Count != nil {
			agentPoolProfile["count"] = int(*profile.Count)
		}

		if profile.MinCount != nil {
			agentPoolProfile["min_count"] = int(*profile.MinCount)
		}

		if profile.MaxCount != nil {
			agentPoolProfile["max_count"] = int(*profile.MaxCount)
		}

		if profile.EnableAutoScaling != nil {
			agentPoolProfile["enable_auto_scaling"] = *profile.EnableAutoScaling
		}

		agentPoolProfile["availability_zones"] = utils.FlattenStringSlice(profile.AvailabilityZones)

		if profile.Name != nil {
			agentPoolProfile["name"] = *profile.Name
		}

		if profile.VMSize != "" {
			agentPoolProfile["vm_size"] = string(profile.VMSize)
		}

		if profile.OsDiskSizeGB != nil {
			agentPoolProfile["os_disk_size_gb"] = int(*profile.OsDiskSizeGB)
		}

		if profile.VnetSubnetID != nil {
			agentPoolProfile["vnet_subnet_id"] = *profile.VnetSubnetID
		}

		if profile.OsType != "" {
			agentPoolProfile["os_type"] = string(profile.OsType)
		}

		if profile.MaxPods != nil {
			agentPoolProfile["max_pods"] = int(*profile.MaxPods)
		}

		if profile.NodeTaints != nil {
			agentPoolProfile["node_taints"] = *profile.NodeTaints
		}

		if profile.EnableNodePublicIP != nil {
			agentPoolProfile["enable_node_public_ip"] = *profile.EnableNodePublicIP
		}

		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile)
	}

	return agentPoolProfiles
}

func flattenKubernetesClusterDataSourceLinuxProfile(input *containerservice.LinuxProfile) []interface{} {
	values := make(map[string]interface{})
	sshKeys := make([]interface{}, 0)

	if profile := input; profile != nil {
		if username := profile.AdminUsername; username != nil {
			values["admin_username"] = *username
		}

		if ssh := profile.SSH; ssh != nil {
			if keys := ssh.PublicKeys; keys != nil {
				for _, sshKey := range *keys {
					if keyData := sshKey.KeyData; keyData != nil {
						outputs := make(map[string]interface{})
						outputs["key_data"] = *keyData
						sshKeys = append(sshKeys, outputs)
					}
				}
			}
		}
	}

	values["ssh_key"] = sshKeys

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceWindowsProfile(input *containerservice.ManagedClusterWindowsProfile) []interface{} {
	if input == nil {
		return []interface{}{}
	}
	values := make(map[string]interface{})

	if username := input.AdminUsername; username != nil {
		values["admin_username"] = *username
	}

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceNetworkProfile(profile *containerservice.NetworkProfileType) []interface{} {
	values := make(map[string]interface{})

	values["network_plugin"] = profile.NetworkPlugin

	if profile.NetworkPolicy != "" {
		values["network_policy"] = string(profile.NetworkPolicy)
	}

	if profile.ServiceCidr != nil {
		values["service_cidr"] = *profile.ServiceCidr
	}

	if profile.DNSServiceIP != nil {
		values["dns_service_ip"] = *profile.DNSServiceIP
	}

	if profile.DockerBridgeCidr != nil {
		values["docker_bridge_cidr"] = *profile.DockerBridgeCidr
	}

	if profile.PodCidr != nil {
		values["pod_cidr"] = *profile.PodCidr
	}

	if profile.LoadBalancerSku != "" {
		values["load_balancer_sku"] = string(profile.LoadBalancerSku)
	}

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceServicePrincipalProfile(profile *containerservice.ManagedClusterServicePrincipalProfile) []interface{} {
	if profile == nil {
		return []interface{}{}
	}

	values := make(map[string]interface{})

	if clientId := profile.ClientID; clientId != nil {
		values["client_id"] = *clientId
	}

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceKubeConfig(config kubernetes.KubeConfig) []interface{} {
	values := make(map[string]interface{})

	cluster := config.Clusters[0].Cluster
	user := config.Users[0].User
	name := config.Users[0].Name

	values["host"] = cluster.Server
	values["username"] = name
	values["password"] = user.Token
	values["client_certificate"] = user.ClientCertificteData
	values["client_key"] = user.ClientKeyData
	values["cluster_ca_certificate"] = cluster.ClusterAuthorityData

	return []interface{}{values}
}

func flattenKubernetesClusterDataSourceKubeConfigAAD(config kubernetes.KubeConfigAAD) []interface{} {
	values := make(map[string]interface{})

	cluster := config.Clusters[0].Cluster
	name := config.Users[0].Name

	values["host"] = cluster.Server
	values["username"] = name

	values["password"] = ""
	values["client_certificate"] = ""
	values["client_key"] = ""

	values["cluster_ca_certificate"] = cluster.ClusterAuthorityData

	return []interface{}{values}
}
