package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"log/slog"

	"github.com/lmittmann/tint"
)

const generateConfigFilename = "generate.hcl"

func main() {
	// setup slog with colour to make it easier to read
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))

	generateFiles := []string{}
	filepath.Walk("./", func(path string, info fs.FileInfo, err error) error {
		filename := filepath.Base(path)
		if filename == generateConfigFilename {
			generateFiles = append(generateFiles, path)
		}
		return nil
	})

	for _, f := range generateFiles {
		config, err := parseGeneratorHCLConfig(f)
		if err != nil {
			slog.Error("Error parsing configuration", "filename", f, "err", err)
			os.Exit(1)
		}
		err = generateFrameworkCode(f, config)
		if err != nil {
			slog.Error("Error generating framework code", "err", err)
			os.Exit(1)
		}
	}
}

func generateFrameworkCode(path string, config GeneratorConfig) error {
	wd := filepath.Dir(path)

	for _, r := range config.Resources {
		if r.Disabled {
			slog.Warn("Code generation is disabled, skipping", "resource", r.Name)
			continue
		}
		slog.Info("Generating framework code", "resource", r.Name)
		spec, err := generateResourceSpec(r)
		if err != nil {
			return fmt.Errorf("error generating provider spec: %v", err)
		}

		gen := NewResourceGenerator(r, spec)

		// generate resource
		resourceCode := gen.GenerateResourceCode()
		outputFilename := fmt.Sprintf("%s_gen.go", r.OutputFilenamePrefix)
		outputFormattedGoFile(wd, outputFilename, resourceCode)
		slog.Info("Generated resource source file", "filename", outputFilename)

		// generate schema
		if r.Generate.Schema {
			schemaCode := gen.GenerateSchemaFunctionCode()
			outputFilename = fmt.Sprintf("%s_schema_gen.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, schemaCode)
			slog.Info("Generated schema source file", "filename", outputFilename)
		}

		// generate CRUD stubs
		if r.Generate.CRUDStubs {
			crudStubCode := gen.GenerateCRUDStubCode()
			outputFilename = fmt.Sprintf("%s_crud_gen.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, crudStubCode)
			slog.Info("Generated CRUD stub source file", "filename", outputFilename)
		}

		// generate model
		if r.Generate.Model {
			crudStubCode := gen.GenerateModelCode()
			outputFilename = fmt.Sprintf("%s_model_gen.go", r.OutputFilenamePrefix)
			outputFormattedGoFile(wd, outputFilename, crudStubCode)
			slog.Info("Generated model source file", "filename", outputFilename)
		}
	}
	return nil
}
