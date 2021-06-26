## 0.5.0 (June 08, 2021)

ENHANCEMENTS:

* Improved CRD schema support: retrieve OpenAPIv3 schema directly from CRD objects (#210)

BUG FIXES:

* Fix a typo in an error message during "apply" (#220)

## 0.4.1 (May 21, 2021)

BUG FIXES:

* CRDs with multiple versions now correctly handled again (#212)
* empty list and map attributes are now correctly handled (#212)

## 0.4.0 (May 19, 2021)

ENHANCEMENTS:

* Support for non-strucural CRs (#170)

## 0.3.3 (May 03, 2021)

BUG FIXES:
* Opportunistically accept unknown value types (allow setting values from other resources) (#190)

## 0.3.2 (March 19, 2021)

BUG FIXES:
* Don't fail validation when manifest contains unknown values (#171)
* Return meaningful diagnostic in case of invalid OpenAPI definition (instead of panic) (#166)
* Checks credentials against the API at plan time and avoid infinite retry loop (#159)

## 0.3.1 (March 11, 2021)

ENHANCEMENTS:
* provider will now throw an error when used with a Terraform version older than 0.14.8

BUG FIXES:
* fix handling of `token`, `username` and `password` attributes in the provider configuration (#162)
* fix infinite retries on discovery API with invalid credentials (#159)

## 0.3.0 (March 10, 2021)

FEATURES
* provider can now manage Custom Resources as per Kinds installed by their parent CRDs
* uses OpenAPI defitions from the target cluster to enforce resource structure and attribute types

ENHANCEMENTS
* completely refactored on top of the [terraform-plugin-go](https://github.com/hashicorp/terraform-plugin-go) SDK
* validations have been added to ensure manifests only specify a namespace when appropriate

DEPRECATIONS:
* the `server_side_planning` configuration attribute has been removed. All planning now uses the OpenAPI based mechanism.

## 0.2.1 (September 18, 2020)

FEATURES
* Add support for waiting on resource fields to reach a certain value (#105)
* Add standalone process debug mode (#121)

BUG FIXES
* Fix login with username and password (#113)
* Fix acceptance tests to work with terraform 0.13
* Defer client initialisation to better cope with transient incomplete client configuration

## 0.2.0 (August 26, 2020)

FEATURES
  * Add wait_for block to kubernetes_manifest resource (#95)

ENHANCEMENTS
  * Published to the Terraform registry

BUG FIXES

## 0.1.0 (June 26, 2020)
