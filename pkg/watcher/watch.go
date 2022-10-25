package watcher

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelsv1 "k8s.io/apimachinery/pkg/labels"

	"github.com/gitops-tools/tekton-ci/pkg/dsl"
	"github.com/gitops-tools/tekton-ci/pkg/logger"
)

const (
	tektonCILabel               = "tekton-ci"
	notificationStateAnnotation = "tekton.dev/ci-notification-state"
)

// WatchPipelineRuns tracks PipelineRuns with the correct label, and reports
// their state as a commit-status to the upstream Git hosting service.
func WatchPipelineRuns(stop <-chan struct{}, scmClient *scm.Client, tektonClient pipelineclientset.Interface, ns string, l logger.Logger) {
	l.Infow("starting to watch for PipelineRuns", "ns", ns)
	api := tektonClient.TektonV1().PipelineRuns(ns)
	listOptions := metav1.ListOptions{
		LabelSelector: labelsv1.Set(map[string]string{"app.kubernetes.io/part-of": "Tekton-CI"}).AsSelector().String(),
	}
	ctx := context.Background()
	watcher, err := api.Watch(ctx, listOptions)
	if err != nil {
		l.Errorf("failed to watch PipelineRuns: %s", err)
		return
	}
	ch := watcher.ResultChan()

	for {
		select {
		case <-stop:
			return
		case v := <-ch:
			pr := v.Object.(*pipelinev1.PipelineRun)
			err := handlePipelineRun(ctx, scmClient, tektonClient, pr, l)
			if err != nil {
				l.Infow(fmt.Sprintf("error handling PipelineRun: %s", err), "name", pr.ObjectMeta.Name)
			}
		}
	}
}

func handlePipelineRun(ctx context.Context, scmClient *scm.Client, tektonClient pipelineclientset.Interface, pr *pipelinev1.PipelineRun, l logger.Logger) error {
	newState := runState(pr)
	l.Infof("Received a PipelineRun %#v %s", pr.Status, newState)
	if newState.String() != notificationState(pr) {
		err := sendNotification(scmClient, pr, l)
		if err != nil {
			return fmt.Errorf("failed to send notification %w", err)
		}
	}
	return updatePRState(ctx, newState, pr, tektonClient)
}

func updatePRState(ctx context.Context, newState State, pr *pipelinev1.PipelineRun, tektonClient pipelineclientset.Interface) error {
	setNotificationState(pr, newState)
	_, err := tektonClient.TektonV1().PipelineRuns(pr.ObjectMeta.Namespace).Update(ctx, pr, metav1.UpdateOptions{})
	return err
}

func notificationState(pr *pipelinev1.PipelineRun) string {
	return pr.ObjectMeta.Annotations[notificationStateAnnotation]
}

func setNotificationState(pr *pipelinev1.PipelineRun, s State) {
	pr.ObjectMeta.Annotations[notificationStateAnnotation] = s.String()
}

func sendNotification(c *scm.Client, pr *pipelinev1.PipelineRun, l logger.Logger) error {
	repo, err := parseRepoFromURL(findRepoURL(pr))
	if err != nil {
		return err
	}
	status := commitStatusInput(pr)
	commit := findCommit(pr)
	if commit == "" {
		return errors.New("could not find a commit-id in the PipelineRun")
	}

	l.Infof("sendNotification", "repo", repo, "status", status, "commit", commit)
	s, _, err := c.Repositories.CreateStatus(context.Background(), repo, commit, status)
	if err != nil {
		return fmt.Errorf("failed to create status: %w", err)
	}

	l.Infof("sendNotification status created: %#v\n", s)
	return nil
}

func findCommit(pr *pipelinev1.PipelineRun) string {
	return pr.GetAnnotations()[dsl.CISourceRefAnnotation]
}

func findRepoURL(pr *pipelinev1.PipelineRun) string {
	return pr.GetAnnotations()[dsl.CISourceURLAnnotation]
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
