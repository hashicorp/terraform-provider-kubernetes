package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	// find all generate.hcl files
	generateFiles := []string{}
	filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		filename := filepath.Base(path)
		if filename == "generate.hcl" {
			generateFiles = append(generateFiles, path)
		}
		return nil
	})

	for _, f := range generateFiles {
		config, err := parseGeneratorHCLConfig(f)
		if err != nil {
			fmt.Printf("error parsing %v: %v\n", f, err)
			os.Exit(1)
		}
		err = generateFrameworkCode(f, config)
		if err != nil {
			fmt.Printf("error generating framework code: %v\n", err)
			os.Exit(1)
		}
	}
}

func generateFrameworkCode(path string, config GeneratorConfig) error {
	wd := filepath.Dir(path)
	fmt.Printf("generating code in %s\n", wd)

	for _, r := range config.Resources {
		spec, err := generateResourceSpec(r)
		if err != nil {
			return fmt.Errorf("error generating provider spec: %v", err)
		}

		gen := NewResourceGenerator(r, spec)

		// generate resource
		resourceCode := gen.GenerateResourceCode()
		outputFilename := fmt.Sprintf("%s.go", r.OutputFilenamePrefix)
		outputFormattedGoFile(wd, outputFilename, resourceCode)

		// generate schema
		if r.Generate.Schema {
			schemaCode := gen.GenerateSchemaFunctionCode()
			outputFilename = fmt.Sprintf("%s_schema.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, schemaCode)
		}

		// generate CRUD stubs
		if r.Generate.CRUDStubs {
			crudStubCode := gen.GenerateCRUDStubCode()
			outputFilename = fmt.Sprintf("%s_crud.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, crudStubCode)
		}

		// generate model
		if r.Generate.Model {
			crudStubCode := gen.GenerateModelCode()
			outputFilename = fmt.Sprintf("%s_model.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, crudStubCode)
		}
	}
	return nil
}
