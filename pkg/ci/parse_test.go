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
			Jobs: []*Job{
				&Job{Name: "format",
					Stage:  "test",
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
		{"testdata/bad-job-no-script.yaml", `invalid job "format" missing script`},
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
