package main

import "github.com/hashicorp/hcl/v2/hclsimple"

// GeneratorConfig is the top level code generator configuration
type GeneratorConfig struct {
	Resources  []Resource   `hcl:"resource,block"`
	DataSource []DataSource `hcl:"data,block"`
}

// Resource configures code generation for a Terraform resource
type Resource struct {
	Name    string `hcl:"name,label"`
	Package string `hcl:"package"`

	OutputFilename    string `hcl:"output_filename,optional"`
	OverridesFilename string `hcl:"overrides_filename,optional"`

	APIVersion string `hcl:"api_version"`
	Kind       string `hcl:"kind"`

	IgnoreFields   []string `hcl:"ignore_fields,optional"`
	ComputedFields []string `hcl:"computed_fields,optional"`

	TerraformPluginGenOpenAPI TerraformPluginGenOpenAPIConfig `hcl:"tfplugingen_openapi,block"`
}

// DataSource configures code generation for a Terraform data source
type DataSource struct {
}

// TerraformPluginGenOpenAPIConfig supplies configuration to tfplugingen-openapi
// See: https://github.com/hashicorp/terraform-plugin-codegen-openapi
type TerraformPluginGenOpenAPIConfig struct {
	OpenAPISpecFilename string `hcl:"openapi_spec_filename"`
}

func parseGeneratorHCLConfig(filename string) (GeneratorConfig, error) {
	config := GeneratorConfig{}
	err := hclsimple.DecodeFile(filename, nil, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}
