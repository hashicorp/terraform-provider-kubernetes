// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestFlattenHTTPRouteSpecRoundtrip(t *testing.T) {
	spec := gatewayv1.HTTPRouteSpec{
		Hostnames: []gatewayv1.Hostname{
			"example.com",
			"*.test.example.com",
		},
		Rules: []gatewayv1.HTTPRouteRule{
			{
				Name: ptr.To(gatewayv1.SectionName("rule1")),
				Matches: []gatewayv1.HTTPRouteMatch{
					{
						Path: &gatewayv1.HTTPPathMatch{
							Type:  ptr.To(gatewayv1.PathMatchType("PathPrefix")),
							Value: ptr.To("/api"),
						},
						Headers: []gatewayv1.HTTPHeaderMatch{
							{
								Name:  gatewayv1.HTTPHeaderName("X-Custom-Header"),
								Value: "value",
								Type:  ptr.To(gatewayv1.HeaderMatchType("Exact")),
							},
						},
						QueryParams: []gatewayv1.HTTPQueryParamMatch{
							{
								Name:  gatewayv1.HTTPHeaderName("color"),
								Value: "blue",
							},
						},
						Method: ptr.To(gatewayv1.HTTPMethod("GET")),
					},
				},
				Filters: []gatewayv1.HTTPRouteFilter{
					{
						Type: gatewayv1.HTTPRouteFilterRequestHeaderModifier,
						RequestHeaderModifier: &gatewayv1.HTTPHeaderFilter{
							Set: []gatewayv1.HTTPHeader{
								{Name: gatewayv1.HTTPHeaderName("X-Request-ID"), Value: "12345"},
							},
						},
					},
					{
						Type: gatewayv1.HTTPRouteFilterRequestRedirect,
						RequestRedirect: &gatewayv1.HTTPRequestRedirectFilter{
							Scheme: ptr.To("https"),
							Port:   ptr.To(gatewayv1.PortNumber(443)),
						},
					},
				},
				BackendRefs: []gatewayv1.HTTPBackendRef{
					{
						BackendRef: gatewayv1.BackendRef{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Group:     ptr.To(gatewayv1.Group("")),
								Kind:      ptr.To(gatewayv1.Kind("Service")),
								Name:      "backend-service",
								Namespace: ptr.To(gatewayv1.Namespace("default")),
								Port:      ptr.To(gatewayv1.PortNumber(8080)),
							},
							Weight: ptr.To(int32(100)),
						},
					},
				},
				Timeouts: &gatewayv1.HTTPRouteTimeouts{
					Request:        ptr.To(gatewayv1.Duration("30s")),
					BackendRequest: ptr.To(gatewayv1.Duration("10s")),
				},
			},
		},
	}

	flattened := flattenHTTPRouteSpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if hostnames, ok := att["hostnames"].([]string); ok {
		if len(hostnames) != 2 {
			t.Errorf("Expected 2 hostnames, got %d", len(hostnames))
		}
		if hostnames[0] != "example.com" {
			t.Errorf("Expected hostname 'example.com', got '%s'", hostnames[0])
		}
	} else {
		t.Error("hostnames not properly flattened")
	}

	if rules, ok := att["rules"].([]interface{}); ok {
		if len(rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(rules))
		}
	} else {
		t.Error("rules not properly flattened")
	}

	t.Logf("HTTPRouteSpec flatten result: %#v", att)
}

