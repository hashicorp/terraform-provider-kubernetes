# Issue Reporting and Lifecycle

## Issue Reporting Checklists

We welcome your feature requests and bug reports. Below you'll find short checklists with guidelines for well-formed
issues of each type.

### [Bug Reports](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/new/choose)

 - [ ] __Test against the latest release__: Make sure you test against the latest
   released version. It is possible we already fixed the bug you're experiencing.

 - [ ] __Search for possible duplicate reports__: It's helpful to keep bug
   reports consolidated to one thread, so do a quick search on existing bug
   reports to check if anybody else has reported the same thing. You can [scope
      searches by the label "bug"](https://github.com/terraform-providers/terraform-provider-kubernetes/issues?q=is%3Aopen+is%3Aissue+label%3Abug) to help narrow things down.

 - [ ] __Include steps to reproduce__: Provide steps to reproduce the issue,
   along with your `.tf` files, with secrets removed, so we can try to
   reproduce it. Without this, it makes it much harder to fix the issue.

 - [ ] __For panics, include `crash.log`__: If you experienced a panic, please
   create a [gist](https://gist.github.com) of the *entire* generated crash log
   for us to look at. Double check no sensitive items were in the log.

### [Feature Requests](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/new/choose)

 - [ ] __Search for possible duplicate requests__: It's helpful to keep requests
   consolidated to one thread, so do a quick search on existing requests to
   check if anybody else has reported the same thing. You can [scope searches by
      the label "enhancement"](https://github.com/terraform-providers/terraform-provider-kubernetes/issues?q=is%3Aopen+is%3Aissue+label%3Aenhancement) to help narrow things down.

 - [ ] __Include a use case description__: In addition to describing the
   behavior of the feature you'd like to see added, it's helpful to also lay
   out the reason why the feature would be important and how it would benefit
   Terraform users.


## Issue Lifecycle

1. The issue is reported on Github.

2. The issue is verified and categorized by a Terraform collaborator.
   Categorization is done via GitHub labels. We use
   one of `bug`, `enhancement`, `documentation`, or `question` using some automated workflows.

3. An initial triage process determines whether the issue is critical and must
    be addressed immediately, or can be left open for community discussion. In this step, we typically assign a size estimate to the work involved for that issue for our reference. We'll label the issue `acknowledged` when we've run through this step.

4. The issue queued in our backlog to be addressed in a pull request or commit. The issue number will be
   referenced in the commit message so that the code that fixes it is clearly
   linked.

5. The issue is closed. Sometimes, valid issues will be closed because they are
   tracked elsewhere or non-actionable. The issue is still indexed and
   available for future viewers, or can be re-opened if necessary.
