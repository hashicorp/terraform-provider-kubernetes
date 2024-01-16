package main

type ModelFieldGenerator struct {
	// ModelFieldName is the name the attribute has in the model struct, e.g ApiVersion
	FieldName   string
	Type        string
	ElementType string

	// AttributeName is the name of the attribute in the terraform schema api_version
	AttributeName string
	AttributeType string

	// ManifestFieldName is the name the attribute has in the Kubernetes manifest, e.g apiVersion
	ManifestFieldName string

	NestedFields ModelFieldsGenerator
}

func (g ModelFieldGenerator) String() string {
	return renderTemplate(modelFieldTemplate, g)
}

type ModelFieldsGenerator []ModelFieldGenerator

func (g ModelFieldsGenerator) String() string {
	return renderTemplate(modelFieldsTemplate, g)
}