func TestFlattenGRPCRouteSpecRoundtrip(t *testing.T) {
	spec := gatewayv1.GRPCRouteSpec{
		Hostnames: []gatewayv1.Hostname{
			"grpc.example.com",
		},
		Rules: []gatewayv1.GRPCRouteRule{
			{
				Name: ptr.To(gatewayv1.SectionName("grpc-rule")),
				Matches: []gatewayv1.GRPCRouteMatch{
					{
						Method: &gatewayv1.GRPCMethodMatch{
							Type:    ptr.To(gatewayv1.GRPCMethodMatchType("Exact")),
							Service: ptr.To("echo.Echo"),
							Method:  ptr.To("Ping"),
						},
						Headers: []gatewayv1.GRPCHeaderMatch{
							{
								Name:  gatewayv1.GRPCHeaderName("custom-header"),
								Value: "value",
							},
						},
					},
				},
				Filters: []gatewayv1.GRPCRouteFilter{
					{
						Type: gatewayv1.GRPCRouteFilterExtensionRef,
						ExtensionRef: &gatewayv1.LocalObjectReference{
							Name: "my-extension",
						},
					},
				},
				BackendRefs: []gatewayv1.GRPCBackendRef{
					{
						BackendRef: gatewayv1.BackendRef{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Group:     ptr.To(gatewayv1.Group("")),
								Kind:      ptr.To(gatewayv1.Kind("Service")),
								Name:      "grpc-backend",
								Namespace: ptr.To(gatewayv1.Namespace("default")),
								Port:      ptr.To(gatewayv1.PortNumber(50051)),
							},
							Weight: ptr.To(int32(50)),
						},
					},
				},
				SessionPersistence: &gatewayv1.SessionPersistence{
					SessionName: ptr.To("grpc-session"),
					Type:        ptr.To(gatewayv1.SessionPersistenceType("Cookie")),
				},
			},
		},
	}

	flattened := flattenGRPCRouteSpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if hostnames, ok := att["hostnames"].([]string); ok {
		if len(hostnames) != 1 {
			t.Errorf("Expected 1 hostname, got %d", len(hostnames))
		}
	} else {
		t.Error("hostnames not properly flattened")
	}

	if rules, ok := att["rules"].([]interface{}); ok {
		if len(rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(rules))
		}
	} else {
		t.Error("rules not properly flattened")
	}

	t.Logf("GRPCRouteSpec flatten result: %#v", att)
}

func TestFlattenTLSRouteSpecRoundtrip(t *testing.T) {
	spec := gatewayv1.TLSRouteSpec{
		Hostnames: []gatewayv1.Hostname{
			"tls.example.com",
		},
		Rules: []gatewayv1.TLSRouteRule{
			{
				Name: ptr.To(gatewayv1.SectionName("tls-rule")),
				BackendRefs: []gatewayv1.BackendRef{
					{
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Group:     ptr.To(gatewayv1.Group("")),
							Kind:      ptr.To(gatewayv1.Kind("Service")),
							Name:      "tls-backend",
							Namespace: ptr.To(gatewayv1.Namespace("default")),
							Port:      ptr.To(gatewayv1.PortNumber(443)),
						},
						Weight: ptr.To(int32(100)),
					},
				},
			},
		},
	}

	flattened := flattenTLSRouteSpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if hostnames, ok := att["hostnames"].([]string); ok {
		if len(hostnames) != 1 {
			t.Errorf("Expected 1 hostname, got %d", len(hostnames))
		}
	} else {
		t.Error("hostnames not properly flattened")
	}

	t.Logf("TLSRouteSpec flatten result: %#v", att)
}

