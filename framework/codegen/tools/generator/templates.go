package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/attributes.go.tpl
var attributesTemplate string

//go:embed templates/attribute.go.tpl
var attributeTemplate string

//go:embed templates/schema_function.go.tpl
var schemaFunctionTemplate string

//go:embed templates/resource.go.tpl
var resourceTemplate string

//go:embed templates/crud_stubs.go.tpl
var crudStubsTemplate string

//go:embed templates/model.go.tpl
var modelTemplate string

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
