# pipeline-runner for TektonCD

This is a very early, pre-Alpha release of this code.

It can take a CI definition, similar to the GitLab CI definition, and execute the steps / tasks as a [TektonCD](https://github.com/tektoncd/pipeline) definition.

It has two different bits:

 * A "pipeline definition" to PipelineRun converter.
 * An HTTP Server that handles Hook requests from GitHub (and go-scm supported
   hosting services)  by requesting pipeline files from the incoming repository, and processing them.

## Building

```shell
$ go build ./cmd/testing
$ ./testing
Error: required flag(s) "branch", "pipeline-file", "repository-url" not set
Usage:
  testing [flags]

Flags:
      --branch string             checkout and execute against this branch (default "master")
  -h, --help                      help for testing
      --pipeline-file string      YAML with pipeline description
      --pipelinerun-name string   inserted into the generated PipelineRun resource (default "test-pipelinerun")
      --repository-url string     e.g. https://github.com/my-org/my-repo.git

2020/03/29 18:48:06 required flag(s) "branch", "pipeline-file", "repository-url" not set
```

## Experimenting with this.

The generated PipelineRun has an embedded Pipeline, with Tasks that execute the scripts defined in the example pipeline definition.

It requires a [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) to carry state between tasks.

There is an example [volume.yaml](./examples/volume.yaml).

By creating a volume, and executing the example pipeline, it should execute and the PipelineRun will complete.

## HTTP Hook Handler.

The `deploy` directory includes a Kubernetes Service `tekton-ci-http` on port `8080` which needs to be exposed to GitHub hooks, it supports two endpoints `/pipeline` and `/pipelinerun`, the first of these accepts pipeline syntax, the second supports a syntax closer to PipelineRuns.

This needs a recent version of Tekton Pipelines installed, it will automatically create a 1Gi Volume claim per run! and there's nothing currently which cleans this up!

```shell
kubectl apply -f deploy/
```

After this, create a simple `.tekton_ci.yaml` in the root of your repository, following the example syntax, and it should be executed when a pull-request is created.

When the handler receives the `pull_request` hook notification, it will try and
get a configuration file from the repository and process it.

## /pipeline endpoint

This supports a GitLab-CI-like syntax, capable of executing scripts, it uses a [Persistent Volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) to transport the source and output between tasks.

### Currently understood syntax

```yaml
# this image is used when executing the script.
image: golang:latest

# before_script is performed before any of the tasks.
before_script:
  - wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0

# after_script is performed before any of the tasks.
after_script:
  - echo "after script"

# This provides ordering of the tasks defined in the pipeline,
# all steps in each stage will be scheduled ahead of the tasks in
# subsequent stages.
stages:
  - test
  - build

# This is a "Task" called "format", it's executed in the "test" stage above.
# It will be executed in the top-level directory of the checked out code.
format:
  stage: test
  script:
    - go mod download
    - go fmt ./...
    - go vet ./...
    - ./bin/golangci-lint run
    - go test -race ./...

# this is another Task, it will be executed in the "build" stage, which because
# of the definition of the stages above, will be executed after the "test" stage
# jobs.
compile:
  stage: build
  script:
    - go build -race -ldflags "-extldflags '-static'" -o testing ./cmd/github-tool
  # If artifacts are specified as part of a Task, an extra container is
  # scheduled to execute after the task, which is executed in the same volume.
  # this will receive the list of artifacts and can upload the artifact
  # somewhere - admittedly this is a bit vague.
  artifacts:
    paths:
      - github-tool
```

## /pipelinerun endpoint

This supports standard [PipelineRuns](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md) with a wrapper around them to automate extraction of the arguments from the incoming hook body.

The example below, if placed in `.tekton/pull_request.yaml` will trigger a simple script that echoes the SHA of the commit when a pull-request is opened.

The expressions in the `filter` and `paramBindings` use [CEL syntax](https://github.com/google/cel-go), and the `hook` comes from the incoming hook, in the example below, this is a [PullRequestHook](https://github.com/jenkins-x/go-scm/blob/master/scm/webhook.go#L251).

The PipelineRunSpec is a standard PipelineRun [spec](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md#syntax).

The PipelineRun is created with an automatically generated name, and the `paramBindings` will be _added_ to the pipeline run parameters, this makes it easy to use standard pipelines, but with a mixture of hard-coded and dynamic parameters.

```yaml
filter: hook.Action == 'opened'
paramBindings:
  - name: COMMIT_SHA
    expression: hook.PullRequest.Sha
pipelineRunSpec:
  pipelineSpec:
    params:
      - name: COMMIT_SHA
        description: "The commit from the pull_request"
        type: string
    tasks:
      - name: echo-commit
        taskSpec:
          params:
          - name: COMMIT
            type: string
          steps:
            - name: echo
              image: ubuntu
              script: |
                #!/usr/bin/env bash
                echo "$(params.COMMIT)"
        params:
          - name: COMMIT
            value: $(params.COMMIT_SHA)
```

## HTTP server

See the deployment file in [deployment.yaml](./deploy/deployment.yaml).

## Things to do

 * Namespaces for PipelineRuns.
 * Watch for ending runs and delete the volue mount
 * Support private Git repositories.
 * Better naming for the handlers (pipeline and pipelinerun are not
   descriptive).
 * Support more syntax items (extra containers, saving and restoring the cache)
 * Fix parallel running of tasks in the same stage
 * Configuration for archiving - currently spawns an image with a URL, how to
   do configuration for this?
 * Provide support for calling other Tekton tasks from the script DSL.
 * Support for service-broker bindings.
 * Move away from the bespoke YAML definition to a more structured approach
   (easier to parse) - this might be required for better integration with Tekton
   tasks.
 * Support more events (Push) and actions other than `opened` for the script DSL format.
 * Filtering of the events (only pushes to "master" for example).
 * ~~Automate volume claims for the script-based DSL.~~
 * ~~Add support for the [commit-status-tracker](https://github.com/tektoncd/experimental/tree/master/commit-status-tracker)~~
 * ~~HTTP hook endpoint to trigger pipelineruns automatically~~
