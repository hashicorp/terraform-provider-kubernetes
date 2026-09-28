# =============================================================================
# kubernetes_storage_class_v1 — demo configuration for a kind cluster
#
# This file explores every supported attribute of the resource.
# It creates four storage classes showing the different shapes/use-cases.
# All classes use the "rancher.io/local-path" provisioner which ships with
# kind by default, so no extra provisioner installation is required.
# =============================================================================

# ---------------------------------------------------------------------------
# 1. MINIMAL — only the required attribute (storage_provisioner).
#    All other fields will use their defaults:
#      reclaim_policy       = "Delete"
#      volume_binding_mode  = "Immediate"
#      allow_volume_expansion = true
# ---------------------------------------------------------------------------
resource "kubernetes_storage_class_v1" "minimal" {
  metadata {
    name = "tf-demo-minimal"
  }

  storage_provisioner = "rancher.io/local-path"
}

# ---------------------------------------------------------------------------
# 2. FULL — every attribute explicitly set so you can see what each one does.
# ---------------------------------------------------------------------------
resource "kubernetes_storage_class_v1" "full" {
  metadata {
    name = "tf-demo-full"

    # Labels are key/value pairs used for selecting / grouping objects.
    labels = {
      managed-by  = "terraform"
      environment = "dev"
    }

    # Annotations are free-form metadata for tooling (not used for selection).
    annotations = {
      "example.com/note"    = "created-by-tf-demo"
      "example.com/version" = "v1"
    }
  }

  # The in-tree or external provisioner that will create volumes.
  # ForceNew: changing this destroys and recreates the storage class.
  storage_provisioner = "rancher.io/local-path"

  # Provisioner-specific key/value tuning.
  # ForceNew: any change recreates the class.
  parameters = {
    pathType = "DirectoryOrCreate"
  }

  # What happens to the PersistentVolume when the PVC that bound it is deleted.
  #   Delete  — the PV and its backing storage are removed (default).
  #   Retain  — the PV is kept; an admin must reclaim it manually.
  #   Recycle — deprecated; basic scrub then make available again.
  # This field CAN be updated in-place (no ForceNew).
  reclaim_policy = "Delete"

  # When provisioning and binding happen:
  #   Immediate            — as soon as a PVC is created (default).
  #   WaitForFirstConsumer — deferred until a Pod actually uses the PVC,
  #                          respecting node topology constraints.
  # ForceNew: changing this destroys and recreates the storage class.
  volume_binding_mode = "WaitForFirstConsumer"

  # Whether PVCs using this class can be resized after creation.
  # This field CAN be updated in-place (no ForceNew).
  allow_volume_expansion = true

  # Extra options passed to the mount(8) command when volumes are attached.
  # ForceNew: any change recreates the class.
  mount_options = ["noatime", "nodiratime"]
}

# ---------------------------------------------------------------------------
# 3. RETAIN POLICY — demonstrates how Retain keeps volumes after PVC deletion.
#    Useful for databases or any data that must survive accidental PVC removal.
# ---------------------------------------------------------------------------
resource "kubernetes_storage_class_v1" "retain" {
  metadata {
    name = "tf-demo-retain"
    labels = {
      managed-by = "terraform"
    }
  }

  storage_provisioner    = "rancher.io/local-path"
  reclaim_policy         = "Retain"
  allow_volume_expansion = true
}

# ---------------------------------------------------------------------------
# 4. TOPOLOGY-AWARE — restricts dynamic provisioning to specific topology zones.
#    allowed_topologies is relevant for cloud providers (GKE, EKS, AKS) where
#    you need volumes in a specific availability zone.
#    On kind this won't filter anything useful, but the block is valid and
#    lets you observe how the API stores it.
#
#    ForceNew: any change to allowed_topologies recreates the class.
# ---------------------------------------------------------------------------
resource "kubernetes_storage_class_v1" "topology" {
  metadata {
    name = "tf-demo-topology"
    labels = {
      managed-by = "terraform"
    }
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Delete"
  # WaitForFirstConsumer is recommended when using topology constraints so that
  # the scheduler can pick a node first and the volume is created in the same zone.
  volume_binding_mode = "WaitForFirstConsumer"

  # Max 1 allowed_topologies block. Inside it, list one or more
  # match_label_expressions to whitelist topology values.
  allowed_topologies {
    match_label_expressions {
      key    = "topology.kubernetes.io/zone"
      values = ["us-east-1a", "us-east-1b"]
    }
  }
}

# =============================================================================
# OUTPUTS — handy for inspecting state after apply
# =============================================================================
output "minimal_name" {
  description = "Name of the minimal storage class"
  value       = kubernetes_storage_class_v1.minimal.metadata[0].name
}

output "minimal_uid" {
  description = "UID assigned by Kubernetes to the minimal storage class"
  value       = kubernetes_storage_class_v1.minimal.metadata[0].uid
}

output "full_name" {
  description = "Name of the full storage class"
  value       = kubernetes_storage_class_v1.full.metadata[0].name
}

output "full_uid" {
  description = "UID assigned by Kubernetes to the full storage class"
  value       = kubernetes_storage_class_v1.full.metadata[0].uid
}

output "retain_name" {
  description = "Name of the retain-policy storage class"
  value       = kubernetes_storage_class_v1.retain.metadata[0].name
}

output "topology_name" {
  description = "Name of the topology-aware storage class"
  value       = kubernetes_storage_class_v1.topology.metadata[0].name
}
