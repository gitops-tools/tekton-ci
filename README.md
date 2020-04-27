# pipeline-runner for TektonCD 

[![Go Report Card](https://goreportcard.com/badge/github.com/bigkevmcd/tekton-ci)](https://goreportcard.com/report/github.com/bigkevmcd/tekton-ci)
![Go](https://github.com/bigkevmcd/tekton-ci/workflows/Go/badge.svg)

This is pre-Beta release of this code.

It can take a CI definition, similar to the GitLab CI definition, and execute the steps / tasks as a [TektonCD](https://github.com/tektoncd/pipeline) PipelineRun.

It has two different bits:

 * A "pipeline definition" to PipelineRun converter.
 * An HTTP Server that handles Hook requests from GitHub (and go-scm supported
   hosting services) by requesting pipeline files from the incoming repository, and processing them.

## Table of contents
1. [Building](#building)
2. [Receiving GitHub hooks](#receiving-github-hooks)
3. [DSL Hook Handler](#dsl-hook-handler)
4. [Testing](#testing)
5. [Things to do](#things-to-do)

## Building

A `Dockerfile` is provided for building a container image.

## Receiving GitHub hooks

The main use of this component is driving CI pipelines when a "push" to a
specific occurs.

### Prerequisites

This requires Tekton Pipelines to be installed, at least v0.11.0

```shell
$ kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/v0.11.0/release.yaml
```

## Private Repo access

You'll need to create an access token, with `repo` scope.

```shell
$ kubectl create secret generic tekton-ci-client --from-literal=token=<access token>
```

### Deploying the container

The hook receiver needs to be deployed to Kubernetes.

In the [`deploy`](./deploy) directory, there are two files:
 * `role.yaml` with a `ServiceAccount` **tekton-ci**, along with a `Role` and
   `RoleBinding` that allows the `ServiceAccount` to create volumes and
   pipeline runs.
   `ServiceAccount`.
 * `deployment.yaml` which contains a Kubernetes `Deployment` resource, and a
   `Service` to expose the deployment.

You'll need to expose these outside of your cluster, so that GitHub hooks can
hit the endpoint.

For simple testing, you can port-forward and use ngrok.

```shell
$ kubectl port-forward tekton-ci-http 8080
# In a separate terminal window
$ ngrok http 8080
```

Then create a [JSON Webhook](https://developer.github.com/webhooks/creating/) to
point at your endpoint, you need to choose a path for your hook endpoint:

 * /pipeline - [this](#dsl-hook-handler) interprets a [GitLab CI](https://docs.gitlab.com/ee/ci/) like syntax.
 * /pipelinerun - [this](#spec-hook-handler) provides for a way to execute [Tekton Pipeline](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md) definitions.

You will also need to create a Secret with a shared secret from your GitHub hook, the name must be `tekton-ci-hook-secrets`.

The key in the secret is your org/repo with the `/` replaced by an `_` (underscore).

NOTE: `/` are not allowed in Secret keys.

```shell
$ kubectl create secret generic tekton-ci-hook-secrets --from-literal=bigkevmcd_tekton-ci=test-secret
```

In this case, the secret in GitHub is `test-secret`, and incoming hooks will be validated against this.

See https://kubernetes.io/docs/concepts/configuration/secret/#creating-a-secret-manually here for more.

Any number of repos can be handled with the same secret, as long as there are
keys for the repository.

## DSL Hook Handler

Once you have a hook pointing at the correct path (/pipeline) then  create a simple `.tekton_ci.yaml` in the root of your repository, following the example syntax, and it should be executed when a push hook is sent from GitHub.

When the handler receives the `push` hook notification, it will try and get a configuration file from the repository and process it.

To do this, it first of all creates a `PersistentVolumeClaim` (this is currently 1Gi) and then converts the pipeline definition into a PipelineRun with an embedded Pipeline and embedded Tasks, including a task that checks out the source code then begins to execute the scripts.

### Currently understood syntax

```yaml
# this image is used when executing the script.
image: golang:latest

# before_script is performed before any of the tasks.
before_script:
  - wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0

tekton:
  # This allows configuration of the serviceAccountName for the PipelineRun.
  serviceAccountName: test-service-account

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
test:
  stage: test
  script:
    - go mod download
    - go fmt ./...
    - go vet ./...
    - ./bin/golangci-lint run
    - go test -race ./...

# This will execute the non-cluster Task "my-test-task".
tekton-task
  stage: test
  tekton:
    taskRef: my-test-task
    # Params here are processed as CEL expressions and passed to the Task.
    params:
      - name: IMAGE_URL
        expr: "'quay.io/testing/testing'"

# this is another Task, it will be executed in the "build" stage, which because
# of the definition of the stages above, will be executed after the "test" stage
# jobs.
compile:
  stage: build
  script:
    - go build -race -ldflags "-extldflags '-static'" -o testing ./cmd/github-tool
  tekton:
    # These are used to create a job matrix, this task will be executed for each
    # option, and the options are split into env-vars, and placed into a task's
    # environment.
    #
    # All the tasks in the matrix are executed in parallel.
    #
    # This can be used to parallelise tests, for example, you can execute your
    # test-runner, and detect the value of the "TESTS_TO_RUN" env-var, and
    # execute accordingly.
    jobs:
      - TESTS_TO_RUN=integration
      - TESTS_TO_RUN=unit
      - TESTS_TO_RUN=e2e
  # If artifacts are specified as part of a Task, an extra container is
  # scheduled to execute after the task, which is executed in the same volume.
  # this will receive the list of artifacts and can upload the artifact
  # somewhere - the image is configurable.
  artifacts:
    paths:
      - github-tool
```

## Spec Hook Handler

The other HTTP handler is at `/pipelinerun`, this supports standard [PipelineRuns](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md) with a wrapper around them to automate extraction of the arguments from the incoming hook body.

The example below, if placed in `.tekton/pull_request.yaml` will trigger a simple script that echoes the SHA of the commit when a pull-request is opened.

The expressions in the `filter` and `paramBindings` use [CEL syntax](https://github.com/google/cel-go), and the `hook` comes from the incoming hook, in the example below, this is a [PushHook](https://github.com/jenkins-x/go-scm/blob/master/scm/webhook.go#L77).

The PipelineRunSpec is a standard PipelineRun [spec](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md#syntax).

The PipelineRun is created with an automatically generated name, and the `paramBindings` will be _added_ to the pipeline run parameters, this makes it easy to use standard pipelines, but with a mixture of hard-coded and dynamic parameters.

```yaml
filter: hook.Ref == 'refs/heads/master'
paramBindings:
  - name: COMMIT_SHA
    expression: hook.Before
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

## Testing

```shell
$ go test -v ./...
```

## Things to do

In no particular order.

 * Metrics.
 * Better naming for the handlers (pipeline and pipelinerun are not
   descriptive).
 * Support more syntax items (extra containers, saving and restoring the cache)
 * Configuration for archiving - currently spawns an image with a URL, how to
   do configuration for this?
 * Support for service-broker bindings.
 * Integration of the GitHub status notifications.
 * Move away from the bespoke YAML definition to a more structured approach
   (easier to parse) - this might be required for better integration with Tekton
   tasks.
 * Configurability of volume creation.
 * Watch for ending runs and delete the volume mount - this is tricky without deleting the pipelinerun that is using it too.
 * ~~Support private Git repositories.~~
 * ~~Provide the hook ID as an "execution ID" to improve traceability.~~
 * ~~Support for secrets to validate incoming Webhooks.~~
 * ~~Support for parallelism via build matrices.~~
 * ~~Allow passing params from the Tekton task mechanism through to the Task.~~
 * ~~Provide support for calling other Tekton tasks from the script DSL.~~
 * ~~Filtering of the events (only pushes to "master" for example).~~
 * ~~Support more events (Push) and actions other than `opened` for the script DSL format.~~
 * ~~Fix parallel running of tasks in the same stage~~
 * ~~Automate volume claims for the script-based DSL.~~
 * ~~Add support for the [commit-status-tracker](https://github.com/tektoncd/experimental/tree/master/commit-status-tracker)~~
 * ~~HTTP hook endpoint to trigger pipelineruns automatically~~
