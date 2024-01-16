package main

type SchemaGenerator struct {
	Name        string
	Description string
	Attributes  AttributesGenerator
}

func (g SchemaGenerator) String() string {
	return renderTemplate(schemaTemplate, g)
}
