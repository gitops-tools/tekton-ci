package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLabels(t *testing.T) {
	want := map[string]string{
		"app.kubernetes.io/managed-by": "dsl",
		"app.kubernetes.io/part-of":    "Tekton-CI",
	}

	if diff := cmp.Diff(want, labels("dsl")); diff != "" {
		t.Fatalf("labels failed: %s\n", diff)
	}
}