func TestFlattenListenerSetSpecRoundtrip(t *testing.T) {
	protocol := gatewayv1.HTTPProtocolType
	port := gatewayv1.PortNumber(80)
	mode := gatewayv1.TLSModeType("Terminate")

	spec := gatewayv1.ListenerSetSpec{
		ParentRef: gatewayv1.ParentGatewayReference{
			Group:     ptr.To(gatewayv1.Group("gateway.networking.k8s.io")),
			Kind:      ptr.To(gatewayv1.Kind("Gateway")),
			Name:      "my-gateway",
			Namespace: ptr.To(gatewayv1.Namespace("istio-system")),
		},
		Listeners: []gatewayv1.ListenerEntry{
			{
				Name:     "http-listener",
				Hostname: ptr.To(gatewayv1.Hostname("*.example.com")),
				Port:     port,
				Protocol: protocol,
				TLS: &gatewayv1.ListenerTLSConfig{
					Mode: &mode,
					CertificateRefs: []gatewayv1.SecretObjectReference{
						{
							Group: ptr.To(gatewayv1.Group("")),
							Kind:  ptr.To(gatewayv1.Kind("Secret")),
							Name:  "my-cert",
						},
					},
				},
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Namespaces: &gatewayv1.RouteNamespaces{
						From: ptr.To(gatewayv1.FromNamespaces("Same")),
					},
					Kinds: []gatewayv1.RouteGroupKind{
						{Group: ptr.To(gatewayv1.Group("gateway.networking.k8s.io")), Kind: gatewayv1.Kind("HTTPRoute")},
						{Group: ptr.To(gatewayv1.Group("gateway.networking.k8s.io")), Kind: gatewayv1.Kind("GRPCRoute")},
					},
				},
			},
		},
	}

	flattened := flattenListenerSetSpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if listeners, ok := att["listeners"].([]interface{}); ok {
		if len(listeners) != 1 {
			t.Errorf("Expected 1 listener, got %d", len(listeners))
		}
	} else {
		t.Error("listeners not properly flattened")
	}

	if parentRef, ok := att["parent_ref"].([]interface{}); ok {
		if len(parentRef) != 1 {
			t.Errorf("Expected 1 parent_ref, got %d", len(parentRef))
		}
	} else {
		t.Error("parent_ref not properly flattened")
	}

	t.Logf("ListenerSetSpec flatten result: %#v", att)
}

func TestFlattenGatewayInfrastructure(t *testing.T) {
	spec := gatewayv1.GatewaySpec{
		GatewayClassName: "test-class",
		Infrastructure: &gatewayv1.GatewayInfrastructure{
			Labels: map[gatewayv1.LabelKey]gatewayv1.LabelValue{
				"env":  "production",
				"team": "networking",
			},
			Annotations: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{
				"description": "Test gateway",
			},
			ParametersRef: &gatewayv1.LocalParametersReference{
				Group: "config.example.com",
				Kind:  "GatewayParameters",
				Name:  "my-params",
			},
		},
		AllowedListeners: &gatewayv1.AllowedListeners{
			Namespaces: &gatewayv1.ListenerNamespaces{
				From: ptr.To(gatewayv1.FromNamespaces("All")),
			},
		},
		Listeners: []gatewayv1.Listener{
			{
				Name:     "http",
				Port:     80,
				Protocol: gatewayv1.HTTPProtocolType,
			},
		},
	}

	flattened := flattenGatewayV1Spec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if infra, ok := att["infrastructure"].([]interface{}); ok {
		if len(infra) != 1 {
			t.Errorf("Expected 1 infrastructure, got %d", len(infra))
		}
		infraMap := infra[0].(map[string]interface{})

		if labels, ok := infraMap["labels"].(map[string]string); ok {
			if labels["env"] != "production" {
				t.Errorf("Expected label 'production', got '%s'", labels["env"])
			}
		} else {
			t.Error("labels not properly flattened")
		}
	} else {
		t.Error("infrastructure not properly flattened")
	}

	if allowedListeners, ok := att["allowed_listeners"].([]interface{}); ok {
		if len(allowedListeners) != 1 {
			t.Errorf("Expected 1 allowed_listeners, got %d", len(allowedListeners))
		}
	} else {
		t.Error("allowed_listeners not properly flattened")
	}

	t.Logf("GatewaySpec with infrastructure flatten result: %#v", att)
}

