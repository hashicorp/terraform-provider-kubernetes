package loggertest

import (
	"encoding/json"
	"fmt"
	"io"
)

func MultilineJSONDecode(data io.Reader) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	dec := json.NewDecoder(data)

	for {
		var entry map[string]interface{}

		err := dec.Decode(&entry)

		if err == io.EOF {
			break
		}

		if err != nil {
			return result, fmt.Errorf("unable to decode JSON: %s", err)
		}

		result = append(result, entry)
	}

	return result, nil
}
