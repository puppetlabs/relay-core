# Contributing to Relay Core

## Guidelines

All interactions between Puppet employees, contributors, and community members
on Relay-related projects are subject to [Puppet Community Code of
Conduct](https://puppet.com/community/community-guidelines/).

Make sure there's not some existing code or a discussion that covers the change
you want to make by searching existing Github issues.

To make it easier to contribute while still staying in the good graces of our
(super wonderful!) Legal department, we require a [Developer Certificate of
Origin](https://developercertificate.org/) sign-off on contributions. See [this
explanation](https://helm.sh/blog/helm-dco/) from the Helm project to understand
the rationale behind the DCO. As a practical matter, this means adding the `-s |
--signoff` flag to your commits.

## Running tests

The integration tests in this repository require a running Kubernetes cluster
(not just a control plane). We recommend using [k3d](https://github.com/k3d-io/k3d) or [kind](https://kind.sigs.k8s.io/)
locally. You don't need any special configuration for it to work with our tests.

Assuming you have a Kubeconfig file at `/tmp/kubeconfig` with the current
context set to your kind cluster, you can run the tests:

```console
$ RELAY_TEST_E2E_KUBECONFIG=/tmp/kubeconfig go test ./...
```

**Important:** Do not run the tests against a cluster you care about! We haven't
audited the tests to make sure they won't delete something they're not supposed
to (and we may never do so!).

## Making changes

* Clone the repository into your own namespace.
* Make commits of logical and atomic units.
* Check for unnecessary whitespace with `git diff --check` before committing.
* Make sure your commit messages are in the proper format. We (try to!) follow
  [Tim Pope's
  guidelines](https://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
  for writing good commit messages: format for short lines, use the imperative
  mood ("Add X to Y"), describe before and after state in the commit message
  body. Remember to add the `-s` flag to commits to DCO-sign them!
* Make sure you have added the necessary tests for your changes.
* Submit a pull request per the usual GitHub PR process.

## Additional resources

* [Puppet community guidelines](https://puppet.com/community/community-guidelines)
* [Puppet community Slack](https://slack.puppet.com)
* [General GitHub documentation](https://help.github.com/)
* [GitHub pull request documentation](https://help.github.com/articles/creating-a-pull-request/)
