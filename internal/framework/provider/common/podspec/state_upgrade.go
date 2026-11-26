// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import "fmt"

// UpgradeState converts container resources and image volumes at a known PodSpec path.
// The path consists of retained singleton blocks; genuine collections keep their
// shape. legacyQuantities repairs SDKv2 v0 limits/requests before unwrapping.
func UpgradeState(object map[string]any, location string, path []string, legacyQuantities bool) error {
	if object == nil {
		return fmt.Errorf("%s must be an object", location)
	}
	if len(path) > 0 {
		name := path[0]
		value := object[name]
		if value == nil {
			return nil
		}
		list, ok := value.([]any)
		if !ok || len(list) > 1 {
			return fmt.Errorf("%s.%s must be a list with at most one object", location, name)
		}
		if len(list) == 0 {
			return nil
		}
		child, ok := list[0].(map[string]any)
		if !ok {
			return fmt.Errorf("%s.%s[0] must be an object", location, name)
		}
		return UpgradeState(child, location+"."+name+"[0]", path[1:], legacyQuantities)
	}
	for _, name := range []string{"container", "init_container"} {
		if object[name] == nil {
			continue
		}
		containers, ok := object[name].([]any)
		if !ok {
			return fmt.Errorf("%s.%s must be a list", location, name)
		}
		for i, entry := range containers {
			at := fmt.Sprintf("%s.%s[%d]", location, name, i)
			container, ok := entry.(map[string]any)
			if !ok {
				return fmt.Errorf("%s must be an object", at)
			}
			if container["resources"] == nil {
				continue
			}
			value := container["resources"]
			if list, ok := value.([]any); ok {
				if len(list) > 1 {
					return fmt.Errorf("%s.resources must contain at most one object", at)
				}
				if len(list) == 0 {
					value = map[string]any{"limits": map[string]any{}, "requests": map[string]any{}}
				} else {
					value = list[0]
					if empty, ok := value.(map[string]any); ok && len(empty) == 0 {
						value = map[string]any{"limits": map[string]any{}, "requests": map[string]any{}}
					}
				}
			}
			resources, ok := value.(map[string]any)
			if !ok || resources == nil {
				return fmt.Errorf("%s.resources must be an object or singleton object list", at)
			}
			if legacyQuantities {
				for _, field := range []string{"limits", "requests"} {
					value := resources[field]
					switch v := value.(type) {
					case nil:
						resources[field] = map[string]any{}
					case map[string]any:
						// Already converted; keep its quantity spelling and keys.
					case []any:
						if len(v) > 1 {
							return fmt.Errorf("%s.resources.%s must contain at most one map", at, field)
						}
						converted := map[string]any{}
						if len(v) == 1 {
							var ok bool
							converted, ok = v[0].(map[string]any)
							if !ok || converted == nil {
								return fmt.Errorf("%s.resources.%s[0] must be a map", at, field)
							}
						}
						resources[field] = converted
					default:
						return fmt.Errorf("%s.resources.%s must be a map or legacy singleton map list", at, field)
					}
				}
			}
			container["resources"] = resources
		}
	}
	if object["volume"] == nil {
		return nil
	}
	volumes, ok := object["volume"].([]any)
	if !ok {
		return fmt.Errorf("%s.volume must be a list", location)
	}
	for i, entry := range volumes {
		at := fmt.Sprintf("%s.volume[%d].image", location, i)
		volume, ok := entry.(map[string]any)
		if !ok {
			return fmt.Errorf("%s.volume[%d] must be an object", location, i)
		}
		image := volume["image"]
		if image == nil {
			continue
		}
		if list, ok := image.([]any); ok {
			if len(list) > 1 {
				return fmt.Errorf("%s must contain at most one object", at)
			}
			if len(list) == 0 {
				volume["image"] = nil
				continue
			}
			image = list[0]
		}
		imageObject, ok := image.(map[string]any)
		if !ok || imageObject == nil {
			return fmt.Errorf("%s must be an object or singleton object list", at)
		}
		volume["image"] = imageObject
	}
	return nil
}
