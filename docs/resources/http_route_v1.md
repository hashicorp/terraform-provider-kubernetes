---
subcategory: "networking/gateway"
page_title: "Kubernetes: kubernetes_httproute_v1"
description: |-
  HTTPRoute provides a way to route HTTP requests.
---

# kubernetes_httproute_v1

HTTPRoute provides a way to route HTTP requests.

## Example Usage

```hcl
resource "kubernetes_gateway_class_v1" "example" {
  metadata {
    name = "example-gateway-class"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "example" {
  metadata {
    name      = "example-gateway"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.example.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_httproute_v1" "example" {
  metadata {
    name      = "example-httproute"
    namespace = "default"
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.example.metadata.0.name
    }
    hostnames = ["example.com", "*.example.com"]
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
        method = "GET"
        headers {
          name  = "X-Custom-Header"
          value = "custom-value"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Request-ID"
            value = "generated-uuid"
          }
        }
      }
      backend_refs {
        name = "backend-service"
        port = 8080
      }
      timeouts {
        request = "30s"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `metadata` (Block List, Required) Standard route's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
- `spec` (Block List, Required) Spec defines the desired state of HTTPRoute.
- `timeouts` (Block, Optional) Standard resource's timeouts. More info: https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/timeouts

### Nested Schema for `metadata`

Required:

- `name` (String) Name of the route, must be unique. Cannot be updated.

Optional:

- `namespace` (String) Namespace defines the space within which name of the route must be unique.
- `labels` (Map of String) Map of string keys and values that can be used to organize and categorize the route.
- `annotations` (Map of String) An unstructured key value map stored with the route.

### Nested Schema for `spec`

Optional:

- `parent_refs` (Block List) ParentRefs references the resources that this Route is attached to.
- `hostnames` (List of String) Hostnames defines a set of hostnames that should match against the HTTP Host header.
- `rules` (Block List) Rules are a list of HTTP matchers, filters and actions.

### Nested Schema for `spec.parent_refs`

Required:

- `name` (String) Name of the referent.

Optional:

- `namespace` (String) Namespace of the referent.
- `group` (String) Group of the referent.
- `kind` (String) Kind of the referent.
- `port` (Number) Port of the referent.

### Nested Schema for `spec.rules`

Optional:

- `name` (String) Name of the route rule.
- `matches` (Block List) Matches define conditions used for matching the rule against incoming HTTP requests.
- `filters` (Block List) Filters define filters that modify the request or response.
- `backend_refs` (Block List) BackendRefs defines the backend(s) where requests matching this rule should be sent.
- `timeouts` (Block, Max: 1) Timeouts defines the timeout values for requests.
- `retry` (Block, Max: 1) Retry defines the retry behavior for requests.
- `session_persistence` (Block, Max: 1) SessionPersistence defines the session persistence configuration.

### Nested Schema for `spec.rules.matches`

Optional:

- `path` (Block, Max: 1) Path specifies the HTTP request path match.
- `headers` (Block List) Headers specifies the HTTP request header match.
- `query_params` (Block List) QueryParams specifies the HTTP query parameter match.
- `method` (String) Method specifies the HTTP method match. Valid values: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE.

### Nested Schema for `spec.rules.filters`

Required:

- `type` (String) Type of filter. Valid values: RequestHeaderModifier, ResponseHeaderModifier, RequestRedirect, URLRewrite, RequestMirror, ExtensionRef.

Optional:

- `request_header_modifier` (Block) RequestHeaderModifier defines a filter that modifies request headers.
- `response_header_modifier` (Block) ResponseHeaderModifier defines a filter that modifies response headers.
- `request_redirect` (Block) RequestRedirect defines a filter that redirects requests.
- `url_rewrite` (Block) URLRewrite defines a filter that rewrites the request URL.
- `request_mirror` (Block) RequestMirror defines a filter that mirrors requests.
- `extension_ref` (Block) ExtensionRef defines a filter that references an external resource.

## Import

`kubernetes_httproute_v1` can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_httproute_v1.example default/example-httproute
```