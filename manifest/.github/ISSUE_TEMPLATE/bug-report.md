---
name: "\U0001F41B Bug Report"
about: "If something isn't working as expected \U0001F914."
title: ''
labels: bug

---

### Terraform, Provider, Kubernetes versions
<!--- Run `terraform -v` to show the version. If you are not running the latest version of Terraform, please upgrade because your issue may have already been fixed. 
Run `kubectl version` to show the server and client versions. Note that this provider only supports Server versions of 1.17 and above.
--->
```
Terraform version:
Provider version:
Kubernetes version:
```

### Affected Resource(s)
<!-- Please list the resources as a list, for example:
- opc_instance
- opc_storage_volume
If this issue appears to affect multiple resources, it may be an issue with Terraform's core, so please mention this. -->

### Terraform Configuration Files
```hcl
# Copy-paste your Terraform configurations here - for large Terraform configs,
# please use a service like Dropbox and share a link to the ZIP file. For
# security, you can also encrypt the files using our GPG public key.
```

### Debug Output
<!--Please provider a link to a GitHub Gist containing the complete debug output: https://www.terraform.io/docs/internals/debugging.html. Please do NOT paste the debug output in the issue; just paste a link to the Gist.-->

### Panic Output
<!--If Terraform produced a panic, please provide a link to a GitHub Gist containing the output of the `crash.log`.-->

### Steps to Reproduce
<!-- Please list the steps required to reproduce the issue, for example:
1. `terraform apply` -->

### Expected Behavior
What should have happened?

### Actual Behavior
What actually happened?

### Important Factoids
<!-- Are there anything atypical about your accounts that we should know? For example: Running in EC2 Classic? Custom version of OpenStack? Tight ACLs?-->

### References
<!--Are there any other GitHub issues (open or closed) or Pull Requests that should be linked here? For example:-->
- GH-1234

### Community Note
<!--- Please keep this note for the community --->
* Please vote on this issue by adding a 👍 [reaction](https://blog.github.com/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/) to the original issue to help the community and maintainers prioritize this request
* If you are interested in working on this issue or have submitted a pull request, please leave a comment

