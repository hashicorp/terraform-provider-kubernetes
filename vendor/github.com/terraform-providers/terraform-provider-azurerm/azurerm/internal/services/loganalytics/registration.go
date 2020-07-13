package loganalytics

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "Log Analytics"
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_log_analytics_workspace": dataSourceLogAnalyticsWorkspace()}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_log_analytics_linked_service":           resourceArmLogAnalyticsLinkedService(),
		"azurerm_log_analytics_solution":                 resourceArmLogAnalyticsSolution(),
		"azurerm_log_analytics_workspace_linked_service": resourceArmLogAnalyticsWorkspaceLinkedService(),
		"azurerm_log_analytics_workspace":                resourceArmLogAnalyticsWorkspace()}
}
