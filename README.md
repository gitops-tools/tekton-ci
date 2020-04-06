# pipeline-runner for TektonCD

This is a very early, pre-Alpha release of this code.

It can take a CI definition, similar to the GitLab CI definition, and execute the steps / tasks as a [TektonCD](https://github.com/tektoncd/pipeline) definition.

It has two different bits:

 * A "pipeline definition" to PipelineRun converter.
 * An HTTP Server that handles Hook requests from GitHub by requesting pipeline
   files from the incoming repository, and processing them.

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

The generated PipelineRun has an embedded Pipeline, with Tasks that execute the
scripts defined in the example pipeline definition.

It requires a [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) to carry state between tasks.

There is an example [volume.yaml](./examples/volume.yaml).

By creating a volume, and executing the example pipeline, it should execute and
the PipelineRun will complete.

## HTTP Hook Handler.

This needs a recent version of Tekton Pipelines installed, and a simple volume claim (see the example above).

```shell
kubectl apply -f deploy/
```

This defines a Kubernetes Service `tekton-ci-http` on port `8080` which needs to be exposed to GitHub hooks.

After this, create a simple `.tekton_ci.yaml` in the root of your repository, following the example syntax, and it should be executed when a pull-request is created.

## Currently understood syntax

```yaml
# this image is used when executing the script.
image: golang:latest

# before_script is performed before any of the jobs.
before_script:
  - wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0

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
```

## HTTP server

See the deployment file in [deployment.yaml](./deploy/deployment.yaml).

## Things to do

 * ~~Add support for the [commit-status-tracker](https://github.com/tektoncd/experimental/tree/master/commit-status-tracker)~~
 * Support more syntax items (extra containers, do something with artifacts).
 * Provide support for calling other Tekton tasks.
 * Support for service-broker bindings
 * ~~HTTP hook endpoint to trigger pipelineruns automatically~~
 * Automatic Volume creation
 * Move away from the bespoke YAML definition to a more structured approach (easier to parse).
