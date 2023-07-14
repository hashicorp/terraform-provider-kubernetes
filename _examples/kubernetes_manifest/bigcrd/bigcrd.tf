# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


resource "kubernetes_manifest" "workspaces_app_terraform_io_crd" {

  manifest = {
    "apiVersion" = "apiextensions.k8s.io/v1"
    "kind"       = "CustomResourceDefinition"
    "metadata" = {
      "name" = "workspaces.app.terraform.io"
    }
    "spec" = {
      "group" = "app.terraform.io"
      "names" = {
        "kind"     = "Workspace"
        "listKind" = "WorkspaceList"
        "plural"   = "workspaces"
        "singular" = "workspace"
      }
      "scope" = "Namespaced"
      "versions" = [
        {
          "name" = "v1alpha1"
          "schema" = {
            "openAPIV3Schema" = {
              "description" = "Workspace is the Schema for the workspaces API"
              "properties" = {
                "apiVersion" = {
                  "description" = "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources"
                  "type"        = "string"
                }
                "kind" = {
                  "description" = "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"
                  "type"        = "string"
                }
                "metadata" = {
                  "type" = "object"
                }
                "spec" = {
                  "description" = "WorkspaceSpec defines the desired state of Workspace"
                  "properties" = {
                    "module" = {
                      "description" = "Module source and version to use"
                      "nullable"    = true
                      "properties" = {
                        "source" = {
                          "description" = "Any remote module source (version control, registry)"
                          "type"        = "string"
                        }
                        "version" = {
                          "description" = "Module version for registry modules"
                          "type"        = "string"
                        }
                      }
                      "required" = [
                        "source",
                      ]
                      "type" = "object"
                    }
                    "organization" = {
                      "description" = "Terraform Cloud organization"
                      "type"        = "string"
                    }
                    "outputs" = {
                      "description" = "Outputs denote outputs wanted"
                      "items" = {
                        "description" = "OutputSpec specifies which values need to be output"
                        "properties" = {
                          "key" = {
                            "description" = "Output name"
                            "type"        = "string"
                          }
                          "moduleOutputName" = {
                            "description" = "Attribute name in module"
                            "type"        = "string"
                          }
                        }
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "secretsMountPath" = {
                      "description" = "File path within operator pod to load workspace secrets"
                      "type"        = "string"
                    }
                    "sshKeyID" = {
                      "description" = "SSH Key ID. This key must already exist in the TF Cloud organization.  This can either be the user assigned name of the SSH Key, or the system assigned ID."
                      "type"        = "string"
                    }
                    "variables" = {
                      "description" = "Variables as inputs to module"
                      "items" = {
                        "description" = "Variable denotes an input to the module"
                        "properties" = {
                          "environmentVariable" = {
                            "description" = "EnvironmentVariable denotes if this variable should be created as environment variable"
                            "type"        = "boolean"
                          }
                          "hcl" = {
                            "description" = "String input should be an HCL-formatted variable"
                            "type"        = "boolean"
                          }
                          "key" = {
                            "description" = "Variable name"
                            "type"        = "string"
                          }
                          "sensitive" = {
                            "description" = "Variable is a secret and should be retrieved from file"
                            "type"        = "boolean"
                          }
                          "value" = {
                            "description" = "Variable value"
                            "type"        = "string"
                          }
                          "valueFrom" = {
                            "description" = "Source for the variable's value. Cannot be used if value is not empty."
                            "properties" = {
                              "configMapKeyRef" = {
                                "description" = "Selects a key of a ConfigMap."
                                "properties" = {
                                  "key" = {
                                    "description" = "The key to select."
                                    "type"        = "string"
                                  }
                                  "name" = {
                                    "description" = "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?"
                                    "type"        = "string"
                                  }
                                  "optional" = {
                                    "description" = "Specify whether the ConfigMap or its key must be defined"
                                    "type"        = "boolean"
                                  }
                                }
                                "required" = [
                                  "key",
                                ]
                                "type" = "object"
                              }
                              "fieldRef" = {
                                "description" = "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs."
                                "properties" = {
                                  "apiVersion" = {
                                    "description" = "Version of the schema the FieldPath is written in terms of, defaults to \"v1\"."
                                    "type"        = "string"
                                  }
                                  "fieldPath" = {
                                    "description" = "Path of the field to select in the specified API version."
                                    "type"        = "string"
                                  }
                                }
                                "required" = [
                                  "fieldPath",
                                ]
                                "type" = "object"
                              }
                              "resourceFieldRef" = {
                                "description" = "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported."
                                "properties" = {
                                  "containerName" = {
                                    "description" = "Container name: required for volumes, optional for env vars"
                                    "type"        = "string"
                                  }
                                  "divisor" = {
                                    "anyOf" = [
                                      {
                                        "type" = "integer"
                                      },
                                      {
                                        "type" = "string"
                                      },
                                    ]
                                    "description"                = "Specifies the output format of the exposed resources, defaults to \"1\""
                                    "pattern"                    = "^(\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))))?$"
                                    "x-kubernetes-int-or-string" = true
                                  }
                                  "resource" = {
                                    "description" = "Required: resource to select"
                                    "type"        = "string"
                                  }
                                }
                                "required" = [
                                  "resource",
                                ]
                                "type" = "object"
                              }
                              "secretKeyRef" = {
                                "description" = "Selects a key of a secret in the pod's namespace"
                                "properties" = {
                                  "key" = {
                                    "description" = "The key of the secret to select from.  Must be a valid secret key."
                                    "type"        = "string"
                                  }
                                  "name" = {
                                    "description" = "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?"
                                    "type"        = "string"
                                  }
                                  "optional" = {
                                    "description" = "Specify whether the Secret or its key must be defined"
                                    "type"        = "boolean"
                                  }
                                }
                                "required" = [
                                  "key",
                                ]
                                "type" = "object"
                              }
                            }
                            "type" = "object"
                          }
                        }
                        "required" = [
                          "environmentVariable",
                          "key",
                          "sensitive",
                        ]
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "vcs" = {
                      "description" = "Details of the VCS repository we want to connect to the workspace"
                      "nullable"    = true
                      "properties" = {
                        "branch" = {
                          "description" = "The repository branch to use"
                          "type"        = "string"
                        }
                        "ingress_submodules" = {
                          "description" = "Whether submodules should be fetched when cloning the VCS repository (Defaults to false)"
                          "type"        = "boolean"
                        }
                        "repo_identifier" = {
                          "description" = "A reference to your VCS repository in the format org/repo"
                          "type"        = "string"
                        }
                        "token_id" = {
                          "description" = "Token ID of the VCS Connection (OAuth Connection Token) to use"
                          "type"        = "string"
                        }
                      }
                      "required" = [
                        "repo_identifier",
                        "token_id",
                      ]
                      "type" = "object"
                    }
                  }
                  "required" = [
                    "organization",
                    "secretsMountPath",
                  ]
                  "type" = "object"
                }
                "status" = {
                  "description" = "WorkspaceStatus defines the observed state of Workspace"
                  "properties" = {
                    "outputs" = {
                      "description" = "Outputs from state file"
                      "items" = {
                        "description" = "OutputStatus outputs the values of Terraform output"
                        "properties" = {
                          "key" = {
                            "description" = "Attribute name in module"
                            "type"        = "string"
                          }
                          "value" = {
                            "description" = "Value"
                            "type"        = "string"
                          }
                        }
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "runID" = {
                      "description" = "Run ID"
                      "type"        = "string"
                    }
                    "runStatus" = {
                      "description" = "Run Status gets the run status"
                      "type"        = "string"
                    }
                    "workspaceID" = {
                      "description" = "Workspace ID"
                      "type"        = "string"
                    }
                  }
                  "required" = [
                    "runID",
                    "runStatus",
                    "workspaceID",
                  ]
                  "type" = "object"
                }
              }
              "type" = "object"
            }
          }
          "served"  = true
          "storage" = true
          "subresources" = {
            "status" = {}
          }
        },
      ]
    }
  }

}
