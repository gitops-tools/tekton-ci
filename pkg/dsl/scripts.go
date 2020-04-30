package dsl

import (
	"fmt"

	"github.com/google/cel-go/common/types"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/cel"
	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/logger"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

const (
	gitCloneTaskName      = "git-clone"
	beforeStepTaskName    = "before-step"
	afterStepTaskName     = "after-step"
	workspaceName         = "git-checkout"
	workspaceBindingName  = "source"
	workspaceSourcePath   = "$(workspaces.source.path)"
	hookIDAnnotation      = "tekton.dev/ci-hook-id"
	ciSourceURLAnnotation = "tekton.dev/ci-source-url"
	ciSourceRefAnnotation = "tekton.dev/ci-source-ref"
	tektonGitInit         = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init"
)

// Source wraps a git clone URL and a specific ref to checkout.
type Source struct {
	RepoURL string
	Ref     string
}

func AnnotateSource(evtID string, src *Source) func(*pipelinev1.PipelineRun) {
	return func(pr *pipelinev1.PipelineRun) {
		pr.ObjectMeta.Annotations[ciSourceURLAnnotation] = src.RepoURL
		pr.ObjectMeta.Annotations[ciSourceRefAnnotation] = src.Ref
		pr.ObjectMeta.Annotations[hookIDAnnotation] = evtID
	}
}

// Convert takes a Pipeline definition, a name, source and volume claim name,
// and generates a TektonCD PipelineRun with an embedded Pipeline with the
// tasks to execute.
func Convert(p *ci.Pipeline, log logger.Logger, config *Configuration, src *Source, volumeClaimName string, ctx *cel.Context, id string) (*pipelinev1.PipelineRun, error) {
	env := makeEnv(p.Variables)
	tasks := []pipelinev1.PipelineTask{
		makeGitCloneTask(env, src),
	}
	logMeta := []interface{}{"volumeClaimName", volumeClaimName, "ref", src.Ref, "repoURL", src.RepoURL}
	log.Infow("converting pipeline", logMeta...)
	previous := []string{gitCloneTaskName}
	if len(p.BeforeScript) > 0 {
		log.Infow("processing before_script", "scriptLen", len(p.BeforeScript))
		tasks = append(tasks, makeScriptTask(beforeStepTaskName, previous, env, p.Image, p.BeforeScript))
		previous = []string{beforeStepTaskName}
	}
	for _, stageName := range p.Stages {
		log.Infow("processing stage", append(logMeta, "stage", stageName)...)
		stageTasks := []string{}
		for _, taskName := range p.TasksForStage(stageName) {
			task := p.Task(taskName)
			log.Infow("processing task", append(logMeta, "task", taskName)...)
			taskMatrix := makeTaskEnvMatrix(env, task)
			for i, m := range taskMatrix {
				image := p.Image
				if task.Tekton != nil && task.Tekton.Image != "" {
					image = task.Tekton.Image
				}
				stageTask, err := makeTaskForStage(task, stageName, previous, m, image, ctx)
				if err != nil {
					return nil, err
				}
				if stageTask != nil {
					if len(taskMatrix) > 1 {
						stageTask.Name = fmt.Sprintf("%s-%d", stageTask.Name, i)
					}
					tasks = append(tasks, *stageTask)
					if len(task.Artifacts.Paths) > 0 {
						archiverTask := makeArchiveArtifactsTask(previous, task.Name+"-archiver", env, config, task.Artifacts.Paths)
						tasks = append(tasks, archiverTask)
						stageTask = &archiverTask
					}
					stageTasks = append(stageTasks, stageTask.Name)
				}
			}
		}
		previous = stageTasks
	}
	if len(p.AfterScript) > 0 {
		tasks = append(tasks, makeScriptTask(afterStepTaskName, previous, env, p.Image, p.AfterScript))
	}
	spec := pipelinev1.PipelineRunSpec{
		ServiceAccountName: config.DefaultServiceAccountName,
		Workspaces: []pipelinev1.WorkspaceBinding{
			{
				Name: workspaceName,
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeClaimName,
				},
			},
		},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{
				{
					Name: workspaceName,
				},
			},
			Tasks: tasks,
		},
	}
	if p.TektonConfig != nil {
		spec.ServiceAccountName = p.TektonConfig.ServiceAccountName
	}
	return resources.PipelineRun("dsl", config.PipelineRunPrefix, spec, AnnotateSource(id, src)), nil
}

func makeTaskForStage(job *ci.Task, stage string, runAfter []string, env []corev1.EnvVar, image string, ctx *cel.Context) (*pipelinev1.PipelineTask, error) {
	ruleResults := make([]string, len(job.Rules))
	for i, r := range job.Rules {
		res, err := ctx.Evaluate(r.If)
		if err != nil {
			return nil, err
		}
		if res == types.True {
			ruleResults[i] = r.When
		}
	}
	if hasNever(ruleResults) {
		return nil, nil
	}
	pt := &pipelinev1.PipelineTask{
		Name:       job.Name + "-stage-" + stage,
		Workspaces: workspacePipelineTaskBindings(),
		RunAfter:   runAfter,
	}
	if job.Tekton != nil && job.Tekton.TaskRef != "" {
		pt.TaskRef = &pipelinev1.TaskRef{
			Name: job.Tekton.TaskRef,
			Kind: "Task",
		}
		params, err := paramsToParams(ctx, job.Tekton.Params)
		if err != nil {
			return nil, err
		}
		pt.Params = params
	} else {
		pt.TaskSpec = makeTaskSpec(makeScriptSteps(env, image, job.Script)...)
	}
	return pt, nil
}

