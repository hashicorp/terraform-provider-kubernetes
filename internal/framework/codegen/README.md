# Kubernetes Provider Code Generator

This directory contains tools that can be used to generate Terraform resources and data sources using the Kubernetes OpenAPI specifications. 


## Downloading the OpenAPI spec

There is a script that will download the OpenAPI specification JSON files from the main kubernetes GitHub repository. 

To download the OpenAPI specifications:

```
cd data 
go generate 
```

The JSON files for each API group will appear under the directory `data/kubernetes-$VERSION`. You can update the version tag in [generate.go](./data/generate.go).