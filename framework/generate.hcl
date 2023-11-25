resource "kubernetes_namespace_v1" {
    package = "corev1"

    api_version = "v1"
    kind = "Namespace"

    output_filename = "namespace_v1.go"

    tfplugingen_openapi {
        openapi_spec_filename = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    }
}

resource "kubernetes_config_map_v1" {
    package = "corev1"

    api_version = "v1"
    kind = "ConfigMap"

    output_filename = "config_map_v1.go"

    tfplugingen_openapi {
        openapi_spec_filename = "./codegen/data/kubernetes-v1.28.3/api/openapi-spec/v3/api__v1_openapi.json"
    }
}



