# Addressing CI Failures

The recommend way of resolving failures is 1) root causing via checking the
logs, 2) determining the appropriate action, and 3) making changes to the CI
pipeline if needed.

## Root Causing via Checking Logs

Logs are exposed via GCB in a link that’s accessible via the “Details” link of
the `public-pr` GitHub CI status. If the results are for some reason not
available, then you can check the build results by opening the Cloud project to
find the corresponding build.

When checking the logs, it is helpful to keep in mind the
[CI steps outlined here](https://github.com/openconfig/models-ci?tab=readme-ov-file#ci-steps)
to know where the error occurred in the CI pipeline.

## Determining the Appropriate Action

While incidents can be due to bugs in the CI pipeline’s implementation, GitHub
or GCB can also be the culprit. For example, if too many GitHub calls are made
(due to CI running too many concurrently-running PRs), then some statuses might
remain in the “pending” state, which just necessitates a re-run. GCB might have
a backward-incompatible change
[that needs updating](https://github.com/openconfig/public/pull/1047/files). In
the past, GitHub has changed its private/public key pair, necessitating
[a change](https://github.com/openconfig/public/pull/838) to the `known_hosts`
file required for GCB to run.

## Making a Change to the CI Pipeline

If you determine that a change is required to the
[CI pipeline](https://github.com/openconfig/models-ci), then understand
[which step](https://github.com/openconfig/models-ci?tab=readme-ov-file#ci-steps)
needs to change and open a PR with the fix. You can test the fix by opening a
test PR in openconfig/public and
[changing the branch](https://github.com/openconfig/public/blob/c00868ed96e8e48993e26d8fba20f093722c0e39/cloudbuild.yaml#L55)
used when pulling the CI pipeline code. Once it is approved, cut a new release
of the CI pipeline based on
[these guidelines](https://github.com/openconfig/models-ci?tab=readme-ov-file#usage-notes-and-versioning).
Lastly, submit a PR in the openconfig/public repo if needed, for example to bump
the CI pipeline version used.

Example 1: [Making a fix](https://github.com/openconfig/models-ci/pull/98)

Example 2:
[Adding a feature to the CI pipeline](https://github.com/openconfig/models-ci/pull/92)

Example 3:
[Adding a validator to the CI pipeline](https://github.com/openconfig/models-ci/pull/93)

Example 4:
[Building the CI image at a regular frequency to avoid stale dependencies](https://github.com/openconfig/models-ci/pull/91)
