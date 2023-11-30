package autocrud

import "strings"

func createID(manifest map[string]interface{}) string {
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		id := metadata["name"].(string)
		if ns, ok := metadata["namespace"].(string); ok && ns != "" {
			id = id + "/" + ns
		}
		return id
	}
	return ""
}

func parseID(id string) (string, string) {
	parts := strings.Split(id, "/")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
