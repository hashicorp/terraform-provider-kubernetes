package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/attributes.tpl
var attributesTemplate string

//go:embed templates/attribute.tpl
var attributeTemplate string

//go:embed templates/schema.tpl
var schemaTemplate string

//go:embed templates/resource_schema.go.tpl
var schemaFunctionTemplate string

//go:embed templates/resource.go.tpl
var resourceTemplate string

//go:embed templates/resources_list.go.tpl
var resourcesListTemplate string

//go:embed templates/resource_crud_stubs.go.tpl
var crudStubsTemplate string

//go:embed templates/resource_autocrud.go.tpl
var autocrudTemplate string

//go:embed templates/resource_model.go.tpl
var modelTemplate string

//go:embed templates/model_fields.tpl
var modelFieldsTemplate string

//go:embed templates/model_field.tpl
var modelFieldTemplate string

func renderTemplate(path string, r any) string {
	tpl, err := template.New("").Parse(path)
	if err != nil {
		panic(fmt.Sprintf("could not parse template: %v", err))
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, r)
	if err != nil {
		panic(fmt.Sprintf("error executing template: %v", err))
	}
	return buf.String()
}
