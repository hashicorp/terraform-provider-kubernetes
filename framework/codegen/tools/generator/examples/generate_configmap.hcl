
resource "kubernetes_config_map_v1" {
  package = "corev1"

  api_version = "v1"
  kind        = "ConfigMap"

  description = "configmaps store information for pods"

  output_filename_prefix = "config_map"

  tfplugingen_openapi {
    openapi_spec_filename = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    
    create_path = "/api/v1/namespaces/{namespace}/configmaps"
    read_path   = "/api/v1/namespaces/{namespace}/configmaps/{name}"
  }

  generate {
    schema     = true
    models     = true
    crud_stubs = true
  }
}
