package common

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

type ServiceRegistration interface {
	// Name is the name of this Service
	Name() string

	// SupportedDataSources returns the supported Data Sources supported by this Service
	SupportedDataSources() map[string]*schema.Resource

	// SupportedResources returns the supported Resources supported by this Service
	SupportedResources() map[string]*schema.Resource
}
