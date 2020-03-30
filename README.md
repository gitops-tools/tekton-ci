# pipeline-runner for TektonCD

This is a very early, pre-Alpha release of this code.

It can take a CI definition, similar to the GitLab CI definition, and execute the steps / tasks as a [TektonCD](https://github.com/tektoncd/pipeline) definition.

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

## Currently understood syntax

```yaml
# this image is used when executing the script.
image: golang:latest

# before_script is performed before any of the jobs.
before_script:
  - wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0

# This provides ordering of the jobs defined in the pipeline,
# all tasks in each stage will be scheduled ahead of the tasks in
# subsequent stages.
stages:
  - test
  - build

# This is a "Job" called "format", it's executed in the "test" stage above.
# It will be executed in the top-level directory of the checked out code.
format:
  stage: test
  script:
    - go mod download
    - go fmt ./...
    - go vet ./...
    - ./bin/golangci-lint run
    - go test -race ./...

# this is another Job, it will be executed in the "build" stage, which because
# of the definition of the stages above, will be executed after the "test" stage
# jobs.
compile:
  stage: build
  script:
    - go build -race -ldflags "-extldflags '-static'" -o testing ./cmd/github-tool
```

## Things to do

 * Add support for the [commit-status-tracker](https://github.com/tektoncd/experimental/tree/master/commit-status-tracker)
 * Support more syntax items (extra containers, do something with artifacts).
 * Provide support for calling other Tekton tasks.
 * Support for service-broker bindings
 * HTTP hook endpoint to trigger pipelineruns automatically
