<!-- heimdall_github_prtemplate:grc-pci_dss-2024-01-05 -->

## Rollback Plan

If a change needs to be reverted, we will publish an updated version of the library.

## Changes to Security Controls

Are there any changes to security controls (access controls, encryption, logging) in this pull request? If so, explain.

<!--- Please leave a helpful description of the changes to security controls here. --->


### Description

<!--- Please leave a helpful description of the pull request here. --->

### Acceptance tests
- [ ] Have you added an acceptance test for the functionality being added?
- [ ] Have you run the acceptance tests on this branch?

Output from acceptance testing:

<!--
Replace TestAccXXX with a pattern that matches the tests affected by this PR.

For more information on the `-run` flag, see the `go test` documentation at https://tip.golang.org/cmd/go/#hdr-Testing_flags.
-->
```
$ make testacc TESTARGS='-run=TestAccXXX'

...
```

### Release Note
Release note for [CHANGELOG](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/CHANGELOG.md):
<!--
If change is not user facing, just write "NONE" in the release-note block below.
-->

```release-note
...
```

### References

<!---
Are there any other GitHub issues (open or closed) or pull requests that should be linked here? Vendor blog posts or documentation?
--->
### Community Note
<!--- Please keep this note for the community --->
* Please vote on this issue by adding a 👍 [reaction](https://blog.github.com/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/) to the original issue to help the community and maintainers prioritize this request
* If you are interested in working on this issue or have submitted a pull request, please leave a comment
