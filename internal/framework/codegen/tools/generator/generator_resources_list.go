package main

import "time"

type ResourcesListGenerator struct {
	GeneratedTimestamp time.Time
	Resources          []ResourceConfig
	Packages           []string
}

func (p ResourcesListGenerator) String() string {
	return renderTemplate(resourcesListTemplate, p)
}
