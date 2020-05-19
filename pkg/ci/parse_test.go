package ci

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	parseTests := []struct {
		filename string
		want     *Pipeline
	}{
		{"testdata/go-gitlab-ci.yaml", testCI},
		{"testdata/after-script-example.yaml", &Pipeline{
			Image:       "golang:latest",
			AfterScript: []string{`echo "testing"`},
			Stages:      []string{"test"},
			Tasks: []*Task{
				{Name: "format",
					Stage:  "test",
					Script: []string{`echo "testing"`},
				},
			},
		}},
		{"testdata/simple.yaml", &Pipeline{
			Image:  "golang:latest",
			Stages: []string{DefaultStage},
			Tasks: []*Task{
				{Name: "format",
					Stage:  DefaultStage,
					Script: []string{`echo "testing"`},
				},
			},
		}},
		{"testdata/script-with-rules.yaml", &Pipeline{
			Image:  "golang:latest",
			Stages: []string{DefaultStage},
			Tasks: []*Task{
				{Name: "format",
					Stage:  DefaultStage,
					Script: []string{`echo "testing"`},
					Rules: []Rule{
						{
							If:   `vars.CI_COMMIT_BRANCH != "master"`,
							When: "never",
						},
						{
							If:   `hook.Forced == true`,
							When: "manual",
						},
					},
				},
			},
		}},
		{"testdata/tekton-task.yaml", &Pipeline{
			Image:  "golang:latest",
			Stages: []string{DefaultStage},
			TektonConfig: &TektonConfig{
				ServiceAccountName: "testing",
			},
			Tasks: []*Task{
				{Name: "format",
					Stage: DefaultStage,
					Tekton: &TektonTask{
						TaskRef: "my-test-task",
						Params: []TektonTaskParam{
							{
								Name:       "IMAGE_URL",
								Expression: "quay.io/testing/testing",
							},
						},
					},
				},
			},
		}},
		{"testdata/simple-with-jobs.yaml", &Pipeline{
			Image:  "golang:latest",
			Stages: []string{DefaultStage},
			Tasks: []*Task{
				{Name: "format",
					Stage: DefaultStage,
					Tekton: &TektonTask{
						Jobs: []map[string]string{
							{"CI_NODE_INDEX": "0"},
							{"CI_NODE_INDEX": "1"},
						},
					},
					Script: []string{`echo "testing"`},
				},
			},
		}},
		{"testdata/simple-with-tekton-image.yaml", &Pipeline{
			Image:  "alpine",
			Stages: []string{DefaultStage},
			Tasks: []*Task{
				{Name: "format",
					Stage:  DefaultStage,
					Script: []string{`echo "testing"`},
					Tekton: &TektonTask{
						Image: "golang:latest",
					},
				},
			},
		}},
		{"testdata/simple-with-cache.yaml", &Pipeline{
			Image: "golang:latest",
			Cache: &CacheConfig{
				Key:    "one-key-to-rule-them-all",
				Policy: "pull-push",
				Paths:  []string{"node_modules/", "public/", "vendor/"},
			},
			Stages: []string{DefaultStage},
			Tasks: []*Task{
				{Name: "format",
					Stage:  DefaultStage,
					Script: []string{`echo "testing"`},
				},
			},
		}},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing %s", tt.filename), func(rt *testing.T) {
			f, err := os.Open(tt.filename)
			if err != nil {
				rt.Errorf("failed to open %v: %s", tt.filename, err)
			}
			defer f.Close()

			got, err := Parse(f)
			if err != nil {
				rt.Errorf("failed to parse %v: %s", tt.filename, err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("Parse(%s) failed diff\n%s", tt.filename, diff)
			}
		})
	}
}

func TestParseBadFiles(t *testing.T) {
	parseTests := []struct {
		filename string
		errMsg   string
	}{
		{"testdata/bad-task-no-script.yaml", `invalid task "format": missing script`},
		{"testdata/bad-tekton-task.yaml", `invalid task "format": provided Tekton taskRef and script`},
		{"testdata/bad-tekton-task-params.yaml", `bad Tekton task parameter`},
		{"testdata/bad-tekton-jobs.yaml", `could not parse CI_NODE_INDEX==0 as an environment variable`},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing %s", tt.filename), func(rt *testing.T) {
			f, err := os.Open(tt.filename)
			if err != nil {
				rt.Errorf("failed to open %v: %s", tt.filename, err)
			}
			defer f.Close()

			_, err = Parse(f)
			if !matchError(t, tt.errMsg, err) {
				rt.Errorf("error match failed, got %s, want %s", err, tt.errMsg)
			}
		})
	}
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	if s == "" && e == nil {
		return true
	}
	if s != "" && e == nil {
		return false
	}
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
