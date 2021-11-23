module github.com/hashicorp/terraform-provider-kubernetes

require (
	cloud.google.com/go/storage v1.14.0 // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/aws/aws-sdk-go v1.38.20 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/getkin/kin-openapi v0.66.0
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.16.0
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.1
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/hcl/v2 v2.9.1
	github.com/hashicorp/terraform-exec v0.14.0
	github.com/hashicorp/terraform-json v0.12.0
	github.com/hashicorp/terraform-plugin-go v0.3.1
	github.com/hashicorp/terraform-plugin-mux v0.2.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.7.0
	github.com/hashicorp/terraform-plugin-test/v2 v2.2.1
	github.com/hashicorp/yamux v0.0.0-20210316155119-a95892c5f864 // indirect
	github.com/jinzhu/copier v0.2.9
	github.com/klauspost/compress v1.12.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
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
	github.com/zclconf/go-cty v1.8.4
	go.starlark.net v0.0.0-20210406145628-7a1108eaa012 // indirect
	golang.org/x/mod v0.4.2
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/term v0.0.0-20210406210042-72f3dc4e9b72 // indirect
	golang.org/x/tools v0.1.1-0.20210302220138-2ac05c832e1a // indirect
	google.golang.org/api v0.44.0 // indirect
	google.golang.org/genproto v0.0.0-20210415145412-64678f1ae2d5 // indirect
	google.golang.org/grpc v1.37.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.21.7
	k8s.io/apiextensions-apiserver v0.21.7
	k8s.io/apimachinery v0.21.7
	k8s.io/client-go v0.21.7
	k8s.io/kube-aggregator v0.21.7
	k8s.io/kubectl v0.21.7
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176 // indirect
)

require (
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
)

// kustomize needs to be kept in sync with the cli-runtime.
// go-openapi needs to be locked at a version that is compatible with kustomize
replace (
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.7
	k8s.io/client-go => k8s.io/client-go v0.21.7 // /Users/alex/work/client-go
	sigs.k8s.io/kustomize/pkg/transformers => ./vendor/k8s.io/cli-runtime/pkg/kustomize/k8sdeps/transformer
	sigs.k8s.io/kustomize/pkg/transformers/config => ./vendor/k8s.io/cli-runtime/pkg/kustomize/k8sdeps/transformer/config
)

go 1.16
