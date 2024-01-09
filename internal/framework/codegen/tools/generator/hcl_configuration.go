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

	IgnoreAttributes    []string `hcl:"ignore_attributes,optional"`
	ComputedAttributes  []string `hcl:"computed_attributes,optional"`
	SensitiveAttributes []string `hcl:"sensitive_attributes,optional"`

	Generate GenerateConfig `hcl:"generate,block"`

	OpenAPIConfig TerraformPluginGenOpenAPIConfig `hcl:"openapi,block"`

	Disabled bool `hcl:"disabled,optional"`
}

// DataSourceConfig configures code generation for a Terraform data source
type DataSourceConfig struct {
	// TODO implement data source generation
}

// TerraformPluginGenOpenAPIConfig supplies configuration to tfplugingen-openapi
// See: https://github.com/hashicorp/terraform-plugin-codegen-openapi
type TerraformPluginGenOpenAPIConfig struct {
	Filename   string `hcl:"filename"`
	CreatePath string `hcl:"create_path"`
	ReadPath   string `hcl:"read_path"`
}

// GenerateConfig configures the options for what we should generate
type GenerateConfig struct {
	Schema    bool `hcl:"schema,optional"`
	Overrides bool `hcl:"overrides,optional"`
	Model     bool `hcl:"model,optional"`
	CRUDAuto  bool `hcl:"autocrud,optional"`
	CRUDStubs bool `hcl:"crud_stubs,optional"`
}

func parseGeneratorHCLConfig(filename string) (GeneratorConfig, error) {
	config := GeneratorConfig{}
	err := hclsimple.DecodeFile(filename, nil, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}
