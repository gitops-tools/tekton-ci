package pipelines

import (
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bigkevmcd/tekton-ci/pkg/ci"
)

const (
	gitCloneTaskName     = "git-clone"
	beforeStepTaskName   = "before-step"
	afterStepTaskName    = "after-step"
	workspaceName        = "git-checkout"
	persistentClaimName  = "shared-task-storage"
	workspaceBindingName = "source"
	workspaceSourcePath  = "$(workspaces.source.path)"
	tektonGitInit        = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init"
)

type Source struct {
	RepoURL string
	Ref     string
}

// Convert takes a Pipelined definition, a name and source, and generates a
// TektonCD PipelineRun with an embedded Pipeline with the tasks to execute.
//
// TODO: allow passing in of the persistentClaimName.
func Convert(p *ci.Pipeline, pipelineRunName string, src *Source) *pipelinev1.PipelineRun {
	env := makeEnv(p.Variables)
	tasks := []pipelinev1.PipelineTask{
		makeGitCloneTask(env, src),
	}
	previous := gitCloneTaskName
	if len(p.BeforeScript) > 0 {
		tasks = append(tasks, makeScriptTask(gitCloneTaskName, beforeStepTaskName, env, p.Image, p.BeforeScript))
		previous = beforeStepTaskName
	}
	for _, name := range p.Stages {
		for _, jobName := range p.TasksForStage(name) {
			job := p.Task(jobName)
			stageTask := makeTaskForStage(job.Name, name, previous, env, p.Image, job.Script)
			tasks = append(tasks, stageTask)
			previous = stageTask.Name
		}
	}

	if len(p.AfterScript) > 0 {
		tasks = append(tasks, makeScriptTask(previous, afterStepTaskName, env, p.Image, p.AfterScript))
		previous = beforeStepTaskName
	}

	return &pipelinev1.PipelineRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: "pipeline.tekton.dev/v1beta1", Kind: "PipelineRun"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: pipelineRunName},
		Spec: pipelinev1.PipelineRunSpec{
			Workspaces: []pipelinev1.WorkspaceBinding{
				pipelinev1.WorkspaceBinding{
					Name: workspaceName,
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: persistentClaimName,
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
		},
	}
}

func makeTaskForStage(job, stage, previous string, env []corev1.EnvVar, image string, script []string) pipelinev1.PipelineTask {
	return pipelinev1.PipelineTask{
		Name:       job + "-stage-" + stage,
		Workspaces: workspacePipelineTaskBindings(),
		RunAfter:   []string{previous},
		TaskSpec:   makeTaskSpec(makeScriptSteps(env, image, script)...),
	}
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

func makeScriptTask(runAfter, name string, env []corev1.EnvVar, image string, script []string) pipelinev1.PipelineTask {
	return pipelinev1.PipelineTask{
		Name:       name,
		Workspaces: workspacePipelineTaskBindings(),
		RunAfter:   []string{runAfter},
		TaskSpec:   makeTaskSpec(makeScriptSteps(env, image, script)...),
	}
}

func makeScriptSteps(env []corev1.EnvVar, image string, commands []string) []pipelinev1.Step {
	steps := make([]pipelinev1.Step, len(commands))
	for i, c := range commands {
		steps[i] = pipelinev1.Step{
			Container: corev1.Container{
				Image:      image,
				Command:    []string{"sh", "-c", c},
				Env:        env,
				WorkingDir: workspaceSourcePath,
			},
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
