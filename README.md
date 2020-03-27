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

## Categories of CI Validators

1.  Per-Model Validators

Validators that are run once per model (per entry in each model's `.spec.yml`).
e.g. pyang

1.  Non Per-Model validators

Validators that are run directly in a simple command on the entire repository.
e.g. regexp tests

## CI Steps

CI has 3 steps:

1.  Validator script generation (`cmd_gen` Go script)
2.  Validator script execution
3.  Parse and post results (`post_results` Go script)

, the scripts are run by minimal per-validator `test.sh` files, and
`post_results` is then able to parse the results of every validator.

#### 1 `cmd_gen`

`cmd_gen` only needs to be run once. It creates a `script.sh` for each validator
which contains the validator commands ready to be invoked. Each validator's
`script.sh` is unique. For example, yanglint's can simply be invoked, whereas
pyang requires the path to the pyang executable and some environment variables
to be passed in as an arugment.

#### 2 Validator script execution

Per-model validators each have a minimal `test.sh` that can be invoked directly
during CI that takes care of all the task of running the script. These bash
files are kept to a minimal as bash is more difficult to work with than Go.

Non per-model validators may not have `test.sh`, and so may require a direct
call in order to execute.

No matter which way, the results are put into a `/workspace/results/<validator
tool>` directory to be processed into a human-readable format. Here `/workspace`
is the root directory for all GCB builds.

| Special File Within Results | Meaning                                       |
: Directory                   :                                               :
| --------------------------- | --------------------------------------------- |
| `script.sh`                 | per-model validator execution script name.    |
| `out`                       | Stores stdout of validator execution.         |
:                             : **required** to be present to indicate that   :
:                             : the script ran.                               :
| `fail`                      | Stores stderr of validator execution. A       |
:                             : non-existent or an empty fail file is a pass. :
| `latest-version.txt`        | Stores the name+version of the @latest        |
:                             : validator to display to the user.             :
| `modelDir==model==status`   | Each model has a file of this format created  |
:                             : by the per-model validator execution script.  :
:                             : `post_results` understands this format, and   :
:                             : scans all of these in order to output the     :
:                             : hierarchical results output to the user for   :
:                             : per-model validators.                         :

#### 3 `post_results`

This script is aware of the results format outputted from each validator. It
parses each result uniquely for each validator, and posts the information on the
GitHub PR.

## How Each Validator is Installed

Validator     | Installation
------------- | -----------------------------------------------------
regexp        | Files are moved into gopath during CI build
pyang-related | pip and git clone for oc-pyang
goyang/ygot   | go get
yanglint      | Debian package periodically uploaded to cloud storage

## Setting Up GCB

models-ci is written to be ran on Google Cloud Build (GCB) on a GitHub
OpenConfig repository. While it is possible for it to be adapted for other CI
infrastructures, it was not written with that in mind.

In particular, models-ci assumes a corresponding
[`cloudbuild.yaml`](https://cloud.google.com/cloud-build/docs/automating-builds/run-builds-on-github#preparing_a_github_repository_with_source_files)
file stored in each OpenConfig models respository in which CI is to be executed.
The details of `cloudbuild.yaml` is heavily dependent on the assumptions made in
this respository.

Each build step of `cloudbuild.yaml` is run of a docker container inside the
same VM. The steps need to made to coordinate with one another manually.

As [pre-built images](https://cloud.google.com/cloud-build/docs/cloud-builders)
provided by GCB are currently used to run CI, `cloudbuild.yaml` requires some
preparation steps. The high-level build steps needed in the `cloudbuild.yaml`
are,

1.  Clone models-ci repo into GOPATH and get dependencies using `go get ./...`.
2.  Call `cmd_gen` to generate the validator scripts for each validator tool.
3.  Prepare each validator tool if necessary.
4.  Run each validator tool either directly, or through the `script.sh`
    generated from `cmd_gen`, redirecting the result into specified directories.
5.  If `script.sh` is not used for a validator tool, then `post_results` needs
    to be called afterwards.

To run this CI tool on GCB for a GitHub project, the
[GCB App](https://github.com/marketplace/google-cloud-build) needs to be enabled
for the models repo.

## Future Improvements

A custom build container image would,

-   simplify the `cloudbuild.yaml` script.
-   speed up the build.