func TestFlattenBackendTLSPolicySpecRoundtrip(t *testing.T) {
	hostname := gatewayv1.PreciseHostname("backend.example.com")

	spec := gatewayv1.BackendTLSPolicySpec{
		TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{
			{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: "gateway.networking.k8s.io",
					Kind:  "GRPCRoute",
					Name:  "my-route",
				},
			},
		},
		Validation: gatewayv1.BackendTLSPolicyValidation{
			CACertificateRefs: []gatewayv1.LocalObjectReference{
				{Name: "ca-cert"},
			},
			Hostname: hostname,
			SubjectAltNames: []gatewayv1.SubjectAltName{
				{
					Type:     gatewayv1.SubjectAltNameType("DNS"),
					Hostname: gatewayv1.Hostname("secure.example.com"),
				},
			},
		},
		Options: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{
			"timeout": "5s",
		},
	}

	flattened := flattenBackendTLSPolicySpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if targetRefs, ok := att["target_refs"].([]interface{}); ok {
		if len(targetRefs) != 1 {
			t.Errorf("Expected 1 target_ref, got %d", len(targetRefs))
		}
	} else {
		t.Error("target_refs not properly flattened")
	}

	t.Logf("BackendTLSPolicySpec flatten result: %#v", att)
}

func TestFlattenReferenceGrantSpecRoundtrip(t *testing.T) {
	spec := gatewayv1.ReferenceGrantSpec{
		From: []gatewayv1.ReferenceGrantFrom{
			{
				Group:     "gateway.networking.k8s.io",
				Kind:      "GRPCRoute",
				Namespace: "sandbox",
			},
		},
		To: []gatewayv1.ReferenceGrantTo{
			{
				Group: "",
				Kind:  "Service",
			},
		},
	}

	flattened := flattenReferenceGrantSpec(spec)

	if len(flattened) != 1 {
		t.Fatalf("Expected 1 flattened spec, got %d", len(flattened))
	}

	att := flattened[0].(map[string]interface{})

	if from, ok := att["from"].([]interface{}); ok {
		if len(from) != 1 {
			t.Errorf("Expected 1 from, got %d", len(from))
		}
	} else {
		t.Error("from not properly flattened")
	}

	if to, ok := att["to"].([]interface{}); ok {
		if len(to) != 1 {
			t.Errorf("Expected 1 to, got %d", len(to))
		}
	} else {
		t.Error("to not properly flattened")
	}

	t.Logf("ReferenceGrantSpec flatten result: %#v", att)
}

func TestExpandHTTPRouteSpecRoundtrip(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"hostnames": []interface{}{"example.com", "*.test.com"},
			"rules": []interface{}{
				map[string]interface{}{
					"name": "test-rule",
					"matches": []interface{}{
						map[string]interface{}{
							"path": []interface{}{
								map[string]interface{}{
									"type":  "PathPrefix",
									"value": "/api",
								},
							},
							"headers": []interface{}{
								map[string]interface{}{
									"name":  "X-Custom",
									"value": "test",
									"type":  "Exact",
								},
							},
						},
					},
					"backend_refs": []interface{}{
						map[string]interface{}{
							"name":      "backend-svc",
							"namespace": "default",
							"port":      8080,
							"weight":    100,
						},
					},
				},
			},
		},
	}

	expanded := expandHTTPRouteSpec(input)

	if len(expanded.Hostnames) != 2 {
		t.Errorf("Expected 2 hostnames, got %d", len(expanded.Hostnames))
	}

	if len(expanded.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(expanded.Rules))
	}

	if expanded.Rules[0].Name == nil || *expanded.Rules[0].Name != "test-rule" {
		t.Error("Rule name not correctly expanded")
	}

	if len(expanded.Rules[0].BackendRefs) != 1 {
		t.Errorf("Expected 1 backend_ref, got %d", len(expanded.Rules[0].BackendRefs))
	} else {
		br := expanded.Rules[0].BackendRefs[0]
		if string(br.Name) != "backend-svc" {
			t.Errorf("Expected backend name 'backend-svc', got '%s'", br.Name)
		}
		if br.Port == nil || *br.Port != 8080 {
			t.Error("Expected backend port 8080")
		}
	}

	t.Logf("HTTPRouteSpec expand result: %#v", expanded)
}

