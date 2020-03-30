package ci

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	parseTests := []struct {
		filename string
		want     *Pipeline
	}{
		{"testdata/go-gitlab-ci.yaml", testCI},
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
