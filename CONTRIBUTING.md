# Contribution Guidelines

The Arm Topo CLI project is open for external contributors, and welcomes contributions.

Topo CLI is licensed under the [Apache-2.0](https://spdx.org/licenses/Apache-2.0.html) license and all accepted contributions must have the same license.

## Contributing code to Topo CLI

- Before this project accepts your contribution, you need to certify its origin and give us your permission. To manage this process, we use [Developer Certificate of Origin (DCO) V1.1](https://developercertificate.org/).
  To indicate that contributors agree to the terms of the DCO, it's necessary to "sign off" the contribution by adding a line with your name and email address to every git commit message:

  ```log
  Signed-off-by: FIRST_NAME SECOND_NAME <your@email.address>
  ```

  This can be done automatically by adding the `-s` option to your `git commit` command. You must use your real name, no pseudonyms or anonymous contributions are accepted.

  ## Code Reviews

  Contributions must go through code review on GitHub.

  See [here](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork)
  for details of how to create a pull request from your working fork.

  Only reviewed contributions can be merged to the main branch.

  ## Continuous integration

  Contributions to Topo CLI must go through testing and formatting before being merged to main. All unit and integration tests must pass, as well as formatting checks, before a contribution is merged. These tests are run through GitHub Actions, which require the permission of a maintainer to run.