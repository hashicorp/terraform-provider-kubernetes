package mysql

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "MySQL"
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_mysql_configuration":        resourceArmMySQLConfiguration(),
		"azurerm_mysql_database":             resourceArmMySqlDatabase(),
		"azurerm_mysql_firewall_rule":        resourceArmMySqlFirewallRule(),
		"azurerm_mysql_server":               resourceArmMySqlServer(),
		"azurerm_mysql_virtual_network_rule": resourceArmMySqlVirtualNetworkRule()}
}
