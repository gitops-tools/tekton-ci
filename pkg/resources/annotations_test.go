package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAnnotations(t *testing.T) {
	want := map[string]string{
		"tekton.dev/git-status":        "true",
		"tekton.dev/status-context":    "tekton-ci",
		"app.kubernetes.io/managed-by": "dsl",
		"app.kubernetes.io/part-of":    "Tekton-CI",
	}

	if diff := cmp.Diff(want, Annotations("dsl")); diff != "" {
		t.Fatalf("annotations failed: %s\n", diff)
	}
}
