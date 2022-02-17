module github.com/hashicorp/terraform-provider-kubernetes

require (
	cloud.google.com/go/storage v1.14.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/aws/aws-sdk-go v1.38.20 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/getkin/kin-openapi v0.66.0
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/google/go-cmp v0.5.7
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-hclog v1.1.0
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/hcl/v2 v2.9.1
	github.com/hashicorp/terraform-exec v0.15.0
	github.com/hashicorp/terraform-json v0.13.0
	github.com/hashicorp/terraform-plugin-go v0.7.1
	github.com/hashicorp/terraform-plugin-mux v0.2.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.10.0
	github.com/hashicorp/terraform-plugin-test/v2 v2.2.1
	github.com/hashicorp/yamux v0.0.0-20210316155119-a95892c5f864 // indirect
	github.com/jinzhu/copier v0.2.9
	github.com/klauspost/compress v1.12.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/robfig/cron v1.2.0
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	github.com/zclconf/go-cty v1.9.1
	go.starlark.net v0.0.0-20210406145628-7a1108eaa012 // indirect
	golang.org/x/mod v0.4.2
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/term v0.0.0-20210406210042-72f3dc4e9b72 // indirect
	google.golang.org/api v0.44.0 // indirect
	google.golang.org/grpc v1.44.0
	k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/kube-aggregator v0.22.4
	k8s.io/kubectl v0.22.4
)

// kustomize needs to be kept in sync with the cli-runtime.
// go-openapi needs to be locked at a version that is compatible with kustomize
replace (
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
	sigs.k8s.io/kustomize/pkg/transformers => ./vendor/k8s.io/cli-runtime/pkg/kustomize/k8sdeps/transformer
	sigs.k8s.io/kustomize/pkg/transformers/config => ./vendor/k8s.io/cli-runtime/pkg/kustomize/k8sdeps/transformer/config
)

go 1.16
