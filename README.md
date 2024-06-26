# models-ci

Continuous integration for OpenConfig models.

[Guidance on how to address a CI failure](docs/addressing_ci_failures.md)

## Usage Notes and Versioning

In order to avoid backwards-incompatible new changes from affecting and breaking
CI, a user repository's `cloudbuild.yaml` should always use `models-ci` with a
specified **major** version only, e.g.

`go get github.com/openconfig/models-ci@v1`

New minor and patch versions are guaranteed to be backwards compatible per
[semantic versioning rules](https://semver.org/)

A backwards compatible change is defined to mean a change not requiring changes
to any possible `cloudbuild.yaml` usage provided by the existing `models-ci`.

-   Example of major revision changes:
    -   Existing `cmd_gen` flag altered in behaviour or deleted by new
        `models-ci` change.
    -   Changes to how an existing validator is set-up or ran within
        `cloudbuild.yaml`.
-   Minor and patch revisions encompass the remaining changes, e.g.
    -   Adding a new `cmd_gen` flag.
    -   Adding a new validator.
    -   Changing an existing validator's behaviour, or any other part of
        `models-ci`, that doesn't break the current `cloudbuild.yaml` interface.

When changing `models-ci`, it's ok to be liberal when deciding whether to bump
up major revisions. It simply requires an update in `cloubuild.yaml` in user
repos to make use of the new features. The main point is to avoid breaking
existing `cloudbuild.yaml`.

## Updating the Build Image

Validators require the use of an image built using [Dockerfile](/Dockerfile).
Periodically this image needs to be rebuilt to avoid issues such as
[this](https://gist.github.com/OpenConfigBot/69d4b1707744221f0c53d49bcf5caaca),
where repositories likely started using `go modules` for dependencies.

## Purpose

There are several widely-used YANG tools that form an important part of what
makes OpenConfig possible in practice. These tools may validate models based on
their interpretation of the YANG RFCs, and may also do processing on them such
as code generation.

This CI suite helps ensure that changes to OpenConfig YANG models remain
compatible with key tools, and are compliant with relevant style guides and
standards. Validators are divided into those that must pass for models changes
to be committed, and those that are informational in nature (i.e., not required
to pass for merging).

## Categories of CI Validators (from a CI Perspective)

-   General (Per-Model) Validators

Validators that make sense to be run once per model (per entry in each model's
`.spec.yml`). e.g. pyang

-   Repo-level validators

Validators that are run directly in a simple command on the entire repository.
e.g. regexp tests

## How to Add a Validator

1.  Determine which category the validator is: per-model or repo-level.
2.  Add to `commonci.go`'s `Validators` map by giving it an ID and a short
    description.
3.  If the validator is per-model, add to `cmd_gen.go`'s `createValidatorFmtStr`
    a format string for creating the validation command for each model. Don't
    forget to add a test. It's possible that it requires a special format
    string, requiring special handling from the other validators.
4.  If special parsing of validator results is necessary, modify
    `parseModelResultsHTML`'s parsing logic to apply special formatting to your
    tool's output.
5.  Add a `<validatorId>/test.sh` file (see others for examples) that invokes
    the generated `script.sh`, creates the special files (see below), and calls
    `post_results` for your validator. Depending on the nature of your
    validator, there may be extra set-up steps required, or the above two calls
    may be sufficient within `test.sh`.
6.  Add a step in `cloudbuild.yaml` to invoke the above `test.sh`. Depending on
    the validator, other preparatory steps may be required within
    `cloudbuild.yaml`.
7.  (optional) If more than one version is to be run (whether by allowing
    arbitrary extra versions to be supplied, or always running two versions),
    look at how this is done for pyang (in `cmd_gen`, its `test.sh`, and its
    `cloudbuild.yaml` step) and add capability for it accordingly.

## CI Steps

CI has 3 steps:

1.  Generate validator scripts (`cmd_gen` Go script)
2.  Execute each validator script
3.  Parse and post results (`post_results` Go script)

### 1 `cmd_gen`

`cmd_gen` only needs to be run once. It creates a `script.sh` for each validator
which contains the validator commands ready to be invoked. Each validator's
`script.sh` is unique. For example, yanglint's can simply be invoked, whereas
pyang requires the path to the pyang executable and some environment variables
to be passed in as an arugment.

`cmd_gen` also creates and stores information inside the
`/workspace/user-config` directory, which contain user flags passed to `cmd_gen`
that controls the remaining CI steps, so that the steps after `cmd_gen` in the
cloudbuild.yaml are configurable through the `cmd_gen` step, and not require
detailed understanding from the user.

### 2 Validator Script Execution

Per-model validators each have a minimal `test.sh` that can be invoked directly
during CI that takes care of all the preparatory, invocation, and
post-processing steps of running the script.

Repo-level validators may not have `test.sh`, and so may require a direct call
in the CI pipline in order to execute.

No matter which way, the results are put into a
`/workspace/results/<validatorId><non-latest-version>` directory to be processed
into a human-readable format. Here `/workspace` is the root directory for all
GCB builds. You can find the `validatorId` for each validator in the `commonci`
package. The latest version is either the latest tagged version, or absent, the
head.

#### Required GCB Variables for Validator Scripts

The following variables must be supplied to each validator script
(i.e.`test.sh`) either as an environment variable in the validator script's
`cloudbuild.yaml` step, or as a
[GCB substitution variable](https://cloud.google.com/cloud-build/docs/configuring-builds/substitute-variable-values).

Variable Name  | Value
-------------- | --------------------------------------------------------------
$\_MODEL\_ROOT | Root GCB directory of all YANG models (e.g. `/workspace/yang`)
$\_REPO\_SLUG  | e.g. `openconfig/public`
$\_PR\_NUMBER  | GitHub PR number
$COMMIT\_SHA   | Full commit SHA for PR
$BRANCH\_NAME  | Name of branch for PR

#### Special Files Within Each Validator's Results Directory and Their Meanings

`script.sh`: per-model validator execution script name.

`out`: Stores stdout of validator execution. **required** to be present to
indicate that the script ran.

`fail`: Stores stderr of validator execution. With no stderr, it should be empty
if execution failed, and non-existent if execution passed. Thus, a non-existent
fail file is interpreted as a successful validator execution, which along with
each model's status (see below) for a per-model validator, or by itself for a
repo-level validator, determine whether the entire validation was successful.

`latest-version.txt`: Stores the name+version of the @latest validator to
display to the user.

`modelDir==model==status`: For per-model validators, each model has a file of
this format created by the validator execution script. `post_results`
understands this format, and scans all of these in order to output the results
in a hierarchical format to the user.

### 3 `post_results`

This script is aware of the results format for each validator. It parses each
result uniquely for each validator, and posts the information as a gist on the
GitHub PR.

The logs for this step resides in the same step as the "Validator Script
Execution" step.

## How Each Validator is Installed

Validator         | Installation
----------------- | -------------------------------------------------------
confd             | Binary unzipped during build
regexp            | Files moved into GOPATH from its folder during CI build
pyang & pyangbind | pip
oc-pyang          | git clone
goyang/ygot       | go get
yanglint          | Debian packages (libyang2 and libyang2-tools) periodically uploaded to cloud storage. These are renamed libyang.deb and yanglint.deb respectively in the GCS bucket.

## Setting Up GCB

models-ci is written to be run on Google Cloud Build (GCB) on a GitHub
OpenConfig models repository. While it is possible for it to be adapted for
other CI infrastructures, it was not written with that in mind.

In particular, models-ci assumes a corresponding
[`cloudbuild.yaml`](https://cloud.google.com/cloud-build/docs/automating-builds/run-builds-on-github#preparing_a_github_repository_with_source_files)
file stored in each OpenConfig models respository in which CI is to be executed
that specifies each CI step within GCB's environment. The details of
`cloudbuild.yaml` is heavily dependent on the assumptions made in this
respository and vice versa.

Each build step of `cloudbuild.yaml` is run by a docker container inside the
same VM. The steps are manually structured as a DAG for parallelism, taking
advantage of GCB's ability to allow steps to have arbitrary dependencies.

As [pre-built images](https://cloud.google.com/cloud-build/docs/cloud-builders)
provided by GCB are currently used to run CI, `cloudbuild.yaml` requires some
preparation steps. The high-level build steps needed in the `cloudbuild.yaml`
are,

1.  Clone models-ci repo into GOPATH and get dependencies using `go get ./...`.
2.  Call `cmd_gen` to generate the validator scripts for each validator tool. If
    a validator script should not gate the changes, but should only serve as a
    reference for committers, then they could be explicitly specified to appear
    in the compatibility report instead using -compat-report flag. Any
    validatorId@version can be skipped (from both the PR status as well as the
    compatibility report) using the `-skipped-validators` flag.
3.  Prepare each validator tool if necessary.
4.  Run each validator tool either directly, or through the `script.sh`
    generated from `cmd_gen`, redirecting the result into specified files.
5.  If `script.sh` is not used for a validator tool, then `post_results` needs
    to be called afterwards as well.

To run this CI tool on GCB for a GitHub project, the
[GCB App](https://github.com/marketplace/google-cloud-build) needs to be enabled
for the target OpenConfig models repo.

## Posting Status Badges

This is done through a code path in `post_results` that generates an
`upload-badge.sh` file if the CI was triggered on a master branch push. The
badge is created using the
[badge-maker](https://www.npmjs.com/package/badge-maker) package used by
[shields.io](https://shields.io/), whose output svg file is then uploaded to
cloud storage and made public. The script also sets the no-cache option to avoid
GitHub from excessively caching the badge.

## Future Improvements

A custom build container image would,

-   simplify the `cloudbuild.yaml` script.
-   speed up the build.

Check runs is a better UI than posting gists as statuses. Unfortunately check
runs are currently
[unsupported by GCB](https://groups.google.com/g/google-cloud-dev/c/fON-kDlykLc).
