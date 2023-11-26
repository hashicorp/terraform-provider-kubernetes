package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	specresource "github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/spec"
	"gopkg.in/yaml.v2"
)

var codegenTempDir = "./codegen-tmp"

// generateResourceSpec uses the supplied configuration to generate the
// framework IR JSON from an OpenAPI spec then marshalls the IR into
// a spec.Resource
func generateResourceSpec(r ResourceConfig) (specresource.Resource, error) {
	// run tfplugingen-openapi to generate the framework IR for the resource
	// TODO should codify this as a struct when the tool is out of preview
	tfpluginOpenAPIConfig := map[string]any{
		"provider": map[string]any{
			"name": "kubernetes",
		},
		"resources": map[string]any{
			r.Name: map[string]any{
				"create": map[string]any{
					"path":   r.TerraformPluginGenOpenAPI.CreatePath,
					"method": "POST",
				},
				"read": map[string]any{
					"path":   r.TerraformPluginGenOpenAPI.ReadPath,
					"method": "GET",
				},
			},
		},
	}
	yamlConfig, err := yaml.Marshal(tfpluginOpenAPIConfig)
	if err != nil {
		return specresource.Resource{}, fmt.Errorf("error marshalling tfplugingen-openapi configuration: %v", err)
	}

	tfplugingenopenapiPath, err := exec.LookPath("tfplugingen-openapi")
	if err != nil {
		return specresource.Resource{}, fmt.Errorf(`could not find "tfplugingen-openapi" in PATH`)
	}

	os.Mkdir(codegenTempDir, os.ModePerm)

	yamlConfigFile, err := os.CreateTemp(codegenTempDir, "terraform-codegen-*.yaml")
	defer func() {
		yamlConfigFile.Close()
	}()
	if err != nil {
		return specresource.Resource{}, fmt.Errorf("error creating temp file: %v", err)
	}
	yamlConfigFile.WriteString(string(yamlConfig))

	frameworkIRFile, err := os.CreateTemp(codegenTempDir, "terraform-framework-ir-*.json")
	defer func() {
		frameworkIRFile.Close()
	}()
	if err != nil {
		return specresource.Resource{}, fmt.Errorf("error creating temp file: %v", err)
	}

	// TODO it would be nice if there was a module interface for this
	// tool, having to exec the binary is yucky
	yamlConfigFilename := yamlConfigFile.Name()
	frameworkIRFilename := frameworkIRFile.Name()
	cmd := exec.Command(tfplugingenopenapiPath,
		"generate",
		"--config", yamlConfigFilename,
		"--output", frameworkIRFilename,
		r.TerraformPluginGenOpenAPI.OpenAPISpecFilename)

	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			fmt.Println(string(out))
		}
		return specresource.Resource{}, fmt.Errorf("error running tfplugingen-openapi: %v", err)
	}

	contents, err := os.ReadFile(frameworkIRFilename)
	if err != nil {
		return specresource.Resource{}, err
	}
	var spec spec.Specification
	err = json.Unmarshal(contents, &spec)
	if err != nil {
		return specresource.Resource{}, err
	}
	return spec.Resources[0], nil
}