func TestExpandListenerSetSpecRoundtrip(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"parent_ref": []interface{}{
				map[string]interface{}{
					"group":     "gateway.networking.k8s.io",
					"kind":      "Gateway",
					"name":      "my-gateway",
					"namespace": "istio-system",
				},
			},
			"listeners": []interface{}{
				map[string]interface{}{
					"name":     "http",
					"hostname": "*.example.com",
					"port":     80,
					"protocol": "HTTP",
					"tls": []interface{}{
						map[string]interface{}{
							"mode": "Terminate",
							"certificate_refs": []interface{}{
								map[string]interface{}{
									"name": "my-cert",
								},
							},
						},
					},
					"allowed_routes": []interface{}{
						map[string]interface{}{
							"namespaces": []interface{}{
								map[string]interface{}{
									"from": "Same",
								},
							},
						},
					},
				},
			},
		},
	}

	expanded := expandListenerSetSpec(input)

	if expanded.ParentRef.Name != "my-gateway" {
		t.Errorf("Expected parent name 'my-gateway', got '%s'", expanded.ParentRef.Name)
	}

	if len(expanded.Listeners) != 1 {
		t.Errorf("Expected 1 listener, got %d", len(expanded.Listeners))
	}

	if expanded.Listeners[0].Name != "http" {
		t.Errorf("Expected listener name 'http', got '%s'", expanded.Listeners[0].Name)
	}

	t.Logf("ListenerSetSpec expand result: %#v", expanded)
}

func TestExpandGatewaySpecWithNewFields(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"gateway_class_name": "istio",
			"infrastructure": []interface{}{
				map[string]interface{}{
					"labels": map[string]interface{}{
						"env":  "prod",
						"team": "platform",
					},
					"annotations": map[string]interface{}{
						"note": "production gateway",
					},
					"parameters_ref": []interface{}{
						map[string]interface{}{
							"group": "config.example.com",
							"kind":  "GatewayParams",
							"name":  "prod-params",
						},
					},
				},
			},
			"allowed_listeners": []interface{}{
				map[string]interface{}{
					"namespaces": []interface{}{
						map[string]interface{}{
							"from": "All",
						},
					},
				},
			},
			"listeners": []interface{}{
				map[string]interface{}{
					"name":     "https",
					"port":     443,
					"protocol": "HTTPS",
				},
			},
		},
	}

	expanded, err := expandGatewayV1Spec(input)
	if err != nil {
		t.Fatalf("expandGatewayV1Spec failed: %v", err)
	}

	if expanded.Infrastructure == nil {
		t.Fatal("Infrastructure not expanded")
	}

	if len(expanded.Infrastructure.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(expanded.Infrastructure.Labels))
	}

	if expanded.Infrastructure.Labels["env"] != "prod" {
		t.Errorf("Expected label 'prod', got '%s'", expanded.Infrastructure.Labels["env"])
	}

	if expanded.AllowedListeners == nil {
		t.Fatal("AllowedListeners not expanded")
	}

	if expanded.AllowedListeners.Namespaces == nil {
		t.Fatal("AllowedListeners.Namespaces not expanded")
	}

	if *expanded.AllowedListeners.Namespaces.From != "All" {
		t.Errorf("Expected From 'All', got '%s'", *expanded.AllowedListeners.Namespaces.From)
	}

	t.Logf("GatewaySpec with new fields expand result: %#v", expanded)
}

