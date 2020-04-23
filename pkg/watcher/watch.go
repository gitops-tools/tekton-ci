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

			for _, tr := range pr.Status.TaskRuns {
				log.Printf("    %#v", tr.Status)
			}
		}
	}
}
