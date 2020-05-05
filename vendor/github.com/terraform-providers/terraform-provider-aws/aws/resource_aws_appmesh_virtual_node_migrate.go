package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func resourceAwsAppmeshVirtualNodeMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found App Mesh virtual node state v0; migrating to v1")
		return migrateAppmeshVirtualNodeStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAppmeshVirtualNodeStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty App Mesh virtual node state; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	is.Attributes["spec.0.backend.#"] = is.Attributes["spec.0.backends.#"]
	delete(is.Attributes, "spec.0.backends.#")
	for k, v := range is.Attributes {
		if strings.HasPrefix(k, "spec.0.backends.") {
			hash := appmeshVirtualNodeBackendHash(map[string]interface{}{
				"virtual_service": []interface{}{map[string]interface{}{
					"virtual_service_name": v,
				}},
			})
			is.Attributes[fmt.Sprintf("spec.0.backend.%d.virtual_service.#", hash)] = "1"
			is.Attributes[fmt.Sprintf("spec.0.backend.%d.virtual_service.0.virtual_service_name", hash)] = v
			delete(is.Attributes, k)
		}
	}

	if is.Attributes["spec.0.service_discovery.0.dns.#"] == "1" {
		is.Attributes["spec.0.service_discovery.0.dns.0.hostname"] = is.Attributes["spec.0.service_discovery.0.dns.0.service_name"]
		delete(is.Attributes, "spec.0.service_discovery.0.dns.0.service_name")
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
