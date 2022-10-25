package dsl

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	"github.com/gitops-tools/tekton-ci/pkg/cel"
	"github.com/gitops-tools/tekton-ci/pkg/ci"
	"github.com/gitops-tools/tekton-ci/pkg/resources"
	"github.com/gitops-tools/tekton-ci/test/hook"
	"github.com/google/go-cmp/cmp"
)

const (
	testPipelineRunPrefix  = "my-pipeline-run-"
	testArchiverImage      = "quay.io/testing/testing"
	testArchiveURL         = "https://example/com/testing"
	testRepoURL            = "https://github.com/myorg/testing.git"
	testServiceAccountName = "test-account"
	testEvtID              = "26400635-d8f4-4cf5-a45f-bd03856bdf2b"
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
		TaskSpec: &pipelinev1.EmbeddedTask{
			TaskSpec: pipelinev1.TaskSpec{
				Workspaces: []pipelinev1.WorkspaceDeclaration{
					{
						Name: "source",
					},
				},
				Steps: []pipelinev1.Step{
					{
						Name:    "git-clone",
						Image:   "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
						Command: []string{"/ko-app/git-init", "-url", testRepoURL, "-revision", "master", "-path", workspaceSourcePath},
						Env: []corev1.EnvVar{
							{
								Name:  "CI_PROJECT_DIR",
								Value: "$(workspaces.source.path)",
							},
							{
								Name:  "TEKTON_RESOURCE_NAME",
								Value: "tekton-ci-git-clone",
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
		TaskSpec: &pipelinev1.EmbeddedTask{
			TaskSpec: pipelinev1.TaskSpec{
				Workspaces: []pipelinev1.WorkspaceDeclaration{
					{
						Name: "source",
					},
				},
				Steps: []pipelinev1.Step{
					step("", "golang:latest", "sh", []string{"-c", "mkdir -p $GOPATH/src/$(dirname $REPO_NAME)"}, env, workspaceSourcePath),
					step("", "golang:latest", "sh", []string{"-c", "ln -svf $CI_PROJECT_DIR $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
					step("", "golang:latest", "sh", []string{"-c", "cd $GOPATH/src/$REPO_NAME"}, env, workspaceSourcePath),
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
				Tekton: &ci.TektonTask{
					Image: "test-compile-image",
				},
			},
		},
		AfterScript: []string{
			"echo after script",
		},
	}

	pr, err := Convert(p, logger.Sugar(), testConfiguration(), source, "my-volume-claim-123", nil, testEvtID)
	if err != nil {
		t.Fatal(err)
	}

	testEnv := makeEnv(p.Variables)
	// TODO flatten this test
	want := resources.PipelineRun("dsl", "my-pipeline-run-", pipelinev1.PipelineRunSpec{
		TaskRunTemplate: pipelinev1.PipelineTaskRunTemplate{
			ServiceAccountName: "test-account",
		},
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
					TaskSpec: &pipelinev1.EmbeddedTask{
						TaskSpec: pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go fmt $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
								{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go vet $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
								{
									Image:      "golang:latest",
									Command:    []string{"sh"},
									Args:       []string{"-c", "go test -race $(go list ./... | grep -v /vendor/)"},
									WorkingDir: "$(workspaces.source.path)",
									Env:        testEnv,
								},
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
					},
					RunAfter:   []string{beforeStepTaskName},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
				},
				{
					Name: "compile-stage-build",
					TaskSpec: &pipelinev1.EmbeddedTask{
						TaskSpec: pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								step("", "test-compile-image", "sh", []string{"-c", `go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`}, testEnv, workspaceSourcePath),
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
					},
					RunAfter:   []string{"format-stage-test"},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
				},
				{
					Name:       "compile-archiver",
					RunAfter:   []string{"format-stage-test"},
					Workspaces: []pipelinev1.WorkspacePipelineTaskBinding{{Name: "source", Workspace: "git-checkout"}},
					TaskSpec: &pipelinev1.EmbeddedTask{
						TaskSpec: pipelinev1.TaskSpec{
							Steps: []pipelinev1.Step{
								step("compile-archiver-archiver", testArchiverImage, "",
									[]string{"archive", "--bucket-url",
										testArchiveURL, "my-test-binary"}, testEnv, workspaceSourcePath),
							},
							Workspaces: []pipelinev1.WorkspaceDeclaration{{Name: "source"}},
						},
					},
				},
				makeScriptTask(afterStepTaskName, []string{"compile-archiver"}, testEnv, p.Image, p.AfterScript),
			},
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{{Name: "git-checkout"}},
		},
	}, AnnotateSource(testEvtID, source))

	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func TestConvertFixtures(t *testing.T) {
	convertTests := []struct {
		name string
	}{
		{"script_with_rules"},
		{"pipeline_with_tekton_task"},
		{"script_with_job_matrix"},
	}

	for _, tt := range convertTests {
		t.Run(tt.name, func(rt *testing.T) {
			source := &Source{RepoURL: "https://github.com/bigkevmcd/github-tool.git", Ref: "refs/pulls/4"}
			p := readPipelineFixture(t, fmt.Sprintf("testdata/%s.yaml", tt.name))
			hook := hook.MakeHookFromFixture(rt, "../testdata/github_push.json", "push")
			ctx, err := cel.New(hook)
			if err != nil {
				t.Fatal(err)
			}
			logger := zaptest.NewLogger(rt, zaptest.Level(zap.WarnLevel))
			pr, err := Convert(p, logger.Sugar(), testConfiguration(), source, "my-volume-claim-123", ctx, testEvtID)
			if err != nil {
				t.Fatal(err)
			}
			want := readPipelineRunFixture(rt, fmt.Sprintf("testdata/%s_pipeline_run.yaml", tt.name))
			if diff := cmp.Diff(want, pr); diff != "" {
				t.Errorf("PipelineRun %s doesn't match:\n%s", tt.name, diff)
			}
		})
	}

}

func TestContainer(t *testing.T) {
	env := []corev1.EnvVar{{Name: "TEST_DIR", Value: "/tmp/test"}}
	got := step("test-name", "test-image", "run", []string{"this"}, env, "/tmp/dir")
	want := pipelinev1.Step{
		Name:       "test-name",
		Image:      "test-image",
		Command:    []string{"run"},
		Env:        env,
		Args:       []string{"this"},
		WorkingDir: "/tmp/dir",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("step doesn't match:\n%s", diff)
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

func TestMakeTaskEnvMatrix(t *testing.T) {
	root := []corev1.EnvVar{
		{Name: "TEST_KEY", Value: "test_val"},
		{Name: "CI_PROJECT_DIR", Value: "$(workspaces.source.path)"},
	}

	varTests := []struct {
		jobs []map[string]string
		want [][]corev1.EnvVar
	}{
		{
			nil,
			[][]corev1.EnvVar{root},
		},
		{
			[]map[string]string{{"TESTING": "test1"}},
			[][]corev1.EnvVar{append(root, corev1.EnvVar{Name: "TESTING", Value: "test1"})},
		},
		{
			[]map[string]string{{"TESTING": "test1"}, {"TESTING": "test2"}},
			[][]corev1.EnvVar{
				append(root, corev1.EnvVar{Name: "TESTING", Value: "test1"}),
				append(root, corev1.EnvVar{Name: "TESTING", Value: "test2"}),
			},
		},
	}
	for _, tt := range varTests {
		task := &ci.Task{
			Name: "format",
			Tekton: &ci.TektonTask{
				Jobs: tt.jobs,
			},
		}

		got := makeTaskEnvMatrix(root, task)
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Fatalf("EnvVars don't match:\n%s", diff)
		}
	}
}

func TestAnnotateSource(t *testing.T) {
	cloneURL := "https://github.com/bigkevmcd/tekton-ci.git"
	ref := "refs/heads/master"
	src := &Source{RepoURL: cloneURL, Ref: ref}
	pr := resources.PipelineRun("dsl", "test-", pipelinev1.PipelineRunSpec{}, AnnotateSource(testEvtID, src))

	want := map[string]string{
		"tekton.dev/ci-source-url": cloneURL,
		"tekton.dev/ci-source-ref": ref,
		"tekton.dev/ci-hook-id":    testEvtID,
	}
	if diff := cmp.Diff(want, pr.ObjectMeta.Annotations); diff != "" {
		t.Fatalf("Source() failed: %s\n", diff)
	}
}

func TestParamsToParams(t *testing.T) {
	t.Skip()
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
		VolumeSize:                resource.MustParse("1G"),
	}
}

func readPipelineFixture(t *testing.T, filename string) *ci.Pipeline {
	t.Helper()
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open %s: %s", filename, err)
	}
	defer f.Close()
	p, err := ci.Parse(f)
	if err != nil {
		t.Fatalf("failed to parse %s: %s", filename, err)
	}
	return p
}

func readPipelineRunFixture(t *testing.T, filename string) *pipelinev1.PipelineRun {
	t.Helper()
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read %s: %s", filename, err)
	}
	var pr *pipelinev1.PipelineRun
	err = yaml.Unmarshal(b, &pr)
	if err != nil {
		t.Fatalf("failed to unmarshal %s: %s", filename, err)
	}
	return pr
}
