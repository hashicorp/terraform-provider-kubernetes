# These variable declarations are only used for interactive testing.
# The test code will template in different variable declarations with a default value when running the test.
#
# To set values for interactive runs, create a var-file and set values in it. 
# If the name of the var-file ends in '.auto.tfvars' (e.g. myvalues.auto.tfvars) 
# it will be automatically picked up and used by Terraform.
#
# DO NOT check in any files named *.auto.tfvars when making changes to tests.

variable "name" {
  type = string
}

variable "namespace" {
  type = string
}

variable "kind" {
  type = string
}

variable "group" {
  type = string
}
variable "cr_version" {
  type = string
}

variable "plural" {
  type = string
}
