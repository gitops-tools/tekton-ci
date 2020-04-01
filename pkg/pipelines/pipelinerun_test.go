package pipelines

import (
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/google/go-cmp/cmp"
)

func TestMakeEnv(t *testing.T) {
	t.Skip()
}

func TestMakeScriptSteps(t *testing.T) {
	t.Skip()
}

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
					Container: corev1.Container{
						Image:      image,
						Command:    []string{"sh", "-c", "mkdir -p $GOPATH/src/$(dirname $REPO_NAME)"},
						WorkingDir: "$(workspaces.source.path)",
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "CI_PROJECT_DIR",
								Value: "$(workspaces.source.path)",
							},
						},
					},
				},
				pipelinev1.Step{
					Container: corev1.Container{
						Image:      image,
						Command:    []string{"sh", "-c", "ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME"},
						WorkingDir: "$(workspaces.source.path)",
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "CI_PROJECT_DIR",
								Value: "$(workspaces.source.path)",
							},
						},
					},
				},
				pipelinev1.Step{
					Container: corev1.Container{
						Image:      image,
						Command:    []string{"sh", "-c", "cd $GOPATH/src/$REPO_NAME"},
						WorkingDir: "$(workspaces.source.path)",
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
				Name: "format", Stage: "test", Script: []string{
					"go fmt $(go list ./... | grep -v /vendor/)",
					"go vet $(go list ./... | grep -v /vendor/)",
					"go test -race $(go list ./... | grep -v /vendor/)",
				},
			},
			&ci.Task{
				Name: "compile", Stage: "build", Script: []string{
					`go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`,
				},
			},
		},
		AfterScript: []string{
			"echo after script",
		},
	}

	pr := Convert(p, "my-pipeline-run", source)

	testEnv := makeEnv(p.Variables)
	want := &pipelinev1.PipelineRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: "my-pipeline-run"},
		Spec: pipelinev1.PipelineRunSpec{
			Workspaces: []pipelinev1.WorkspaceBinding{
				pipelinev1.WorkspaceBinding{
					Name: "git-checkout",
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "shared-task-storage",
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
								{
									Container: corev1.Container{
										Image:      "golang:latest",
										Command:    []string{"sh", "-c", "go fmt $(go list ./... | grep -v /vendor/)"},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
								{
									Container: corev1.Container{
										Image:      "golang:latest",
										Command:    []string{"sh", "-c", "go vet $(go list ./... | grep -v /vendor/)"},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
								{
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
					{
						Name: "compile-stage-build",
						TaskSpec: &pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								{
									Container: corev1.Container{
										Image: "golang:latest",
										Command: []string{
											"sh",
											"-c",
											`go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`,
										},
										WorkingDir: "$(workspaces.source.path)",
										Env:        testEnv,
									},
								},
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
						RunAfter:   []string{"format-stage-test"},
						Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
					},
					makeScriptTask("compile-stage-build", afterStepTaskName, testEnv, p.Image, p.AfterScript),
				},
				Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
			},
		},
	}

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}
