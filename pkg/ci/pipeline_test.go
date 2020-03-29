package ci

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	compileJob = &Job{
		Name:  "compile",
		Stage: "build",
		Script: []string{
			`go build -race -ldflags "-extldflags '-static'" -o $CI_PROJECT_DIR/mybinary`,
		},
	}

	formatJob = &Job{
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
		Jobs: []*Job{
			compileJob,
			formatJob,
		},
	}
)

func TestJobsForStage(t *testing.T) {
	want := map[string][]string{
		"test":  []string{"format"},
		"build": []string{"compile"},
	}

	for k, want := range want {
		if diff := cmp.Diff(want, testCI.JobsForStage(k)); diff != "" {
			t.Errorf("JobsForStage(%v) failed diff\n%s", k, diff)
		}
	}
}

func TestJob(t *testing.T) {
	tests := map[string]*Job{"compile": compileJob, "format": formatJob}

	for k, want := range tests {
		if diff := cmp.Diff(want, testCI.Job(k)); diff != "" {
			t.Errorf("Jobs(%v) failed diff\n%s", k, diff)
		}
	}
}
