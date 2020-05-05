package aws

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/mitchellh/copystructure"
)

// EcsContainerDefinitionsAreEquivalent determines equality between two ECS container definition JSON strings
// Note: This function will be moved out of the aws package in the future.
func EcsContainerDefinitionsAreEquivalent(def1, def2 string, isAWSVPC bool) (bool, error) {
	var obj1 containerDefinitions
	err := json.Unmarshal([]byte(def1), &obj1)
	if err != nil {
		return false, err
	}
	err = obj1.Reduce(isAWSVPC)
	if err != nil {
		return false, err
	}
	canonicalJson1, err := jsonutil.BuildJSON(obj1)
	if err != nil {
		return false, err
	}

	var obj2 containerDefinitions
	err = json.Unmarshal([]byte(def2), &obj2)
	if err != nil {
		return false, err
	}
	err = obj2.Reduce(isAWSVPC)
	if err != nil {
		return false, err
	}

	canonicalJson2, err := jsonutil.BuildJSON(obj2)
	if err != nil {
		return false, err
	}

	equal := bytes.Equal(canonicalJson1, canonicalJson2)
	if !equal {
		log.Printf("[DEBUG] Canonical definitions are not equal.\nFirst: %s\nSecond: %s\n",
			canonicalJson1, canonicalJson2)
	}
	return equal, nil
}

type containerDefinitions []*ecs.ContainerDefinition

func (cd containerDefinitions) Reduce(isAWSVPC bool) error {
	for i, def := range cd {
		// Deal with special fields which have defaults
		if def.Cpu != nil && *def.Cpu == 0 {
			def.Cpu = nil
		}
		if def.Essential == nil {
			def.Essential = aws.Bool(true)
		}
		for j, pm := range def.PortMappings {
			if pm.Protocol != nil && *pm.Protocol == "tcp" {
				cd[i].PortMappings[j].Protocol = nil
			}
			if pm.HostPort != nil && *pm.HostPort == 0 {
				cd[i].PortMappings[j].HostPort = nil
			}
			if isAWSVPC && cd[i].PortMappings[j].HostPort == nil {
				cd[i].PortMappings[j].HostPort = cd[i].PortMappings[j].ContainerPort
			}
		}

		// Deal with fields which may be re-ordered in the API
		sort.Slice(def.Environment, func(i, j int) bool {
			return aws.StringValue(def.Environment[i].Name) < aws.StringValue(def.Environment[j].Name)
		})

		// Create a mutable copy
		defCopy, err := copystructure.Copy(def)
		if err != nil {
			return err
		}

		definition := reflect.ValueOf(defCopy).Elem()
		for i := 0; i < definition.NumField(); i++ {
			sf := definition.Field(i)

			// Set all empty slices to nil
			if sf.Kind() == reflect.Slice {
				if sf.IsValid() && !sf.IsNil() && sf.Len() == 0 {
					sf.Set(reflect.Zero(sf.Type()))
				}
			}
		}
		iface := definition.Interface().(ecs.ContainerDefinition)
		cd[i] = &iface
	}
	return nil
}
