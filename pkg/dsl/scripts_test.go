package dsl

import (
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/cel"
	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
	"github.com/bigkevmcd/tekton-ci/test/hook"
	"github.com/google/go-cmp/cmp"
)

const (
	testPipelineRunPrefix  = "my-pipeline-run-"
	testArchiverImage      = "quay.io/testing/testing"
	testArchiveURL         = "https://example/com/testing"
	testRepoURL            = "https://github.com/myorg/testing.git"
	testServiceAccountName = "test-account"
)

func TestMakeGitCloneTask(t *testing.T) {
	env := []corev1.EnvVar{
		{Name: "CI_PROJECT_DIR", Value: "$(workspaces.source.path)"},
	}
	task := makeGitCloneTask(env, &Source{RepoURL: testRepoURL, Ref: "master"})

	want := pipelinev1.PipelineTask{
		Name: gitCloneTaskName,
		Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{
			{Name: "source", Workspace: workspaceName},
		},
		TaskSpec: &pipelinev1.TaskSpec{
			Workspaces: []pipelinev1.WorkspaceDeclaration{
				{
					Name: "source",
				},
			},
			Steps: []pipelinev1.Step{
				{
					Container: corev1.Container{
						Name:    "git-clone",
						Image:   "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
						Command: []string{"/ko-app/git-init", "-url", testRepoURL, "-revision", "master", "-path", workspaceSourcePath},
						Env: []corev1.EnvVar{
							{
								Name:  "CI_PROJECT_DIR",
								Value: "$(workspaces.source.path)",
							},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, task); diff != "" {
		t.Fatalf("PipelineTask doesn't match:\n%s", diff)
	}
}

func TestMakeScriptTask(t *testing.T) {
	image := "golang:latest"
	beforeScript := []string{
		"mkdir -p $GOPATH/src/$(dirname $REPO_NAME)",
		"ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME",
		"cd $GOPATH/src/$REPO_NAME",
	}
	env := []corev1.EnvVar{
		{Name: "CI_PROJECT_DIR", Value: "$(workspaces.source.path)"},
	}

	task := makeScriptTask("test-script-task", []string{gitCloneTaskName}, env, image, beforeScript)
	want := pipelinev1.PipelineTask{
		Name:     "test-script-task",
		RunAfter: []string{gitCloneTaskName},
		Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{
			{Name: "source", Workspace: workspaceName},
		},
		TaskSpec: &pipelinev1.TaskSpec{
			Workspaces: []pipelinev1.WorkspaceDeclaration{
				{
					Name: "source",
				},
			},
			Steps: []pipelinev1.Step{
				{
					Container: container("", "golang:latest", "sh", []string{"-c", "mkdir -p $GOPATH/src/$(dirname $REPO_NAME)"}, env, workspaceSourcePath),
				},
				{
					Container: container("", "golang:latest", "sh", []string{"-c", "ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
				},
				{
					Container: container("", "golang:latest", "sh", []string{"-c", "cd $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
				},
			},
		},
	}

	if diff := cmp.Diff(want, task); diff != "" {
		t.Fatalf("PipelineTask doesn't match:\n%s", diff)
	}
}

func TestConvert(t *testing.T) {
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	source := &Source{RepoURL: "https://github.com/bigkevmcd/github-tool.git", Ref: "master"}
	p := &ci.Pipeline{
		Image:     "golang:latest",
		Variables: map[string]string{"REPO_NAME": "github.com/bigkevmcd/github-tool"},
		BeforeScript: []string{
			"mkdir -p $GOPATH/src/$(dirname $REPO_NAME)",
			"ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME",
			"cd $GOPATH/src/$REPO_NAME",
		},
		Stages: []string{
			"test", "build",
		},
		Tasks: []*ci.Task{
			{
				Name:  "format",
				Stage: "test",
				Script: []string{
					"go fmt $(go list ./... | grep -v /vendor/)",
					"go vet $(go list ./... | grep -v /vendor/)",
					"go test -race $(go list ./... | grep -v /vendor/)",
				},
			},
			{
				Name:  "compile",
				Stage: "build",
				Script: []string{
					`go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`,
				},
				Artifacts: ci.Artifacts{
					Paths: []string{"my-test-binary"},
				},
			},
		},
		AfterScript: []string{
			"echo after script",
		},
	}

	pr, err := Convert(p, logger.Sugar(), testConfiguration(), source, "my-volume-claim-123", nil)
	if err != nil {
		t.Fatal(err)
	}

	testEnv := makeEnv(p.Variables)
	// TODO flatten this test
	want := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		ServiceAccountName: testServiceAccountName,
		Workspaces: []pipelinev1.WorkspaceBinding{
			{
				Name: "git-checkout",
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "my-volume-claim-123",
				},
			},
		},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{
				makeGitCloneTask(testEnv, source),
				makeScriptTask(beforeStepTaskName, []string{gitCloneTaskName}, testEnv, p.Image, p.BeforeScript),
				{
					Name: "format-stage-test",
					TaskSpec: &pipelinev1.TaskSpec{
						Steps: []pipelinev1.Step{
							{
								Container: corev1.Container{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go fmt $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
							},
							{
								Container: corev1.Container{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go vet $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
							},
							{
								Container: corev1.Container{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go test -race $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
							},
						},
						Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
					},
					RunAfter:   []string{beforeStepTaskName},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
				},
				{
					Name: "compile-stage-build",
					TaskSpec: &pipelinev1.TaskSpec{
						Steps: []pipelinev1.Step{
							{
								Container: container("", "golang:latest", "sh", []string{"-c", `go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`}, testEnv, workspaceSourcePath),
							},
						},
						Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
					},
					RunAfter:   []string{"format-stage-test"},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
				},
				{
					Name:       "compile-archiver",
					RunAfter:   []string{"format-stage-test"},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
					TaskSpec: &pipelinev1.TaskSpec{
						Steps: []pipelinev1.Step{
							{
								Container: container("compile-archiver-archiver", testArchiverImage, "",
									[]string{"archive", "--bucket-url",
										testArchiveURL, "my-test-binary"}, testEnv, workspaceSourcePath),
							},
						},
						Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
					},
				},
				makeScriptTask(afterStepTaskName, []string{"compile-archiver"}, testEnv, p.Image, p.AfterScript),
			},
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
		},
	})

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func TestConvertWithRules(t *testing.T) {
	source := &Source{RepoURL: "https://github.com/bigkevmcd/github-tool.git", Ref: "refs/pulls/4"}
	p := &ci.Pipeline{
		Image:     "golang:latest",
		Variables: map[string]string{"REPO_NAME": "github.com/bigkevmcd/github-tool"},
		Stages: []string{
			"test",
		},
		Tasks: []*ci.Task{
			{
				Name:  "format",
				Stage: "test",
				Script: []string{
					"go test -race $(go list ./... | grep -v /vendor/)",
				},
				Rules: []ci.Rule{
					{
						If:   `hook.PullRequest.Head.Ref != "master"`,
						When: "never",
					},
				},
			},
		},
	}
	hook := hook.MakeHookFromFixture(t, "../testdata/github_pull_request.json", "pull_request")
	ctx, err := cel.New(hook)
	if err != nil {
		t.Fatal(err)
	}
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr, err := Convert(p, logger.Sugar(), testConfiguration(), source, "my-volume-claim-123", ctx)
	if err != nil {
		t.Fatal(err)
	}

	testEnv := makeEnv(p.Variables)
	// TODO flatten this test
	want := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		ServiceAccountName: testServiceAccountName,
		Workspaces: []pipelinev1.WorkspaceBinding{
			{
				Name: "git-checkout",
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "my-volume-claim-123",
				},
			},
		},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{
				makeGitCloneTask(testEnv, source),
			},
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
		},
	})

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func TestConvertWithTektonTask(t *testing.T) {
	source := &Source{RepoURL: "https://github.com/bigkevmcd/github-tool.git", Ref: "refs/pulls/4"}
	p := &ci.Pipeline{
		Image:     "golang:latest",
		Variables: map[string]string{"REPO_NAME": "github.com/bigkevmcd/github-tool"},
		Stages: []string{
			"test",
		},
		TektonConfig: &ci.TektonConfig{
			ServiceAccountName: "testing",
		},
		Tasks: []*ci.Task{
			{
				Name:  "format",
				Stage: "test",
				Tekton: &ci.TektonTask{
					TaskRef: "my-test-task",
					Params: []ci.TektonTaskParam{
						{Name: "MY_TEST_PARAM", Expression: "vars.CI_COMMIT_BRANCH"},
					},
				},
			},
		},
	}
	hook := hook.MakeHookFromFixture(t, "../testdata/github_pull_request.json", "pull_request")
	ctx, err := cel.New(hook)
	if err != nil {
		t.Fatal(err)
	}
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	pr, err := Convert(p, logger.Sugar(), testConfiguration(), source, "my-volume-claim-123", ctx)
	if err != nil {
		t.Fatal(err)
	}

	testEnv := makeEnv(p.Variables)
	// TODO flatten this test
	want := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		ServiceAccountName: "testing",
		Workspaces: []pipelinev1.WorkspaceBinding{
			{
				Name: "git-checkout",
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "my-volume-claim-123",
				},
			},
		},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Tasks: []pipelinev1.PipelineTask{
				makeGitCloneTask(testEnv, source),
				{
					Name: "format-stage-test",
					TaskRef: &pipelinev1.TaskRef{
						Name: "my-test-task",
						Kind: "Task",
					},
					Params: []pipelinev1.Param{
						{
							Name:  "MY_TEST_PARAM",
							Value: pipelinev1.ArrayOrString{Type: "string", StringVal: "changes"},
						},
					},
					RunAfter:   []string{"git-clone"},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
				},
			},
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
		},
	})

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func TestContainer(t *testing.T) {
	env := []corev1.EnvVar{{Name: "TEST_DIR", Value: "/tmp/test"}}
	got := container("test-name", "test-image", "run", []string{"this"}, env, "/tmp/dir")
	want := corev1.Container{
		Name:       "test-name",
		Image:      "test-image",
		Command:    []string{"run"},
		Env:        env,
		Args:       []string{"this"},
		WorkingDir: "/tmp/dir",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("container doesn't match:\n%s", diff)
	}
}

func TestMakeEnv(t *testing.T) {
	env := makeEnv(map[string]string{
		"TEST_KEY": "test_val",
	})

	want := []corev1.EnvVar{
		{Name: "TEST_KEY", Value: "test_val"},
		{Name: "CI_PROJECT_DIR", Value: "$(workspaces.source.path)"},
	}
	if diff := cmp.Diff(want, env); diff != "" {
		t.Fatalf("env doesn't match:\n%s", diff)
	}

}

func TestMakeScriptSteps(t *testing.T) {
	t.Skip()
}

func testConfiguration() *Configuration {
	return &Configuration{
		PipelineRunPrefix:         testPipelineRunPrefix,
		ArchiverImage:             testArchiverImage,
		ArchiveURL:                testArchiveURL,
		DefaultServiceAccountName: testServiceAccountName,
	}
}
