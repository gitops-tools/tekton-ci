package pipelinerun

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	hookGUID = "3ae36614-ca30-45e0-b579-3b2e857dc964"
	testSHA  = "93c48c7c0acefc114e2370ba93290264d8fec9f8"
)

func TestExecute(t *testing.T) {
	d := readDefinition(t, "testdata/example.yaml")
	hook := scm.PullRequestHook{
		Action: scm.ActionOpen,
		Repo:   scm.Repository{},
		PullRequest: scm.PullRequest{
			Sha: testSHA,
		},
		GUID: hookGUID,
	}

	pr, err := Execute(d, hook, "new-pipeline-run")
	if err != nil {
		t.Fatal(err)
	}
	want := &pipelinev1.PipelineRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "", GenerateName: "new-pipeline-run-"},
		Spec: pipelinev1.PipelineRunSpec{
			Params: []pipelinev1.Param{
				pipelinev1.Param{
					Name:  "COMMIT_SHA",
					Value: pipelinev1.ArrayOrString{StringVal: testSHA, Type: "string"},
				},
			},
			PipelineSpec: testPipelineSpec,
		},
	}
	if diff := cmp.Diff(want, pr); diff != "" {
		t.Fatalf("PipelineRun doesn't match:\n%s", diff)
	}
}

func readDefinition(t *testing.T, filename string) *PipelineDefinition {
	t.Helper()
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	d, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
