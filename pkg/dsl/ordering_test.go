package dsl

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
)

func TestTaskOrdering(t *testing.T) {
	orderingTests := []struct {
		name   string
		before bool
		after  bool
		stages []string
		tasks  []testTask
		want   []testTask
	}{
		{"single task", false, false, []string{},
			[]testTask{{"build", "", ""}},
			[]testTask{{"git-clone", "", ""}, {"build-stage-default", "", "git-clone"}},
		},
		{"before task", true, false, []string{},
			[]testTask{{"build", "", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"before-step", "", "git-clone"},
				{"build-stage-default", "", "before-step"},
			},
		},
		{"after task", false, true, []string{},
			[]testTask{{"build", "", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"build-stage-default", "", "git-clone"},
				{"after-step", "", "build-stage-default"},
			},
		},
		// TODO: fix this flakey test
		{"tasks in different stages", false, false, []string{},
			[]testTask{{"build", "stage-a", ""}, {"test", "stage-b", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"build-stage-stage-a", "", "git-clone"},
				{"test-stage-stage-b", "", "build-stage-stage-a"},
			},
		},
		{"tasks in the same stage", false, false, []string{},
			[]testTask{{"lint", "test", ""}, {"test", "test", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"lint-stage-test", "", "git-clone"},
				{"test-stage-test", "", "git-clone"},
			},
		},
		{"tasks in the same stage and after script", false, true, []string{},
			[]testTask{{"lint", "test", ""}, {"test", "test", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"lint-stage-test", "", "git-clone"},
				{"test-stage-test", "", "git-clone"},
				{"after-step", "", "lint-stage-test,test-stage-test"},
			},
		},
		{"tasks in different stages, explicit ordering", false, false,
			[]string{"stage-b", "stage-a"},
			[]testTask{{"build", "stage-a", ""}, {"test", "stage-b", ""}},
			[]testTask{
				{"git-clone", "", ""},
				{"test-stage-stage-b", "", "git-clone"},
				{"build-stage-stage-a", "", "test-stage-stage-b"},
			},
		},
	}

	for _, tt := range orderingTests {
		t.Run(tt.name, func(rt *testing.T) {
			logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
			p := makeOrderingPipeline(tt.before, tt.after, tt.stages, tt.tasks)
			src := &Source{RepoURL: testRepoURL, Ref: "master"}
			pr, err := Convert(p, logger.Sugar(), testConfiguration(), src, "test-volume", nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, tasksFromPipelineRun(pr)); diff != "" {
				rt.Errorf("%s failed diff: %s\n", tt.name, diff)
			}
		})
	}
}

type testTask struct {
	Name  string
	Stage string
	After string
}

func tasksFromPipelineRun(pr *pipelinev1.PipelineRun) []testTask {
	tasks := []testTask{}
	for _, t := range pr.Spec.PipelineSpec.Tasks {
		tasks = append(tasks, testTask{t.Name, "", strings.Join(t.RunAfter, ",")})
	}
	return tasks
}

func makeOrderingPipeline(before, after bool, stages []string, tasks []testTask) *ci.Pipeline {
	p := &ci.Pipeline{
		Stages: stages,
	}
	if before {
		p.BeforeScript = []string{"echo before"}
	}

	if after {
		p.AfterScript = []string{"echo after"}
	}
	ciTasks := []*ci.Task{}
	for _, t := range tasks {
		task := &ci.Task{
			Name:   t.Name,
			Stage:  t.Stage,
			Script: []string{"echo hello"},
		}
		if task.Stage == "" {
			task.Stage = "default"
		}
		ciTasks = append(ciTasks, task)
	}
	p.Tasks = ciTasks
	p.Stages = stages
	if len(stages) == 0 {
		p.Stages = findStages(p.Tasks)
	}
	return p
}

func findStages(tasks []*ci.Task) []string {
	foundStages := map[string]bool{}
	for _, t := range tasks {
		foundStages[t.Stage] = true
	}
	stages := []string{}
	for k := range foundStages {
		stages = append(stages, k)
	}
	if len(stages) > 0 {
		return stages
	}
	return []string{ci.DefaultStage}
}
