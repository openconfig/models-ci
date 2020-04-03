# models-ci

Continuous integration for OpenConfig models.

## Purpose

There are several widely-used YANG tools that form an important part of what
makes OpenConfig possible.

These tools either do validation directly based on their interpretation of the
YANG RFCs, or, more importantly, do processing on them (e.g. code generation)
that require them to be in some format.

This CI tool helps model committers avoid breaking these important YANG tools
and comply with the RFCs.

## How to Add a Validator

1.  Determine which category the validator is: per-model or not.
2.  Add to `commonci.go`'s `Validators` map by giving it an ID and a short
    description.
3.  If the validator is per-model, add to `cmd_gen.go`'s `createValidatorFmtStr`
    a format string for creating the validation command for each model. Don't
    forget to add a test. It's possible that it requires a special format
    string, requiring special handling from the other validators.
4.  If special parsing is necessary, modify `parseModelResultsHTML`'s parsing
    logic to apply special formatting for your tool's output.
5.  Add a `<validatorId>/test.sh` file (see others for examples) that invokes
    the generated `script.sh`, creates the special files, and calls
    `post_results` for your validator. Depending on the nature of your
    validator, there may be extra set-up that's required, or it could be very
    simple.
6.  Add a step in `cloudbuild.yaml` to invoke `test.sh`. Depending on the
    validator, other preparatory steps may be required.
7.  (optional) If more than one version is to be ran (e.g. allowing extra
    versions to be supplied), look at how this is done for pyang (in `cmd_gen`,
    its `test.sh`, and its `cloudbuild.yaml` step) and add capability for it
    accordingly.

## Categories of CI Validators

-   Per-Model Validators

Validators that are run once per model (per entry in each model's `.spec.yml`).
e.g. pyang

-   Non Per-Model validators

Validators that are run directly in a simple command on the entire repository.
e.g. regexp tests

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

### 2 Validator Script Execution

Per-model validators each have a minimal `test.sh` that can be invoked directly
during CI that takes care of all the preparatory, invocation, and
post-processing steps of running the script. These bash files are kept to a
minimal as bash is more difficult to write than Go.

Non per-model validators may not have `test.sh`, and so may require a direct
call in the CI pipline in order to execute.

No matter which way, the results are put into a
`/workspace/results/<validatorId><non-latest-version>` directory to be processed
into a human-readable format. Here `/workspace` is the root directory for all
GCB builds. You can find the `validatorId` for each validator in the `commonci`
package. The latest version is either the latest tagged version, or absent, the
head.

#### Special Files Within Each Validator's Results Directory and Their Meanings

`script.sh`: per-model validator execution script name.

`out`: Stores stdout of validator execution. **required** to be present to
indicate that the script ran.

`fail`: Stores stderr of validator execution. A non-existent or an empty fail
file is a pass.

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

## How Each Validator is Installed

Validator         | Installation
----------------- | -----------------------------------------------------------
regexp            | Files are moved into GOPATH from its folder during CI build
pyang & pyangbind | pip
oc-pyang          | git clone
goyang/ygot       | go get
yanglint          | Debian package periodically uploaded to cloud storage

## Setting Up GCB

models-ci is written to be run on Google Cloud Build (GCB) on a GitHub
OpenConfig models repository. While it is possible for it to be adapted for
other CI infrastructures, it was not written with that in mind.

In particular, models-ci assumes a corresponding
[`cloudbuild.yaml`](https://cloud.google.com/cloud-build/docs/automating-builds/run-builds-on-github#preparing_a_github_repository_with_source_files)
file stored in each OpenConfig models respository in which CI is to be executed.
The details of `cloudbuild.yaml` is heavily dependent on the assumptions made in
this respository.

Each build step of `cloudbuild.yaml` is run by a docker container inside the
same VM. The steps need to made to coordinate with one another manually. GCB
allows steps to have arbitrary dependencies, and parallel execution of steps
whose ancestor steps have already been executed.

As [pre-built images](https://cloud.google.com/cloud-build/docs/cloud-builders)
provided by GCB are currently used to run CI, `cloudbuild.yaml` requires some
preparation steps. The high-level build steps needed in the `cloudbuild.yaml`
are,

1.  Clone models-ci repo into GOPATH and get dependencies using `go get ./...`.
2.  Call `cmd_gen` to generate the validator scripts for each validator tool.
3.  Prepare each validator tool if necessary.
4.  Run each validator tool either directly, or through the `script.sh`
    generated from `cmd_gen`, redirecting the result into specified files.
5.  If `script.sh` is not used for a validator tool, then `post_results` needs
    to be called afterwards as well.

To run this CI tool on GCB for a GitHub project, the
[GCB App](https://github.com/marketplace/google-cloud-build) needs to be enabled
for the target OpenConfig models repo.

## Future Improvements

A custom build container image would,

-   simplify the `cloudbuild.yaml` script.
-   speed up the build.

Check runs is a better UI than posting gists as statuses. Unfortunately check
runs are currently
[unsupported by GCB](https://groups.google.com/g/google-cloud-dev/c/fON-kDlykLc).