func makeGitCloneTask(env []corev1.EnvVar, src *Source) pipelinev1.PipelineTask {
	return pipelinev1.PipelineTask{
		Name:       gitCloneTaskName,
		Workspaces: workspacePipelineTaskBindings(),
		TaskSpec: makeTaskSpec(
			pipelinev1.Step{
				Container: corev1.Container{
					Name:    "git-clone",
					Image:   tektonGitInit,
					Command: []string{"/ko-app/git-init", "-url", src.RepoURL, "-revision", src.Ref, "-path", workspaceSourcePath},
					Env:     append(env, corev1.EnvVar{Name: "TEKTON_RESOURCE_NAME", Value: "tekton-ci-git-clone"}),
				},
			},
		),
	}
}

func makeScriptTask(name string, runAfter []string, env []corev1.EnvVar, image string, script []string) pipelinev1.PipelineTask {
	return pipelinev1.PipelineTask{
		Name:       name,
		Workspaces: workspacePipelineTaskBindings(),
		RunAfter:   runAfter,
		TaskSpec:   makeTaskSpec(makeScriptSteps(env, image, script)...),
	}
}

func makeArchiveArtifactsTask(runAfter []string, name string, env []corev1.EnvVar, config *Configuration, artifacts []string) pipelinev1.PipelineTask {
	return pipelinev1.PipelineTask{
		Name:       name,
		Workspaces: workspacePipelineTaskBindings(),
		RunAfter:   runAfter,
		TaskSpec: makeTaskSpec(
			pipelinev1.Step{
				Container: container(name+"-archiver", config.ArchiverImage, "",
					append([]string{"archive", "--bucket-url", config.ArchiveURL}, artifacts...),
					env, workspaceSourcePath),
			},
		),
	}
}

func makeScriptSteps(env []corev1.EnvVar, image string, commands []string) []pipelinev1.Step {
	steps := make([]pipelinev1.Step, len(commands))
	for i, c := range commands {
		steps[i] = pipelinev1.Step{
			Container: container("", image, "sh", []string{"-c", c}, env, workspaceSourcePath),
		}
	}
	return steps
}

func workspacePipelineTaskBindings() []pipelinev1.WorkspacePipelineTaskBinding {
	return []pipelinev1.WorkspacePipelineTaskBinding{
		{
			Name:      workspaceBindingName,
			Workspace: workspaceName,
		},
	}
}

func makeTaskSpec(steps ...pipelinev1.Step) *pipelinev1.TaskSpec {
	return &pipelinev1.TaskSpec{
		Workspaces: []pipelinev1.WorkspaceDeclaration{
			{
				Name: workspaceBindingName,
			},
		},
		Steps: steps,
	}
}

func makeEnv(m map[string]string) []corev1.EnvVar {
	vars := []corev1.EnvVar{}
	for k, v := range m {
		vars = append(vars, corev1.EnvVar{Name: k, Value: v})
	}
	vars = append(vars, corev1.EnvVar{Name: "CI_PROJECT_DIR", Value: workspaceSourcePath})
	return vars
}

func container(name, image string, command string, args []string, env []corev1.EnvVar, workDir string) corev1.Container {
	c := corev1.Container{
		Name:       name,
		Image:      image,
		Args:       args,
		Env:        env,
		WorkingDir: workDir,
	}
	if command != "" {
		c.Command = []string{command}
	}
	return c
}

// Returns true if any of the rule results for a Task has a "never".
func hasNever(whens []string) bool {
	for _, v := range whens {
		if v == "never" {
			return true
		}
	}
	return false
}

// This converts the CI TaskParam model to a Tekton Pipeline Param.
//
// It evaluates the values as CEL expressions, and places the resulting value
// into the Param.
func paramsToParams(ctx *cel.Context, ciParams []ci.TektonTaskParam) ([]pipelinev1.Param, error) {
	params := []pipelinev1.Param{}
	for _, c := range ciParams {
		v, err := ctx.EvaluateToString(c.Expression)
		if err != nil {
			return nil, err
		}
		params = append(params, pipelinev1.Param{Name: c.Name, Value: pipelinev1.ArrayOrString{StringVal: v, Type: "string"}})
	}
	return params, nil
}

// This takes a slice of EnvVars, and returns a new slice of slices of EnvVars.
//
// Each slice in the new slice, will include the root EnvVars, plus a var from
// the task's TektonJobs.
//
// If Task has no Jobs, then the return is just a slice with the root EnvVars.
func makeTaskEnvMatrix(root []corev1.EnvVar, task *ci.Task) [][]corev1.EnvVar {
	if task.Tekton == nil || len(task.Tekton.Jobs) == 0 {
		return [][]corev1.EnvVar{root}
	}
	result := [][]corev1.EnvVar{}
	for _, job := range task.Tekton.Jobs {
		envVars := []corev1.EnvVar{}
		envVars = append(envVars, root...)
		for k, v := range job {
			envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
		}
		result = append(result, envVars)
	}
	return result
}
