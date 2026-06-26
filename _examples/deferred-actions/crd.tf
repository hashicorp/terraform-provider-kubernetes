# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "crd_workspaces" {
  manifest = {
    "apiVersion" = "apiextensions.k8s.io/v1"
    "kind"       = "CustomResourceDefinition"
    "metadata" = {
      "annotations" = {
        "controller-gen.kubebuilder.io/version" = "v0.14.0"
      }
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
          "additionalPrinterColumns" = [
            {
              "jsonPath" = ".status.workspaceID"
              "name"     = "Workspace ID"
              "type"     = "string"
            },
          ]
          "name" = "v1alpha2"
          "schema" = {
            "openAPIV3Schema" = {
              "description" = "Workspace is the Schema for the workspaces API"
              "properties" = {
                "apiVersion" = {
                  "description" = <<-EOT
                  APIVersion defines the versioned schema of this representation of an object.
                  Servers should convert recognized schemas to the latest internal value, and
                  may reject unrecognized values.
                  More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
                  EOT
                  "type"        = "string"
                }
                "kind" = {
                  "description" = <<-EOT
                  Kind is a string value representing the REST resource this object represents.
                  Servers may infer this from the endpoint the client submits requests to.
                  Cannot be updated.
                  In CamelCase.
                  More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
                  EOT
                  "type"        = "string"
                }
                "metadata" = {
                  "type" = "object"
                }
                "spec" = {
                  "description" = "WorkspaceSpec defines the desired state of Workspace."
                  "properties" = {
                    "agentPool" = {
                      "description" = <<-EOT
                      HCP Terraform Agents allow HCP Terraform to communicate with isolated, private, or on-premises infrastructure.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/agents
                      EOT
                      "properties" = {
                        "id" = {
                          "description" = <<-EOT
                          Agent Pool ID.
                          Must match pattern: `^apool-[a-zA-Z0-9]+$`
                          EOT
                          "pattern"     = "^apool-[a-zA-Z0-9]+$"
                          "type"        = "string"
                        }
                        "name" = {
                          "description" = "Agent Pool name."
                          "minLength"   = 1
                          "type"        = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "allowDestroyPlan" = {
                      "default"     = true
                      "description" = <<-EOT
                      Allows a destroy plan to be created and applied.
                      Default: `true`.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#destruction-and-deletion
                      EOT
                      "type"        = "boolean"
                    }
                    "applyMethod" = {
                      "default"     = "manual"
                      "description" = <<-EOT
                      Define either change will be applied automatically(auto) or require an operator to confirm(manual).
                      Must be one of the following values: `auto`, `manual`.
                      Default: `manual`.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#auto-apply-and-manual-apply
                      EOT
                      "pattern"     = "^(auto|manual)$"
                      "type"        = "string"
                    }
                    "description" = {
                      "description" = "Workspace description."
                      "minLength"   = 1
                      "type"        = "string"
                    }
                    "environmentVariables" = {
                      "description" = <<-EOT
                      Terraform Environment variables for all plans and applies in this workspace.
                      Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#environment-variables
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        Variables let you customize configurations, modify Terraform's behavior, and store information like provider credentials.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
                        EOT
                        "properties" = {
                          "description" = {
                            "description" = "Description of the variable."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "hcl" = {
                            "default"     = false
                            "description" = <<-EOT
                            Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime.
                            Default: `false`.
                            EOT
                            "type"        = "boolean"
                          }
                          "name" = {
                            "description" = "Name of the variable."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "sensitive" = {
                            "default"     = false
                            "description" = <<-EOT
                            Sensitive variables are never shown in the UI or API.
                            They may appear in Terraform logs if your configuration is designed to output them.
                            Default: `false`.
                            EOT
                            "type"        = "boolean"
                          }
                          "value" = {
                            "description" = "Value of the variable."
                            "minLength"   = 1
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
                                    "description" = <<-EOT
                                    Name of the referent.
                                    More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                                    TODO: Add other useful fields. apiVersion, kind, uid?
                                    EOT
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
                                "type"                  = "object"
                                "x-kubernetes-map-type" = "atomic"
                              }
                              "secretKeyRef" = {
                                "description" = "Selects a key of a Secret."
                                "properties" = {
                                  "key" = {
                                    "description" = "The key of the secret to select from.  Must be a valid secret key."
                                    "type"        = "string"
                                  }
                                  "name" = {
                                    "description" = <<-EOT
                                    Name of the referent.
                                    More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                                    TODO: Add other useful fields. apiVersion, kind, uid?
                                    EOT
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
                                "type"                  = "object"
                                "x-kubernetes-map-type" = "atomic"
                              }
                            }
                            "type" = "object"
                          }
                        }
                        "required" = [
                          "name",
                        ]
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "executionMode" = {
                      "default"     = "remote"
                      "description" = <<-EOT
                      Define where the Terraform code will be executed.
                      Must be one of the following values: `agent`, `local`, `remote`.
                      Default: `remote`.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#execution-mode
                      EOT
                      "pattern"     = "^(agent|local|remote)$"
                      "type"        = "string"
                    }
                    "name" = {
                      "description" = "Workspace name."
                      "minLength"   = 1
                      "type"        = "string"
                    }
                    "notifications" = {
                      "description" = <<-EOT
                      Notifications allow you to send messages to other applications based on run and workspace events.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        Notifications allow you to send messages to other applications based on run and workspace events.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/notifications
                        EOT
                        "properties" = {
                          "emailAddresses" = {
                            "description" = <<-EOT
                            The list of email addresses that will receive notification emails.
                            It is only available for Terraform Enterprise users. It is not available in HCP Terraform.
                            EOT
                            "items" = {
                              "type" = "string"
                            }
                            "minItems" = 1
                            "type"     = "array"
                          }
                          "emailUsers" = {
                            "description" = "The list of users belonging to the organization that will receive notification emails."
                            "items" = {
                              "type" = "string"
                            }
                            "minItems" = 1
                            "type"     = "array"
                          }
                          "enabled" = {
                            "default"     = true
                            "description" = <<-EOT
                            Whether the notification configuration should be enabled or not.
                            Default: `true`.
                            EOT
                            "type"        = "boolean"
                          }
                          "name" = {
                            "description" = "Notification name."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "token" = {
                            "description" = "The token of the notification."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "triggers" = {
                            "description" = <<-EOT
                            The list of run events that will trigger notifications.
                            Trigger represents the different TFC notifications that can be sent as a run's progress transitions between different states.
                            There are two categories of triggers:
                              - Health Events: `assessment:check_failure`, `assessment:drifted`, `assessment:failed`.
                              - Run Events: `run:applying`, `run:completed`, `run:created`, `run:errored`, `run:needs_attention`, `run:planning`.
                            EOT
                            "items" = {
                              "description" = <<-EOT
                              NotificationTrigger represents the different TFC notifications that can be sent as a run's progress transitions between different states.
                              This must be aligned with go-tfe type `NotificationTriggerType`.
                              Must be one of the following values: `run:applying`, `assessment:check_failure`, `run:completed`, `run:created`, `assessment:drifted`, `run:errored`, `assessment:failed`, `run:needs_attention`, `run:planning`.
                              EOT
                              "enum" = [
                                "run:applying",
                                "assessment:check_failure",
                                "run:completed",
                                "run:created",
                                "assessment:drifted",
                                "run:errored",
                                "assessment:failed",
                                "run:needs_attention",
                                "run:planning",
                              ]
                              "type" = "string"
                            }
                            "minItems" = 1
                            "type"     = "array"
                          }
                          "type" = {
                            "description" = <<-EOT
                            The type of the notification.
                            Must be one of the following values: `email`, `generic`, `microsoft-teams`, `slack`.
                            EOT
                            "enum" = [
                              "email",
                              "generic",
                              "microsoft-teams",
                              "slack",
                            ]
                            "type" = "string"
                          }
                          "url" = {
                            "description" = <<-EOT
                            The URL of the notification.
                            Must match pattern: `^https?://.*`
                            EOT
                            "pattern"     = "^https?://.*"
                            "type"        = "string"
                          }
                        }
                        "required" = [
                          "name",
                          "type",
                        ]
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "organization" = {
                      "description" = <<-EOT
                      Organization name where the Workspace will be created.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/organizations
                      EOT
                      "minLength"   = 1
                      "type"        = "string"
                    }
                    "project" = {
                      "description" = <<-EOT
                      Projects let you organize your workspaces into groups.
                      Default: default organization project.
                      More information:
                        - https://developer.hashicorp.com/terraform/tutorials/cloud/projects
                      EOT
                      "properties" = {
                        "id" = {
                          "description" = <<-EOT
                          Project ID.
                          Must match pattern: `^prj-[a-zA-Z0-9]+$`
                          EOT
                          "pattern"     = "^prj-[a-zA-Z0-9]+$"
                          "type"        = "string"
                        }
                        "name" = {
                          "description" = "Project name."
                          "minLength"   = 1
                          "type"        = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "remoteStateSharing" = {
                      "description" = <<-EOT
                      Remote state access between workspaces.
                      By default, new workspaces in HCP Terraform do not allow other workspaces to access their state.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#accessing-state-from-other-workspaces
                      EOT
                      "properties" = {
                        "allWorkspaces" = {
                          "default"     = false
                          "description" = <<-EOT
                          Allow access to the state for all workspaces within the same organization.
                          Default: `false`.
                          EOT
                          "type"        = "boolean"
                        }
                        "workspaces" = {
                          "description" = "Allow access to the state for specific workspaces within the same organization."
                          "items" = {
                            "description" = <<-EOT
                            ConsumerWorkspace allows access to the state for specific workspaces within the same organization.
                            Only one of the fields `ID` or `Name` is allowed.
                            At least one of the fields `ID` or `Name` is mandatory.
                            More information:
                              - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/state#remote-state-access-controls
                            EOT
                            "properties" = {
                              "id" = {
                                "description" = <<-EOT
                                Consumer Workspace ID.
                                Must match pattern: `^ws-[a-zA-Z0-9]+$`
                                EOT
                                "pattern"     = "^ws-[a-zA-Z0-9]+$"
                                "type"        = "string"
                              }
                              "name" = {
                                "description" = "Consumer Workspace name."
                                "minLength"   = 1
                                "type"        = "string"
                              }
                            }
                            "type" = "object"
                          }
                          "minItems" = 1
                          "type"     = "array"
                        }
                      }
                      "type" = "object"
                    }
                    "runTasks" = {
                      "description" = <<-EOT
                      Run tasks allow HCP Terraform to interact with external systems at specific points in the HCP Terraform run lifecycle.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        Run tasks allow HCP Terraform to interact with external systems at specific points in the HCP Terraform run lifecycle.
                        Only one of the fields `ID` or `Name` is allowed.
                        At least one of the fields `ID` or `Name` is mandatory.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-tasks
                        EOT
                        "properties" = {
                          "enforcementLevel" = {
                            "default"     = "advisory"
                            "description" = <<-EOT
                            Run Task Enforcement Level. Can be one of `advisory` or `mandatory`. Default: `advisory`.
                            Must be one of the following values: `advisory`, `mandatory`
                            Default: `advisory`.
                            EOT
                            "pattern"     = "^(advisory|mandatory)$"
                            "type"        = "string"
                          }
                          "id" = {
                            "description" = <<-EOT
                            Run Task ID.
                            Must match pattern: `^task-[a-zA-Z0-9]+$`
                            EOT
                            "pattern"     = "^task-[a-zA-Z0-9]+$"
                            "type"        = "string"
                          }
                          "name" = {
                            "description" = "Run Task Name."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "stage" = {
                            "default"     = "post_plan"
                            "description" = <<-EOT
                            Run Task Stage.
                            Must be one of the following values: `pre_apply`, `pre_plan`, `post_plan`.
                            Default: `post_plan`.
                            EOT
                            "pattern"     = "^(pre_apply|pre_plan|post_plan)$"
                            "type"        = "string"
                          }
                        }
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "runTriggers" = {
                      "description" = <<-EOT
                      Run triggers allow you to connect this workspace to one or more source workspaces.
                      These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        RunTrigger allows you to connect this workspace to one or more source workspaces.
                        These connections allow runs to queue automatically in this workspace on successful apply of runs in any of the source workspaces.
                        Only one of the fields `ID` or `Name` is allowed.
                        At least one of the fields `ID` or `Name` is mandatory.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/run-triggers
                        EOT
                        "properties" = {
                          "id" = {
                            "description" = <<-EOT
                            Source Workspace ID.
                            Must match pattern: `^ws-[a-zA-Z0-9]+$`
                            EOT
                            "pattern"     = "^ws-[a-zA-Z0-9]+$"
                            "type"        = "string"
                          }
                          "name" = {
                            "description" = "Source Workspace Name."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                        }
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "sshKey" = {
                      "description" = <<-EOT
                      SSH key used to clone Terraform modules.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/ssh-keys
                      EOT
                      "properties" = {
                        "id" = {
                          "description" = <<-EOT
                          SSH key ID.
                          Must match pattern: `^sshkey-[a-zA-Z0-9]+$`
                          EOT
                          "pattern"     = "^sshkey-[a-zA-Z0-9]+$"
                          "type"        = "string"
                        }
                        "name" = {
                          "description" = "SSH key name."
                          "minLength"   = 1
                          "type"        = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "tags" = {
                      "description" = <<-EOT
                      Workspace tags are used to help identify and group together workspaces.
                      Tags must be one or more characters; can include letters, numbers, colons, hyphens, and underscores; and must begin and end with a letter or number.
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        Tags allows you to correlate, organize, and even filter workspaces based on the assigned tags.
                        Tags must be one or more characters; can include letters, numbers, colons, hyphens, and underscores; and must begin and end with a letter or number.
                        Must match pattern: `^[A-Za-z0-9][A-Za-z0-9:_-]*$`
                        EOT
                        "pattern"     = "^[A-Za-z0-9][A-Za-z0-9:_-]*$"
                        "type"        = "string"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "teamAccess" = {
                      "description" = <<-EOT
                      HCP Terraform workspaces can only be accessed by users with the correct permissions.
                      You can manage permissions for a workspace on a per-team basis.
                      When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,
                      with full admin permissions. These teams' access can't be removed from a workspace.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        HCP Terraform workspaces can only be accessed by users with the correct permissions.
                        You can manage permissions for a workspace on a per-team basis.
                        When a workspace is created, only the owners team and teams with the "manage workspaces" permission can access it,
                        with full admin permissions. These teams' access can't be removed from a workspace.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings/access
                        EOT
                        "properties" = {
                          "access" = {
                            "description" = <<-EOT
                            There are two ways to choose which permissions a given team has on a workspace: fixed permission sets, and custom permissions.
                            Must be one of the following values: `admin`, `custom`, `plan`, `read`, `write`.
                            More information:
                              - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#workspace-permissions
                            EOT
                            "pattern"     = "^(admin|custom|plan|read|write)$"
                            "type"        = "string"
                          }
                          "custom" = {
                            "description" = <<-EOT
                            Custom permissions let you assign specific, finer-grained permissions to a team than the broader fixed permission sets provide.
                            More information:
                              - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions#custom-workspace-permissions
                            EOT
                            "properties" = {
                              "runTasks" = {
                                "description" = <<-EOT
                                Manage Workspace Run Tasks.
                                Default: `false`.
                                EOT
                                "type"        = "boolean"
                              }
                              "runs" = {
                                "default"     = "read"
                                "description" = <<-EOT
                                Run access.
                                Must be one of the following values: `apply`, `plan`, `read`.
                                Default: `read`.
                                EOT
                                "pattern"     = "^(apply|plan|read)$"
                                "type"        = "string"
                              }
                              "sentinel" = {
                                "default"     = "none"
                                "description" = <<-EOT
                                Download Sentinel mocks.
                                Must be one of the following values: `none`, `read`.
                                Default: `none`.
                                EOT
                                "pattern"     = "^(none|read)$"
                                "type"        = "string"
                              }
                              "stateVersions" = {
                                "default"     = "none"
                                "description" = <<-EOT
                                State access.
                                Must be one of the following values: `none`, `read`, `read-outputs`, `write`.
                                Default: `none`.
                                EOT
                                "pattern"     = "^(none|read|read-outputs|write)$"
                                "type"        = "string"
                              }
                              "variables" = {
                                "default"     = "none"
                                "description" = <<-EOT
                                Variable access.
                                Must be one of the following values: `none`, `read`, `write`.
                                Default: `none`.
                                EOT
                                "pattern"     = "^(none|read|write)$"
                                "type"        = "string"
                              }
                              "workspaceLocking" = {
                                "default"     = false
                                "description" = <<-EOT
                                Lock/unlock workspace.
                                Default: `false`.
                                EOT
                                "type"        = "boolean"
                              }
                            }
                            "type" = "object"
                          }
                          "team" = {
                            "description" = <<-EOT
                            Team to grant access.
                            More information:
                              - https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams
                            EOT
                            "properties" = {
                              "id" = {
                                "description" = <<-EOT
                                Team ID.
                                Must match pattern: `^team-[a-zA-Z0-9]+$`
                                EOT
                                "pattern"     = "^team-[a-zA-Z0-9]+$"
                                "type"        = "string"
                              }
                              "name" = {
                                "description" = "Team name."
                                "minLength"   = 1
                                "type"        = "string"
                              }
                            }
                            "type" = "object"
                          }
                        }
                        "required" = [
                          "access",
                          "team",
                        ]
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "terraformVariables" = {
                      "description" = <<-EOT
                      Terraform variables for all plans and applies in this workspace.
                      Variables defined within a workspace always overwrite variables from variable sets that have the same type and the same key.
                      More information:
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
                        - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables#terraform-variables
                      EOT
                      "items" = {
                        "description" = <<-EOT
                        Variables let you customize configurations, modify Terraform's behavior, and store information like provider credentials.
                        More information:
                          - https://developer.hashicorp.com/terraform/cloud-docs/workspaces/variables
                        EOT
                        "properties" = {
                          "description" = {
                            "description" = "Description of the variable."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "hcl" = {
                            "default"     = false
                            "description" = <<-EOT
                            Parse this field as HashiCorp Configuration Language (HCL). This allows you to interpolate values at runtime.
                            Default: `false`.
                            EOT
                            "type"        = "boolean"
                          }
                          "name" = {
                            "description" = "Name of the variable."
                            "minLength"   = 1
                            "type"        = "string"
                          }
                          "sensitive" = {
                            "default"     = false
                            "description" = <<-EOT
                            Sensitive variables are never shown in the UI or API.
                            They may appear in Terraform logs if your configuration is designed to output them.
                            Default: `false`.
                            EOT
                            "type"        = "boolean"
                          }
                          "value" = {
                            "description" = "Value of the variable."
                            "minLength"   = 1
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
                                    "description" = <<-EOT
                                    Name of the referent.
                                    More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                                    TODO: Add other useful fields. apiVersion, kind, uid?
                                    EOT
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
                                "type"                  = "object"
                                "x-kubernetes-map-type" = "atomic"
                              }
                              "secretKeyRef" = {
                                "description" = "Selects a key of a Secret."
                                "properties" = {
                                  "key" = {
                                    "description" = "The key of the secret to select from.  Must be a valid secret key."
                                    "type"        = "string"
                                  }
                                  "name" = {
                                    "description" = <<-EOT
                                    Name of the referent.
                                    More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                                    TODO: Add other useful fields. apiVersion, kind, uid?
                                    EOT
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
                                "type"                  = "object"
                                "x-kubernetes-map-type" = "atomic"
                              }
                            }
                            "type" = "object"
                          }
                        }
                        "required" = [
                          "name",
                        ]
                        "type" = "object"
                      }
                      "minItems" = 1
                      "type"     = "array"
                    }
                    "terraformVersion" = {
                      "description" = <<-EOT
                      The version of Terraform to use for this workspace.
                      If not specified, the latest available version will be used.
                      Must match pattern: `^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$`
                      More information:
                        - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-version
                      EOT
                      "pattern"     = "^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
                      "type"        = "string"
                    }
                    "token" = {
                      "description" = "API Token to be used for API calls."
                      "properties" = {
                        "secretKeyRef" = {
                          "description" = "Selects a key of a secret in the workspace's namespace"
                          "properties" = {
                            "key" = {
                              "description" = "The key of the secret to select from.  Must be a valid secret key."
                              "type"        = "string"
                            }
                            "name" = {
                              "description" = <<-EOT
                              Name of the referent.
                              More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                              TODO: Add other useful fields. apiVersion, kind, uid?
                              EOT
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
                          "type"                  = "object"
                          "x-kubernetes-map-type" = "atomic"
                        }
                      }
                      "required" = [
                        "secretKeyRef",
                      ]
                      "type" = "object"
                    }
                    "versionControl" = {
                      "description" = <<-EOT
                      Settings for the workspace's VCS repository, enabling the UI/VCS-driven run workflow.
                      Omit this argument to utilize the CLI-driven and API-driven workflows, where runs are not driven by webhooks on your VCS provider.
                      More information:
                        - https://www.terraform.io/cloud-docs/run/ui
                        - https://www.terraform.io/cloud-docs/vcs
                      EOT
                      "properties" = {
                        "branch" = {
                          "description" = "The repository branch that Run will execute from. This defaults to the repository's default branch (e.g. main)."
                          "minLength"   = 1
                          "type"        = "string"
                        }
                        "oAuthTokenID" = {
                          "description" = <<-EOT
                          The VCS Connection (OAuth Connection + Token) to use.
                          Must match pattern: `^ot-[a-zA-Z0-9]+$`
                          EOT
                          "pattern"     = "^ot-[a-zA-Z0-9]+$"
                          "type"        = "string"
                        }
                        "repository" = {
                          "description" = "A reference to your VCS repository in the format `<organization>/<repository>` where `<organization>` and `<repository>` refer to the organization and repository in your VCS provider."
                          "minLength"   = 1
                          "type"        = "string"
                        }
                        "speculativePlans" = {
                          "default"     = true
                          "description" = <<-EOT
                          Whether this workspace allows automatic speculative plans on PR.
                          Default: `true`.
                          More information:
                            - https://developer.hashicorp.com/terraform/cloud-docs/run/ui#speculative-plans-on-pull-requests
                            - https://developer.hashicorp.com/terraform/cloud-docs/run/remote-operations#speculative-plans
                          EOT
                          "type"        = "boolean"
                        }
                      }
                      "type" = "object"
                    }
                    "workingDirectory" = {
                      "description" = <<-EOT
                      The directory where Terraform will execute, specified as a relative path from the root of the configuration directory.
                      More information:
                        - https://www.terraform.io/cloud-docs/workspaces/settings#terraform-working-directory
                      EOT
                      "minLength"   = 1
                      "type"        = "string"
                    }
                  }
                  "required" = [
                    "name",
                    "organization",
                    "token",
                  ]
                  "type" = "object"
                }
                "status" = {
                  "description" = "WorkspaceStatus defines the observed state of Workspace."
                  "properties" = {
                    "observedGeneration" = {
                      "description" = "Real world state generation."
                      "format"      = "int64"
                      "type"        = "integer"
                    }
                    "plan" = {
                      "description" = "Run status of plan-only/speculative plan that was triggered manually."
                      "properties" = {
                        "id" = {
                          "description" = "Latest plan-only/speculative plan HCP Terraform run ID."
                          "type"        = "string"
                        }
                        "status" = {
                          "description" = "Latest plan-only/speculative plan HCP Terraform run status."
                          "type"        = "string"
                        }
                        "terraformVersion" = {
                          "description" = "The version of Terraform to use for this run."
                          "pattern"     = "^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
                          "type"        = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "runStatus" = {
                      "description" = "Workspace Runs status."
                      "properties" = {
                        "configurationVersion" = {
                          "description" = "The configuration version of this run."
                          "type"        = "string"
                        }
                        "id" = {
                          "description" = "Current(both active and finished) HCP Terraform run ID."
                          "type"        = "string"
                        }
                        "outputRunID" = {
                          "description" = "Run ID of the latest run that could update the outputs."
                          "type"        = "string"
                        }
                        "status" = {
                          "description" = "Current(both active and finished) HCP Terraform run status."
                          "type"        = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "terraformVersion" = {
                      "description" = "Workspace Terraform version."
                      "pattern"     = "^\\d{1}\\.\\d{1,2}\\.\\d{1,2}$"
                      "type"        = "string"
                    }
                    "updateAt" = {
                      "description" = "Workspace last update timestamp."
                      "format"      = "int64"
                      "type"        = "integer"
                    }
                    "variables" = {
                      "description" = "Workspace variables."
                      "items" = {
                        "properties" = {
                          "category" = {
                            "description" = "Category of the variable."
                            "type"        = "string"
                          }
                          "id" = {
                            "description" = "ID of the variable."
                            "type"        = "string"
                          }
                          "name" = {
                            "description" = "Name of the variable."
                            "type"        = "string"
                          }
                          "valueID" = {
                            "description" = "ValueID is a hash of the variable on the CRD end."
                            "type"        = "string"
                          }
                          "versionID" = {
                            "description" = "VersionID is a hash of the variable on the TFC end."
                            "type"        = "string"
                          }
                        }
                        "required" = [
                          "category",
                          "id",
                          "name",
                          "valueID",
                          "versionID",
                        ]
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "workspaceID" = {
                      "description" = "Workspace ID that is managed by the controller."
                      "type"        = "string"
                    }
                  }
                  "required" = [
                    "workspaceID",
                  ]
                  "type" = "object"
                }
              }
              "required" = [
                "spec",
              ]
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
