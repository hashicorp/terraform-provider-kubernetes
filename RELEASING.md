# Releasing

The purpose of this document is to outline the release process for the Kubernetes Provider for Terraform.

The Semantic Versioning agreement is being followed by this project. Further details can be found [here](https://semver.org/).

## How To Release

To create a new release, adhere to the following steps:

- Decide on the version number that you intend to release. Throughout the following steps, it will be denoted as `<SEMVER>`.

- Switch to the `main` branch and fetch the latest changes:

  ```console
  $ git switch main
  $ git pull
  ```

- Create a new branch from the `main`. The branch name is required to adhere to the following template: `release/v<SEMVER>`.

  ```console
  $ git checkout -b release/v<SEMVER>
  ```

- Generate change log entries:

  ```console
  $ make changelog
  ```

- Update the [`CHANGELOG`](./CHANGELOG.md) file with the output produced in the previous step preceded by the release version and the planned release date expressed as `Mon DD, YYYY` format. The version number in this file must correspond with the `<SEMVER>` of the release branch name.

- Create a pull request against the `main` branch and follow the regular code review and merge procedures.

- After merging the release branch into the `main` one, a git tag with the new release version number needs to be attached to the release commit to start a release process. The version number in the tag must correspond with the `v<SEMVER>` of the merged release branch name. Below is an example of the commands that need to be run in this step:

  ```console
  $ git switch main
  $ git pull
  $ git log --pretty=oneline -n 1

  ccd98a787308e6887c97291652430bd083106ccb (HEAD -> main, ...) v<SEMVER> (#XXXX)

  $ git tag v<SEMVER> ccd98a787308e6887c97291652430bd083106ccb
  $ git log --pretty=oneline -n 1

  ccd98a787308e6887c97291652430bd083106ccb (..., tag: v<SEMVER>) v<SEMVER> (#XXXX)

  $ git push origin v<SEMVER>
  ```

- Confirm this succeeded by viewing the repository [tags](https://github.com/hashicorp/terraform-provider-kubernetes/tags).

- Monitor the [release](https://github.com/hashicorp/terraform-provider-kubernetes/actions/workflows/release.yaml) action on GitHub. Once it is completed, a new release should be available on the [registry](https://registry.terraform.io/providers/hashicorp/kubernetes/latest) portal within 15-30 minutes. If this does not happen or the action fails, please reach out to the release engineering team to troubleshoot the issue.
