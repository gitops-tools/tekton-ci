package ci

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	compileTask = &Task{
		Name:  "compile",
		Stage: "build",
		Script: []string{
			`go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`,
		},
	}

	formatTask = &Task{
		Name:  "format",
		Stage: "test",
		Script: []string{
			"go fmt $(go list ./... | grep -v /vendor/)",
			"go vet $(go list ./... | grep -v /vendor/)",
			"go test -race $(go list ./... | grep -v /vendor/)",
		},
	}

	testCI = &Pipeline{
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
		Tasks: []*Task{
			compileTask,
			formatTask,
		},
	}
)

func TestTasksForStage(t *testing.T) {
	want := map[string][]string{
		"test":  []string{"format"},
		"build": []string{"compile"},
	}

	for k, want := range want {
		if diff := cmp.Diff(want, testCI.TasksForStage(k)); diff != "" {
			t.Errorf("TasksForStage(%v) failed diff\n%s", k, diff)
		}
	}
}

func TestTask(t *testing.T) {
	tests := map[string]*Task{"compile": compileTask, "format": formatTask}

	for k, want := range tests {
		if diff := cmp.Diff(want, testCI.Task(k)); diff != "" {
			t.Errorf("Tasks(%v) failed diff\n%s", k, diff)
		}
	}
}
