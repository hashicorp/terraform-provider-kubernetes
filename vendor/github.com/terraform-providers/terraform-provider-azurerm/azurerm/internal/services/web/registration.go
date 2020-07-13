package web

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "Web"
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_app_service_plan":              dataSourceAppServicePlan(),
		"azurerm_app_service_certificate":       dataSourceAppServiceCertificate(),
		"azurerm_app_service":                   dataSourceArmAppService(),
		"azurerm_app_service_certificate_order": dataSourceArmAppServiceCertificateOrder(),
		"azurerm_function_app":                  dataSourceArmFunctionApp(),
	}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_app_service_active_slot":                      resourceArmAppServiceActiveSlot(),
		"azurerm_app_service_certificate":                      resourceArmAppServiceCertificate(),
		"azurerm_app_service_certificate_order":                resourceArmAppServiceCertificateOrder(),
		"azurerm_app_service_custom_hostname_binding":          resourceArmAppServiceCustomHostnameBinding(),
		"azurerm_app_service_plan":                             resourceArmAppServicePlan(),
		"azurerm_app_service_slot":                             resourceArmAppServiceSlot(),
		"azurerm_app_service_source_control_token":             resourceArmAppServiceSourceControlToken(),
		"azurerm_app_service_virtual_network_swift_connection": resourceArmAppServiceVirtualNetworkSwiftConnection(),
		"azurerm_app_service":                                  resourceArmAppService(),
		"azurerm_function_app":                                 resourceArmFunctionApp(),
	}
}
