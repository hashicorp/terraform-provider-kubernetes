package main

import "github.com/hashicorp/hcl/v2/hclsimple"

// GeneratorConfig is the top level code generator configuration
type GeneratorConfig struct {
	Resources  []ResourceConfig   `hcl:"resource,block"`
	DataSource []DataSourceConfig `hcl:"data,block"`
}

// ResourceConfig configures code generation for a Terraform resource
type ResourceConfig struct {
	Name    string `hcl:"name,label"`
	Package string `hcl:"package"`

	OutputFilenamePrefix string `hcl:"output_filename_prefix"`

	APIVersion string `hcl:"api_version"`
	Kind       string `hcl:"kind"`

	Description string `hcl:"description"`

	IgnoreFields    []string `hcl:"ignore_fields,optional"`
	ComputedFields  []string `hcl:"computed_fields,optional"`
	SensitiveFields []string `hcl:"sensitive_fields,optional"`

	Generate GenerateConfig `hcl:"generate,block"`

	TerraformPluginGenOpenAPI TerraformPluginGenOpenAPIConfig `hcl:"tfplugingen_openapi,block"`
}

// DataSourceConfig configures code generation for a Terraform data source
type DataSourceConfig struct {
}

// TerraformPluginGenOpenAPIConfig supplies configuration to tfplugingen-openapi
// See: https://github.com/hashicorp/terraform-plugin-codegen-openapi
type TerraformPluginGenOpenAPIConfig struct {
	OpenAPISpecFilename string `hcl:"openapi_spec_filename"`
	CreatePath          string `hcl:"create_path"`
	ReadPath            string `hcl:"read_path"`
}

// GenerateConfig configures the options for what we should generate
type GenerateConfig struct {
	Schema        bool `hcl:"schema,optional"`
	Overrides     bool `hcl:"overrides,optional"`
	Model         bool `hcl:"model,optional"`
	CRUDUNiversal bool `hcl:"crud_universal,optional"`
	CRUDStubs     bool `hcl:"crud_stubs,optional"`
}

func parseGeneratorHCLConfig(filename string) (GeneratorConfig, error) {
	config := GeneratorConfig{}
	err := hclsimple.DecodeFile(filename, nil, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}
