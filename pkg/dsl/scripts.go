package dsl

import (
	"github.com/google/cel-go/common/types"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/cel"
	"github.com/bigkevmcd/tekton-ci/pkg/ci"
	"github.com/bigkevmcd/tekton-ci/pkg/resources"
)

const (
	gitCloneTaskName     = "git-clone"
	beforeStepTaskName   = "before-step"
	afterStepTaskName    = "after-step"
	workspaceName        = "git-checkout"
	workspaceBindingName = "source"
	workspaceSourcePath  = "$(workspaces.source.path)"
	tektonGitInit        = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init"
)

type Source struct {
	RepoURL string
	Ref     string
}

// Convert takes a Pipeline definition, a name, source and volume claim name,
// and generates a TektonCD PipelineRun with an embedded Pipeline with the
// tasks to execute.
func Convert(p *ci.Pipeline, config *Configuration, src *Source, volumeClaimName string, ctx *cel.Context) (*pipelinev1.PipelineRun, error) {
	env := makeEnv(p.Variables)
	tasks := []pipelinev1.PipelineTask{
		makeGitCloneTask(env, src),
	}
	previous := []string{gitCloneTaskName}
	if len(p.BeforeScript) > 0 {
		tasks = append(tasks, makeScriptTask(beforeStepTaskName, previous, env, p.Image, p.BeforeScript))
		previous = []string{beforeStepTaskName}
	}
	for _, name := range p.Stages {
		stageTasks := []string{}
		for _, taskName := range p.TasksForStage(name) {
			task := p.Task(taskName)
			stageTask, err := makeTaskForStage(task, name, previous, env, p.Image, ctx)
			if err != nil {
				return nil, err
			}
			if stageTask != nil {
				tasks = append(tasks, *stageTask)
				if len(task.Artifacts.Paths) > 0 {
					archiverTask := makeArchiveArtifactsTask(previous, task.Name+"-archiver", env, config, task.Artifacts.Paths)
					tasks = append(tasks, archiverTask)
					stageTask = &archiverTask
				}
				stageTasks = append(stageTasks, stageTask.Name)
			}
		}
		previous = stageTasks
	}
	if len(p.AfterScript) > 0 {
		tasks = append(tasks, makeScriptTask(afterStepTaskName, previous, env, p.Image, p.AfterScript))
	}
	spec := pipelinev1.PipelineRunSpec{
		Workspaces: []pipelinev1.WorkspaceBinding{
			pipelinev1.WorkspaceBinding{
				Name: workspaceName,
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeClaimName,
				},
			},
		},
		PipelineSpec: &pipelinev1.PipelineSpec{
			Workspaces: []pipelinev1.WorkspacePipelineDeclaration{
				pipelinev1.WorkspacePipelineDeclaration{
					Name: workspaceName,
				},
			},
			Tasks: tasks,
		},
	}
	return resources.PipelineRun("dsl", config.PipelineRunPrefix, spec), nil
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
	if job.Tekton != nil {
		pt.TaskRef = &pipelinev1.TaskRef{
			Name: job.Tekton.TaskRef,
			Kind: "Task",
		}
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
					Env:     env,
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
		pipelinev1.WorkspacePipelineTaskBinding{
			Name:      workspaceBindingName,
			Workspace: workspaceName,
		},
	}
}

func makeTaskSpec(steps ...pipelinev1.Step) *pipelinev1.TaskSpec {
	return &pipelinev1.TaskSpec{
		Workspaces: []pipelinev1.WorkspaceDeclaration{
			pipelinev1.WorkspaceDeclaration{
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

func hasNever(whens []string) bool {
	for _, v := range whens {
		if v == "never" {
			return true
		}
	}
	return false
}
