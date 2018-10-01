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

func ecsContainerDefinitionsAreEquivalent(def1, def2 string) (bool, error) {
	var obj1 containerDefinitions
	err := json.Unmarshal([]byte(def1), &obj1)
	if err != nil {
		return false, err
	}
	err = obj1.Reduce()
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
	err = obj2.Reduce()
	if err != nil {
		return false, err
	}

	canonicalJson2, err := jsonutil.BuildJSON(obj2)
	if err != nil {
		return false, err
	}

	equal := bytes.Compare(canonicalJson1, canonicalJson2) == 0
	if !equal {
		log.Printf("[DEBUG] Canonical definitions are not equal.\nFirst: %s\nSecond: %s\n",
			canonicalJson1, canonicalJson2)
	}
	return equal, nil
}

type containerDefinitions []*ecs.ContainerDefinition

func (cd containerDefinitions) Reduce() error {
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
		}

		// Deal with fields which may be re-ordered in the API
		sort.Slice(def.Environment, func(i, j int) bool {
			return *def.Environment[i].Name < *def.Environment[j].Name
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