func TestExpandGRPCRouteSpecRoundtrip(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"hostnames": []interface{}{"grpc.example.com"},
			"rules": []interface{}{
				map[string]interface{}{
					"name": "grpc-rule",
					"matches": []interface{}{
						map[string]interface{}{
							"method": []interface{}{
								map[string]interface{}{
									"type":    "Exact",
									"service": "echo.Echo",
									"method":  "Ping",
								},
							},
							"headers": []interface{}{
								map[string]interface{}{
									"name":  "custom-header",
									"value": "value",
								},
							},
						},
					},
					"backend_refs": []interface{}{
						map[string]interface{}{
							"name":      "grpc-backend",
							"namespace": "default",
							"port":      50051,
							"weight":    50,
						},
					},
					"session_persistence": []interface{}{
						map[string]interface{}{
							"session_name": "grpc-session",
							"type":         "Cookie",
						},
					},
				},
			},
		},
	}

	expanded := expandGRPCRouteSpec(input)

	if len(expanded.Hostnames) != 1 {
		t.Errorf("Expected 1 hostname, got %d", len(expanded.Hostnames))
	}

	if len(expanded.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(expanded.Rules))
	}

	if expanded.Rules[0].Name == nil || *expanded.Rules[0].Name != "grpc-rule" {
		t.Error("Rule name not correctly expanded")
	}

	if expanded.Rules[0].SessionPersistence == nil {
		t.Error("SessionPersistence not expanded")
	}

	if len(expanded.Rules[0].BackendRefs) != 1 {
		t.Errorf("Expected 1 backend_ref, got %d", len(expanded.Rules[0].BackendRefs))
	} else {
		br := expanded.Rules[0].BackendRefs[0]
		if string(br.Name) != "grpc-backend" {
			t.Errorf("Expected backend name 'grpc-backend', got '%s'", br.Name)
		}
		if br.Port == nil || *br.Port != 50051 {
			t.Error("Expected backend port 50051")
		}
	}

	t.Logf("GRPCRouteSpec expand result: %#v", expanded)
}

func TestExpandTLSRouteSpecRoundtrip(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"hostnames": []interface{}{"tls.example.com"},
			"rules": []interface{}{
				map[string]interface{}{
					"name": "tls-rule",
					"backend_refs": []interface{}{
						map[string]interface{}{
							"name":      "tls-backend",
							"namespace": "default",
							"port":      443,
							"weight":    100,
						},
					},
				},
			},
		},
	}

	expanded := expandTLSRouteSpec(input)

	if len(expanded.Hostnames) != 1 {
		t.Errorf("Expected 1 hostname, got %d", len(expanded.Hostnames))
	}

	if len(expanded.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(expanded.Rules))
	}

	if len(expanded.Rules[0].BackendRefs) != 1 {
		t.Errorf("Expected 1 backend_ref, got %d", len(expanded.Rules[0].BackendRefs))
	} else {
		br := expanded.Rules[0].BackendRefs[0]
		if string(br.Name) != "tls-backend" {
			t.Errorf("Expected backend name 'tls-backend', got '%s'", br.Name)
		}
		if br.Port == nil || *br.Port != 443 {
			t.Error("Expected backend port 443")
		}
	}

	t.Logf("TLSRouteSpec expand result: %#v", expanded)
}

func TestExpandReferenceGrantSpecRoundtrip(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"from": []interface{}{
				map[string]interface{}{
					"group":     "gateway.networking.k8s.io",
					"kind":      "GRPCRoute",
					"namespace": "sandbox",
				},
			},
			"to": []interface{}{
				map[string]interface{}{
					"group": "",
					"kind":  "Service",
				},
			},
		},
	}

	expanded := expandReferenceGrantSpec(input)

	if len(expanded.From) != 1 {
		t.Errorf("Expected 1 from, got %d", len(expanded.From))
	}

	if len(expanded.To) != 1 {
		t.Errorf("Expected 1 to, got %d", len(expanded.To))
	}

	if expanded.From[0].Namespace != "sandbox" {
		t.Errorf("Expected namespace 'sandbox', got '%s'", expanded.From[0].Namespace)
	}

	t.Logf("ReferenceGrantSpec expand result: %#v", expanded)
}
