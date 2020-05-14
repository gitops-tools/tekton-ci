package watcher

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"

	"github.com/jenkins-x/go-scm/scm"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// State represents the state of a Pipeline.
type State int

const (
	// Pending indicates that the PipelineRun is not yet complete.
	Pending State = iota
	// Failed indicates that at least one Task in the PipelineRun failed.
	Failed
	// Successful indicates that all Tasks in the PipelineRun completed
	// successfully.
	Successful
)

func (s State) String() string {
	names := [...]string{
		"Pending",
		"Failed",
		"Successful"}
	return names[s]
}

// runState returns whether or not a PipelineRun was successful or
// not.
//
// It can return a Pending result if the task has not yet completed.
// TODO: will likely need to work out if a task was killed OOM.
func runState(p *pipelinev1.PipelineRun) State {
	for _, c := range p.Status.Conditions {
		if c.Type == apis.ConditionSucceeded {
			switch c.Status {
			case
				corev1.ConditionFalse:
				return Failed
			case corev1.ConditionTrue:
				return Successful
			case corev1.ConditionUnknown:
				return Pending
			}
		}
	}
	return Pending
}

// convertState converts between pipeline run state, and the commit status.
func convertState(s State) scm.State {
	switch s {
	case Failed:
		return scm.StateFailure
	case Pending:
		return scm.StatePending
	case Successful:
		return scm.StateSuccess
	default:
		return scm.StateUnknown
	}
}
