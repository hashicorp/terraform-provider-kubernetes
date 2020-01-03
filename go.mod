module github.com/terraform-providers/terraform-provider-kubernetes

require (
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	github.com/Azure/go-autorest/autorest v0.1.0 // indirect
	github.com/frankban/quicktest v1.4.2 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gophercloud/gophercloud v0.3.1-0.20190807175045-25a84d593c97 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hil v0.0.0-20190212132231-97b3a9cdfa93 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.3.0
	github.com/hashicorp/vault v1.1.2 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/keybase/go-crypto v0.0.0-20190523171820-b785b22cc757 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pierrec/lz4 v2.3.0+incompatible // indirect
	github.com/robfig/cron v1.2.0
	github.com/terraform-providers/terraform-provider-aws v1.60.0
	github.com/terraform-providers/terraform-provider-google v0.0.0-20191008212436-363f2d283518
	github.com/terraform-providers/terraform-provider-random v0.0.0-20190925210718-83518d96ae4f // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	k8s.io/api v0.0.0-20191025225708-5524a3672fbb
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20191025230902-aa872b06629d
)

go 1.13

// These transitive dependencies have invalid pseudo-versions. Override them
// to the correct pseudo-version (https://tip.golang.org/doc/go1.13#version-validation).
// These can be removed once our dependencies fix their go.mod files to use the
// correct pseudo-versions.
replace github.com/keybase/go-crypto v0.0.0-20190523171820-b785b22cc757 => github.com/keybase/go-crypto v0.0.0-20190416182011-b785b22cc757

replace github.com/Azure/go-autorest v11.1.2+incompatible => github.com/Azure/go-autorest v0.0.0-20190419221215-22f92829cda7

replace k8s.io/client-go v12.0.0+incompatible => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
