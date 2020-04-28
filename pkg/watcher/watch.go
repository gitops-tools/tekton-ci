package watcher

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelsv1 "k8s.io/apimachinery/pkg/labels"

	"github.com/bigkevmcd/tekton-ci/pkg/logger"
)

const (
	tektonCILabel               = "tekton-ci"
	notificationStateAnnotation = "tekton.dev/ci-notification-state"
)

func WatchPipelineRuns(scmClient *scm.Client, tektonClient pipelineclientset.Interface, ns string, l logger.Logger) error {
	l.Infow("starting to watch for pipelineruns", "ns", ns)
	api := tektonClient.TektonV1beta1().PipelineRuns(ns)
	listOptions := metav1.ListOptions{
		LabelSelector: labelsv1.Set(map[string]string{"app.kubernetes.io/part-of": "Tekton-CI"}).AsSelector().String(),
	}
	watcher, err := api.Watch(listOptions)
	if err != nil {
		l.Errorf("failed to watch pipelineruns: %s\n", err)
		return err
	}
	ch := watcher.ResultChan()

	for {
		select {
		case v := <-ch:
			pr := v.Object.(*pipelinev1.PipelineRun)
			err := handlePipelineRun(scmClient, tektonClient, pr, l)
			if err != nil {
				l.Infow(fmt.Sprintf("error handling PipelineRun: %s", err), "name", pr.ObjectMeta.Name)
			}
		}
	}
}

func handlePipelineRun(scmClient *scm.Client, tektonClient pipelineclientset.Interface, pr *pipelinev1.PipelineRun, l logger.Logger) error {
	state := runState(pr)
	l.Infof("Received a PipelineRun %#v %s", pr.Status, state)
	if state.String() != findNotificationState(pr) {
		err := sendNotification(scmClient, pr, l)
		if err != nil {
			return fmt.Errorf("failed to send notification %w", err)
		}
	}
	return nil
}

func findNotificationState(pr *pipelinev1.PipelineRun) string {
	return pr.ObjectMeta.Annotations[notificationStateAnnotation]
}

func sendNotification(c *scm.Client, pr *pipelinev1.PipelineRun, l logger.Logger) error {
	repo, err := parseRepoFromURL(findRepoURL(pr))
	if err != nil {
		return err
	}
	// TODO: this should check for empty
	status := commitStatusInput(pr)
	commit := findCommit(pr)

	l.Infof("sendNotification", "repo", repo, "status", status, "commit", commit)
	s, _, err := c.Repositories.CreateStatus(context.Background(), repo, commit, status)
	if err != nil {
		return fmt.Errorf("failed to create status: %w", err)
	}
	l.Infof("sendNotification status created: %#v\n", s)
	return nil
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
		Label: tektonCILabel,
		Desc:  "Tekton CI Status",
	}
}

func parseRepoFromURL(s string) (string, error) {
	p, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %s: %s", s, err)
	}
	parts := strings.Split(p.Path, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid repo URL: %s", s)
	}
	return strings.Join([]string{parts[1], strings.TrimSuffix(parts[2], ".git")}, "/"), nil
}
