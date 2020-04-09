package dsl

import (
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
	"github.com/google/go-cmp/cmp"
)

const (
	testPipelineRunPrefix = "my-pipeline-run-"
	testArchiverImage     = "quay.io/testing/testing"
	testArchiveURL        = "https://example/com/testing"
)

func TestMakeGitCloneTask(t *testing.T) {
	repoURL := "https://github.com/myorg/testing.git"
	env := []corev1.EnvVar{
		{Name: "CI_PROJECT_DIR", Value: "$(workspaces.source.path)"},
	}
	task := makeGitCloneTask(env, &Source{RepoURL: repoURL, Ref: "master"})

	want := pipelinev1.PipelineTask{
		Name: gitCloneTaskName,
		Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{
			pipelinev1.WorkspacePipelineTaskBinding{Name: "source", Workspace: workspaceName},
		},
		TaskSpec: &pipelinev1.TaskSpec{
			Workspaces: []pipelinev1.WorkspaceDeclaration{
				pipelinev1.WorkspaceDeclaration{
					Name: "source",
				},
			},
			Steps: []pipelinev1.Step{
				pipelinev1.Step{
					Container: corev1.Container{
						Name:    "git-clone",
						Image:   "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
						Command: []string{"/ko-app/git-init", "-url", repoURL, "-revision", "master", "-path", workspaceSourcePath},
						Env: []corev1.EnvVar{
							corev1.EnvVar{
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

	task := makeScriptTask(gitCloneTaskName, "test-script-task", env, image, beforeScript)
	want := pipelinev1.PipelineTask{
		Name:     "test-script-task",
		RunAfter: []string{gitCloneTaskName},
		Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{
			pipelinev1.WorkspacePipelineTaskBinding{Name: "source", Workspace: workspaceName},
		},
		TaskSpec: &pipelinev1.TaskSpec{
			Workspaces: []pipelinev1.WorkspaceDeclaration{
				pipelinev1.WorkspaceDeclaration{
					Name: "source",
				},
			},
			Steps: []pipelinev1.Step{
				pipelinev1.Step{
					Container: container("", "golang:latest", []string{"sh", "-c", "mkdir -p $GOPATH/src/$(dirname $REPO_NAME)"}, env, workspaceSourcePath),
				},
				pipelinev1.Step{
					Container: container("", "golang:latest", []string{"sh", "-c", "ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
				},
				pipelinev1.Step{
					Container: container("", "golang:latest", []string{"sh", "-c", "cd $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
				},
			},
		},
	}

	if diff := cmp.Diff(want, task); diff != "" {
		t.Fatalf("PipelineTask doesn't match:\n%s", diff)
	}
}

// TODO: PersistentVolumeClaim and name
func TestConvert(t *testing.T) {
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
			&ci.Task{
				Name:  "format",
				Stage: "test",
				Script: []string{
					"go fmt $(go list ./... | grep -v /vendor/)",
					"go vet $(go list ./... | grep -v /vendor/)",
					"go test -race $(go list ./... | grep -v /vendor/)",
				},
			},
			&ci.Task{
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

	pr := Convert(p, testConfiguration(), source, "my-volume-claim-123")

	testEnv := makeEnv(p.Variables)
	// TODO flatten this test
	want := &pipelinev1.PipelineRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "", GenerateName: "my-pipeline-run-", Annotations: resources.Annotations("dsl")},
		Spec: pipelinev1.PipelineRunSpec{
			Workspaces: []pipelinev1.WorkspaceBinding{
				pipelinev1.WorkspaceBinding{
					Name: "git-checkout",
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "my-volume-claim-123",
					},
				},
			},
			PipelineSpec: &pipelinev1.PipelineSpec{
				Tasks: []pipelinev1.PipelineTask{
					makeGitCloneTask(testEnv, source),
					makeScriptTask(gitCloneTaskName, beforeStepTaskName, testEnv, p.Image, p.BeforeScript),
					pipelinev1.PipelineTask{
						Name: "format-stage-test",
						TaskSpec: &pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								pipelinev1.Step{
									Container: corev1.Container{
										Image:      "golang:latest",
										Command:    []string{"sh", "-c", "go fmt $(go list ./... | grep -v /vendor/)"},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
								pipelinev1.Step{
									Container: corev1.Container{
										Image:      "golang:latest",
										Command:    []string{"sh", "-c", "go vet $(go list ./... | grep -v /vendor/)"},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
								pipelinev1.Step{
									Container: corev1.Container{
										Image:      "golang:latest",
										Command:    []string{"sh", "-c", "go test -race $(go list ./... | grep -v /vendor/)"},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
						RunAfter:   []string{"before-step"},
						Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
					},
					pipelinev1.PipelineTask{
						Name: "compile-stage-build",
						TaskSpec: &pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								pipelinev1.Step{
									Container: container("", "golang:latest", []string{"sh", "-c", `go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`}, testEnv, workspaceSourcePath),
								},
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
						RunAfter:   []string{"format-stage-test"},
						Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
					},
					pipelinev1.PipelineTask{
						Name:       "compile-archiver",
						RunAfter:   []string{"format-stage-test"},
						Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
						TaskSpec: &pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								pipelinev1.Step{
									Container: container("compile-archiver-archiver", testArchiverImage, []string{"./archiver", "--url", testArchiveURL, "my-test-binary"}, testEnv, workspaceSourcePath),
								},
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
					},
					makeScriptTask("compile-archiver", afterStepTaskName, testEnv, p.Image, p.AfterScript),
				},
				Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
			},
		},
	}

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func TestContainer(t *testing.T) {
	env := []corev1.EnvVar{corev1.EnvVar{Name: "TEST_DIR", Value: "/tmp/test"}}
	got := container("test-name", "test-image", []string{"run", "this"}, env, "/tmp/dir")
	want := corev1.Container{
		Name:       "test-name",
		Image:      "test-image",
		Command:    []string{"run", "this"},
		Env:        env,
		WorkingDir: "/tmp/dir",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Container doesn't match:\n%s", diff)
	}
}

func TestMakeEnv(t *testing.T) {
	t.Skip()
}

func TestMakeScriptSteps(t *testing.T) {
	t.Skip()
}

func testConfiguration() *Configuration {
	return &Configuration{
		PipelineRunPrefix: testPipelineRunPrefix,
		ArchiverImage:     testArchiverImage,
		ArchiveURL:        testArchiveURL,
	}
}
