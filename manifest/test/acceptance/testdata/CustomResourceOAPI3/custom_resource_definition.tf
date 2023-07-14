# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_manifest" "test_crd" {

  manifest = {
    "apiVersion" = "apiextensions.k8s.io/v1"
    "kind"       = "CustomResourceDefinition"
    "metadata" = {
      "name" = "${var.plural}.${var.group}"
    }
    "spec" = {
      "group" = var.group
      "names" = {
        "categories" = [
          "all",
        ]
        "kind"     = var.kind
        "listKind" = "${var.kind}List"
        "plural"   = var.plural
        "singular" = lower(var.kind)
      }
      "scope" = "Namespaced"
      "versions" = [
        {
          "additionalPrinterColumns" = [
            {
              "description" = "Team responsible for Postgres cluster"
              "jsonPath"    = ".spec.teamId"
              "name"        = "Team"
              "type"        = "string"
            },
            {
              "description" = "PostgreSQL version"
              "jsonPath"    = ".spec.postgresql.version"
              "name"        = "Version"
              "type"        = "string"
            },
            {
              "description" = "Number of Pods per Postgres cluster"
              "jsonPath"    = ".spec.numberOfInstances"
              "name"        = "Pods"
              "type"        = "integer"
            },
            {
              "description" = "Size of the bound volume"
              "jsonPath"    = ".spec.volume.size"
              "name"        = "Volume"
              "type"        = "string"
            },
            {
              "description" = "Requested CPU for Postgres containers"
              "jsonPath"    = ".spec.resources.requests.cpu"
              "name"        = "CPU-Request"
              "type"        = "string"
            },
            {
              "description" = "Requested memory for Postgres containers"
              "jsonPath"    = ".spec.resources.requests.memory"
              "name"        = "Memory-Request"
              "type"        = "string"
            },
            {
              "jsonPath" = ".metadata.creationTimestamp"
              "name"     = "Age"
              "type"     = "date"
            },
            {
              "description" = "Current sync status of postgresql resource"
              "jsonPath"    = ".status.PostgresClusterStatus"
              "name"        = "Status"
              "type"        = "string"
            },
          ]
          "name" = var.cr_version
          "schema" = {
            "openAPIV3Schema" = {
              "properties" = {
                "apiVersion" = {
                  "enum" = [
                    "${var.group}/${var.cr_version}",
                  ]
                  "type" = "string"
                }
                "kind" = {
                  "enum" = [
                    var.kind,
                  ]
                  "type" = "string"
                }
                "spec" = {
                  "properties" = {
                    "additionalVolumes" = {
                      "items" = {
                        "properties" = {
                          "mountPath" = {
                            "type" = "string"
                          }
                          "name" = {
                            "type" = "string"
                          }
                          "subPath" = {
                            "type" = "string"
                          }
                          "targetContainers" = {
                            "items" = {
                              "type" = "string"
                            }
                            "nullable" = true
                            "type"     = "array"
                          }
                          "volumeSource" = {
                            "type"                                 = "object"
                            "x-kubernetes-preserve-unknown-fields" = true
                          }
                        }
                        "required" = [
                          "name",
                          "mountPath",
                          "volumeSource",
                        ]
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "allowedSourceRanges" = {
                      "items" = {
                        "pattern" = "^(\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5])\\.(\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5])\\.(\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5])\\.(\\d|[1-9]\\d|1\\d\\d|2[0-4]\\d|25[0-5])\\/(\\d|[1-2]\\d|3[0-2])$"
                        "type"    = "string"
                      }
                      "nullable" = true
                      "type"     = "array"
                    }
                    "clone" = {
                      "properties" = {
                        "cluster" = {
                          "type" = "string"
                        }
                        "s3_access_key_id" = {
                          "type" = "string"
                        }
                        "s3_endpoint" = {
                          "type" = "string"
                        }
                        "s3_force_path_style" = {
                          "type" = "boolean"
                        }
                        "s3_secret_access_key" = {
                          "type" = "string"
                        }
                        "s3_wal_path" = {
                          "type" = "string"
                        }
                        "timestamp" = {
                          "pattern" = "^([0-9]+)-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]|60)(\\.[0-9]+)?(([+-]([01][0-9]|2[0-3]):[0-5][0-9]))$"
                          "type"    = "string"
                        }
                        "uid" = {
                          "format" = "uuid"
                          "type"   = "string"
                        }
                      }
                      "required" = [
                        "cluster",
                      ]
                      "type" = "object"
                    }
                    "connectionPooler" = {
                      "properties" = {
                        "dockerImage" = {
                          "type" = "string"
                        }
                        "maxDBConnections" = {
                          "type" = "integer"
                        }
                        "mode" = {
                          "enum" = [
                            "session",
                            "transaction",
                          ]
                          "type" = "string"
                        }
                        "numberOfInstances" = {
                          "minimum" = 2
                          "type"    = "integer"
                        }
                        "resources" = {
                          "properties" = {
                            "limits" = {
                              "properties" = {
                                "cpu" = {
                                  "pattern" = "^(\\d+m|\\d+(\\.\\d{1,3})?)$"
                                  "type"    = "string"
                                }
                                "memory" = {
                                  "pattern" = "^(\\d+(e\\d+)?|\\d+(\\.\\d+)?(e\\d+)?[EPTGMK]i?)$"
                                  "type"    = "string"
                                }
                              }
                              "required" = [
                                "cpu",
                                "memory",
                              ]
                              "type" = "object"
                            }
                            "requests" = {
                              "properties" = {
                                "cpu" = {
                                  "pattern" = "^(\\d+m|\\d+(\\.\\d{1,3})?)$"
                                  "type"    = "string"
                                }
                                "memory" = {
                                  "pattern" = "^(\\d+(e\\d+)?|\\d+(\\.\\d+)?(e\\d+)?[EPTGMK]i?)$"
                                  "type"    = "string"
                                }
                              }
                              "required" = [
                                "cpu",
                                "memory",
                              ]
                              "type" = "object"
                            }
                          }
                          "required" = [
                            "requests",
                            "limits",
                          ]
                          "type" = "object"
                        }
                        "schema" = {
                          "type" = "string"
                        }
                        "user" = {
                          "type" = "string"
                        }
                      }
                      "type" = "object"
                    }
                    "databases" = {
                      "additionalProperties" = {
                        "type" = "string"
                      }
                      "type" = "object"
                    }
                    "dockerImage" = {
                      "type" = "string"
                    }
                    "enableConnectionPooler" = {
                      "type" = "boolean"
                    }
                    "enableLogicalBackup" = {
                      "type" = "boolean"
                    }
                    "enableMasterLoadBalancer" = {
                      "type" = "boolean"
                    }
                    "enableReplicaConnectionPooler" = {
                      "type" = "boolean"
                    }
                    "enableReplicaLoadBalancer" = {
                      "type" = "boolean"
                    }
                    "enableShmVolume" = {
                      "type" = "boolean"
                    }
                    "initContainers" = {
                      "items" = {
                        "type"                                 = "object"
                        "x-kubernetes-preserve-unknown-fields" = true
                      }
                      "nullable" = true
                      "type"     = "array"
                    }
                    "init_containers" = {
                      "items" = {
                        "type"                                 = "object"
                        "x-kubernetes-preserve-unknown-fields" = true
                      }
                      "nullable" = true
                      "type"     = "array"
                    }
                    "logicalBackupSchedule" = {
                      "pattern" = "^(\\d+|\\*)(/\\d+)?(\\s+(\\d+|\\*)(/\\d+)?){4}$"
                      "type"    = "string"
                    }
                    "maintenanceWindows" = {
                      "items" = {
                        "pattern" = "^\\ *((Mon|Tue|Wed|Thu|Fri|Sat|Sun):(2[0-3]|[01]?\\d):([0-5]?\\d)|(2[0-3]|[01]?\\d):([0-5]?\\d))-((Mon|Tue|Wed|Thu|Fri|Sat|Sun):(2[0-3]|[01]?\\d):([0-5]?\\d)|(2[0-3]|[01]?\\d):([0-5]?\\d))\\ *$"
                        "type"    = "string"
                      }
                      "type" = "array"
                    }
                    "nodeAffinity" = {
                      "properties" = {
                        "preferredDuringSchedulingIgnoredDuringExecution" = {
                          "items" = {
                            "properties" = {
                              "preference" = {
                                "properties" = {
                                  "matchExpressions" = {
                                    "items" = {
                                      "properties" = {
                                        "key" = {
                                          "type" = "string"
                                        }
                                        "operator" = {
                                          "type" = "string"
                                        }
                                        "values" = {
                                          "items" = {
                                            "type" = "string"
                                          }
                                          "type" = "array"
                                        }
                                      }
                                      "required" = [
                                        "key",
                                        "operator",
                                      ]
                                      "type" = "object"
                                    }
                                    "type" = "array"
                                  }
                                  "matchFields" = {
                                    "items" = {
                                      "properties" = {
                                        "key" = {
                                          "type" = "string"
                                        }
                                        "operator" = {
                                          "type" = "string"
                                        }
                                        "values" = {
                                          "items" = {
                                            "type" = "string"
                                          }
                                          "type" = "array"
                                        }
                                      }
                                      "required" = [
                                        "key",
                                        "operator",
                                      ]
                                      "type" = "object"
                                    }
                                    "type" = "array"
                                  }
                                }
                                "type" = "object"
                              }
                              "weight" = {
                                "format" = "int32"
                                "type"   = "integer"
                              }
                            }
                            "required" = [
                              "weight",
                              "preference",
                            ]
                            "type" = "object"
                          }
                          "type" = "array"
                        }
                        "requiredDuringSchedulingIgnoredDuringExecution" = {
                          "properties" = {
                            "nodeSelectorTerms" = {
                              "items" = {
                                "properties" = {
                                  "matchExpressions" = {
                                    "items" = {
                                      "properties" = {
                                        "key" = {
                                          "type" = "string"
                                        }
                                        "operator" = {
                                          "type" = "string"
                                        }
                                        "values" = {
                                          "items" = {
                                            "type" = "string"
                                          }
                                          "type" = "array"
                                        }
                                      }
                                      "required" = [
                                        "key",
                                        "operator",
                                      ]
                                      "type" = "object"
                                    }
                                    "type" = "array"
                                  }
                                  "matchFields" = {
                                    "items" = {
                                      "properties" = {
                                        "key" = {
                                          "type" = "string"
                                        }
                                        "operator" = {
                                          "type" = "string"
                                        }
                                        "values" = {
                                          "items" = {
                                            "type" = "string"
                                          }
                                          "type" = "array"
                                        }
                                      }
                                      "required" = [
                                        "key",
                                        "operator",
                                      ]
                                      "type" = "object"
                                    }
                                    "type" = "array"
                                  }
                                }
                                "type" = "object"
                              }
                              "type" = "array"
                            }
                          }
                          "required" = [
                            "nodeSelectorTerms",
                          ]
                          "type" = "object"
                        }
                      }
                      "type" = "object"
                    }
                    "numberOfInstances" = {
                      "minimum" = 0
                      "type"    = "integer"
                    }
                    "patroni" = {
                      "properties" = {
                        "initdb" = {
                          "additionalProperties" = {
                            "type" = "string"
                          }
                          "type" = "object"
                        }
                        "loop_wait" = {
                          "type" = "integer"
                        }
                        "maximum_lag_on_failover" = {
                          "type" = "integer"
                        }
                        "pg_hba" = {
                          "items" = {
                            "type" = "string"
                          }
                          "type" = "array"
                        }
                        "retry_timeout" = {
                          "type" = "integer"
                        }
                        "slots" = {
                          "additionalProperties" = {
                            "additionalProperties" = {
                              "type" = "string"
                            }
                            "type" = "object"
                          }
                          "type" = "object"
                        }
                        "synchronous_mode" = {
                          "type" = "boolean"
                        }
                        "synchronous_mode_strict" = {
                          "type" = "boolean"
                        }
                        "ttl" = {
                          "type" = "integer"
                        }
                      }
                      "type" = "object"
                    }
                    "podAnnotations" = {
                      "additionalProperties" = {
                        "type" = "string"
                      }
                      "type" = "object"
                    }
                    "podPriorityClassName" = {
                      "type" = "string"
                    }
                    "pod_priority_class_name" = {
                      "type" = "string"
                    }
                    "postgresql" = {
                      "properties" = {
                        "parameters" = {
                          "additionalProperties" = {
                            "type" = "string"
                          }
                          "type" = "object"
                        }
                        "version" = {
                          "enum" = [
                            "9.3",
                            "9.4",
                            "9.5",
                            "9.6",
                            "10",
                            "11",
                            "12",
                            "13",
                          ]
                          "type" = "string"
                        }
                      }
                      "required" = [
                        "version",
                      ]
                      "type" = "object"
                    }
                    "preparedDatabases" = {
                      "additionalProperties" = {
                        "properties" = {
                          "defaultUsers" = {
                            "type" = "boolean"
                          }
                          "extensions" = {
                            "additionalProperties" = {
                              "type" = "string"
                            }
                            "type" = "object"
                          }
                          "schemas" = {
                            "additionalProperties" = {
                              "properties" = {
                                "defaultRoles" = {
                                  "type" = "boolean"
                                }
                                "defaultUsers" = {
                                  "type" = "boolean"
                                }
                              }
                              "type" = "object"
                            }
                            "type" = "object"
                          }
                        }
                        "type" = "object"
                      }
                      "type" = "object"
                    }
                    "replicaLoadBalancer" = {
                      "type" = "boolean"
                    }
                    "resources" = {
                      "properties" = {
                        "limits" = {
                          "properties" = {
                            "cpu" = {
                              "pattern" = "^(\\d+m|\\d+(\\.\\d{1,3})?)$"
                              "type"    = "string"
                            }
                            "memory" = {
                              "pattern" = "^(\\d+(e\\d+)?|\\d+(\\.\\d+)?(e\\d+)?[EPTGMK]i?)$"
                              "type"    = "string"
                            }
                          }
                          "required" = [
                            "cpu",
                            "memory",
                          ]
                          "type" = "object"
                        }
                        "requests" = {
                          "properties" = {
                            "cpu" = {
                              "pattern" = "^(\\d+m|\\d+(\\.\\d{1,3})?)$"
                              "type"    = "string"
                            }
                            "memory" = {
                              "pattern" = "^(\\d+(e\\d+)?|\\d+(\\.\\d+)?(e\\d+)?[EPTGMK]i?)$"
                              "type"    = "string"
                            }
                          }
                          "required" = [
                            "cpu",
                            "memory",
                          ]
                          "type" = "object"
                        }
                      }
                      "required" = [
                        "requests",
                        "limits",
                      ]
                      "type" = "object"
                    }
                    "schedulerName" = {
                      "type" = "string"
                    }
                    "serviceAnnotations" = {
                      "additionalProperties" = {
                        "type" = "string"
                      }
                      "type" = "object"
                    }
                    "sidecars" = {
                      "items" = {
                        "type"                                 = "object"
                        "x-kubernetes-preserve-unknown-fields" = true
                      }
                      "nullable" = true
                      "type"     = "array"
                    }
                    "spiloFSGroup" = {
                      "type" = "integer"
                    }
                    "spiloRunAsGroup" = {
                      "type" = "integer"
                    }
                    "spiloRunAsUser" = {
                      "type" = "integer"
                    }
                    "standby" = {
                      "properties" = {
                        "s3_wal_path" = {
                          "type" = "string"
                        }
                      }
                      "required" = [
                        "s3_wal_path",
                      ]
                      "type" = "object"
                    }
                    "teamId" = {
                      "type" = "string"
                    }
                    "tls" = {
                      "properties" = {
                        "caFile" = {
                          "type" = "string"
                        }
                        "caSecretName" = {
                          "type" = "string"
                        }
                        "certificateFile" = {
                          "type" = "string"
                        }
                        "privateKeyFile" = {
                          "type" = "string"
                        }
                        "secretName" = {
                          "type" = "string"
                        }
                      }
                      "required" = [
                        "secretName",
                      ]
                      "type" = "object"
                    }
                    "tolerations" = {
                      "items" = {
                        "properties" = {
                          "effect" = {
                            "enum" = [
                              "NoExecute",
                              "NoSchedule",
                              "PreferNoSchedule",
                            ]
                            "type" = "string"
                          }
                          "key" = {
                            "type" = "string"
                          }
                          "operator" = {
                            "enum" = [
                              "Equal",
                              "Exists",
                            ]
                            "type" = "string"
                          }
                          "tolerationSeconds" = {
                            "type" = "integer"
                          }
                          "value" = {
                            "type" = "string"
                          }
                        }
                        "required" = [
                          "key",
                          "operator",
                          "effect",
                        ]
                        "type" = "object"
                      }
                      "type" = "array"
                    }
                    "useLoadBalancer" = {
                      "type" = "boolean"
                    }
                    "users" = {
                      "additionalProperties" = {
                        "description" = "Role flags specified here must not contradict each other"
                        "items" = {
                          "enum" = [
                            "bypassrls",
                            "BYPASSRLS",
                            "nobypassrls",
                            "NOBYPASSRLS",
                            "createdb",
                            "CREATEDB",
                            "nocreatedb",
                            "NOCREATEDB",
                            "createrole",
                            "CREATEROLE",
                            "nocreaterole",
                            "NOCREATEROLE",
                            "inherit",
                            "INHERIT",
                            "noinherit",
                            "NOINHERIT",
                            "login",
                            "LOGIN",
                            "nologin",
                            "NOLOGIN",
                            "replication",
                            "REPLICATION",
                            "noreplication",
                            "NOREPLICATION",
                            "superuser",
                            "SUPERUSER",
                            "nosuperuser",
                            "NOSUPERUSER",
                          ]
                          "type" = "string"
                        }
                        "nullable" = true
                        "type"     = "array"
                      }
                      "type" = "object"
                    }
                    "volume" = {
                      "properties" = {
                        "iops" = {
                          "type" = "integer"
                        }
                        "size" = {
                          "pattern" = "^(\\d+(e\\d+)?|\\d+(\\.\\d+)?(e\\d+)?[EPTGMK]i?)$"
                          "type"    = "string"
                        }
                        "storageClass" = {
                          "type" = "string"
                        }
                        "subPath" = {
                          "type" = "string"
                        }
                        "throughput" = {
                          "type" = "integer"
                        }
                      }
                      "required" = [
                        "size",
                      ]
                      "type" = "object"
                    }
                  }
                  "required" = [
                    "numberOfInstances",
                    "teamId",
                    "postgresql",
                    "volume",
                  ]
                  "type" = "object"
                }
                "status" = {
                  "additionalProperties" = {
                    "type" = "string"
                  }
                  "type" = "object"
                }
              }
              "required" = [
                "kind",
                "apiVersion",
                "spec",
              ]
              "type"                                 = "object"
              "x-kubernetes-preserve-unknown-fields" = true
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
