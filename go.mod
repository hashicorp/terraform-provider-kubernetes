module github.com/hashicorp/terraform-provider-kubernetes

require (
	github.com/Azure/go-autorest/autorest v0.9.2 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1-0.20191028180845-3492b2aff503 // indirect
	github.com/frankban/quicktest v1.4.2 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gophercloud/gophercloud v0.3.1-0.20190807175045-25a84d593c97 // indirect
	github.com/hashicorp/go-getter v1.4.2-0.20200106182914-9813cbd4eb02 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl/v2 v2.3.0 // indirect
	github.com/hashicorp/terraform-config-inspect v0.0.0-20191212124732-c6ae6269b9d7 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.7.0
	github.com/hashicorp/vault v1.1.2 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/keybase/go-crypto v0.0.0-20190416182011-b785b22cc757 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pierrec/lz4 v2.3.0+incompatible // indirect
	github.com/robfig/cron v1.2.0
	github.com/terraform-providers/terraform-provider-aws v1.60.1-0.20191010190908-1261a98537f2
	github.com/terraform-providers/terraform-provider-google v1.20.1-0.20191008212436-363f2d283518
	github.com/terraform-providers/terraform-provider-random v1.3.2-0.20190925210718-83518d96ae4f // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	k8s.io/api v0.16.12
	k8s.io/apimachinery v0.16.12
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20191025230902-aa872b06629d
	k8s.io/kubectl v0.16.12
)

// Override invalid go-autorest pseudo-version. This can be removed once
// all transitive dependencies on go-autorest use correct pseudo-versions.
// See https://tip.golang.org/doc/go1.13#version-validation
// and https://github.com/Azure/go-autorest/issues/481
replace (
	github.com/Azure/go-autorest v11.1.2+incompatible => github.com/Azure/go-autorest v12.1.0+incompatible
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
)

go 1.14
