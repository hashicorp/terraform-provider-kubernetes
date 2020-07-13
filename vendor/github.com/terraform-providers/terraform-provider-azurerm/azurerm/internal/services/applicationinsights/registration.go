package applicationinsights

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "Application Insights"
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_application_insights": dataSourceArmApplicationInsights(),
	}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_application_insights_api_key":        resourceArmApplicationInsightsAPIKey(),
		"azurerm_application_insights":                resourceArmApplicationInsights(),
		"azurerm_application_insights_analytics_item": resourceArmApplicationInsightsAnalyticsItem(),
		"azurerm_application_insights_web_test":       resourceArmApplicationInsightsWebTests(),
	}
}
