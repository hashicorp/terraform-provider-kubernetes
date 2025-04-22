schema = "2"

project "terraform-provider-crt-example" {
  team = "terraform"

  slack {
    notification_channel = "C051FAAHL8M" // #feed-terraform-ecosystem-kubernetes-releases
  }

  github {
    organization     = "hashicorp"
    repository       = "terraform-provider-kubernetes"
    release_branches = ["main", "release/**"]
  }
}

event "merge" {
}

event "build" {
  action "build" {
    depends = ["merge"]

    organization = "hashicorp"
    repository   = "terraform-provider-kubernetes"
    workflow     = "build"
  }
}

event "prepare" {
  depends = ["build"]

  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
    depends      = ["build"]
  }

  notification {
    on = "fail"
  }
}

event "trigger-staging" {
}

event "promote-staging" {
  action "promote-staging" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging"
    depends      = null
    config       = "release-metadata.hcl"
  }

  depends = ["trigger-staging"]

  notification {
    on = "always"
  }
}

event "trigger-production" {
}

event "promote-production" {
  action "promote-production" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-production"
    depends      = null
    config       = ""
  }

  depends = ["trigger-production"]

  notification {
    on = "always"
  }
}
