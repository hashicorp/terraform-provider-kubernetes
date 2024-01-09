package {{ .ResourceConfig.Package }}

import "github.com/hashicorp/terraform-plugin-framework/types"

type {{ .ResourceConfig.Kind }}Model struct {
  {{ .ModelFields }}
}
