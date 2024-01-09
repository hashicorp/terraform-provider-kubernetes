package main

type AttributeGenerator struct {
	Name string

	AttributeType string
	ElementType   string

	Required    bool
	Description string
	Computed    bool
	Sensitive   bool

	NestedAttributes AttributesGenerator
}

func (g AttributeGenerator) String() string {
	return renderTemplate(attributeTemplate, g)
}

type AttributesGenerator []AttributeGenerator

func (g AttributesGenerator) String() string {
	return renderTemplate(attributesTemplate, g)
}
