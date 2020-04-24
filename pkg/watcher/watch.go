package watcher

import (
	"log"

	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const TektonCILabel = "tekton-ci"

func WatchPipelineRuns(done chan struct{}, c pipelineclientset.Interface, ns string) {
	log.Println("starting to watch for pipelineruns")
	api := c.TektonV1beta1().PipelineRuns(ns)
	listOptions := metav1.ListOptions{}
	watcher, err := api.Watch(listOptions)
	if err != nil {
		log.Fatalf("failed to watch pipelineruns: %s\n", err)
	}
	ch := watcher.ResultChan()

	for {
		select {
		case v := <-ch:
			pr := v.Object.(*pipelinev1.PipelineRun)
			log.Printf("Received a PipelineRun %#v %s", pr.Status, runState(pr))
			commit := findCommit(pr)
			if commit != "" {
				log.Printf("identified the commit as: %s", commit)
			}
			repoURL := findRepoURL(pr)
			if repoURL != "" {
				log.Printf("identified the repoURL as: %s", repoURL)
			}
		}
	}
}

func findCommit(pr *pipelinev1.PipelineRun) string {
	for _, tr := range pr.Status.TaskRuns {
		for _, v := range tr.Status.ResourcesResult {
			if v.Key == "commit" {
				return v.Value
			}
		}
	}
	return ""
}

func findRepoURL(pr *pipelinev1.PipelineRun) string {
	return pr.ObjectMeta.Annotations["tekton.dev/ci-source-url"]
}

func commitStatusInput(pr *pipelinev1.PipelineRun) *scm.StatusInput {
	return &scm.StatusInput{
		State: convertState(runState(pr)),
		Label: TektonCILabel,
		Desc:  "Tekton CI Status",
	}
}
