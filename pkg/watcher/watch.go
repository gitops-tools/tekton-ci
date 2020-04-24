package watcher

import (
	"log"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WatchPipelineRuns(c pipelineclientset.Interface, ns string) {
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
			commit, err := FindCommit(pr)
			if err != nil {
				log.Printf("got an error finding the commit: %s", err)
				continue
			}
			log.Printf("identified the commit as: %s", commit)
		}
	}
}

func FindCommit(pr *pipelinev1.PipelineRun) (string, error) {
	for name, tr := range pr.Status.TaskRuns {
		for _, v := range tr.Status.ResourcesResult {
			if v.Key == "commit" {
				return v.Value, nil
			}
		}
	}

	return "", nil
}
