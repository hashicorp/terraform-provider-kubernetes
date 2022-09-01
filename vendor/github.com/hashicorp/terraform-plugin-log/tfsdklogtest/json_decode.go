package tfsdklogtest

import (
	"io"

	"github.com/hashicorp/terraform-plugin-log/internal/loggertest"
)

// MultilineJSONDecode supports decoding the output of a JSON logger into a
// slice of maps, with each element representing a log entry.
func MultilineJSONDecode(data io.Reader) ([]map[string]interface{}, error) {
	return loggertest.MultilineJSONDecode(data)
}
