package ci

// Pipeline represents the parsed Pipeline.
type Pipeline struct {
	Image        string            `json:"image,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	BeforeScript []string          `json:"before_script,omitempty"`
	AfterScript  []string          `json:"after_script,omitempty"`
	Stages       []string          `json:"stages,omitempty"`
	Tasks        []*Task           `json:"tasks,omitempty"`
	TektonConfig *TektonConfig     `json:"tekton,omitempty"`
}

// Task represents the parsed Task from the Pipeline.
type Task struct {
	Name      string      `json:"name"`
	Stage     string      `json:"stage,omitempty"`
	Tekton    *TektonTask `json:"tekton,omitempty"`
	Script    []string    `json:"script,omitempty"`
	Artifacts Artifacts   `json:"artifacts,omitempty"`
	Rules     []Rule      `json:"rules,omitempty"`
}

// Artifacts represents a set of paths that should be treated as artifacts and
// archived in some way.
type Artifacts struct {
	Paths []string `json:"paths,omitempty"`
}

// Rule represents a rule that determines when a PipelineRun is triggered.
type Rule struct {
	If   string `json:"if"`
	When string `json:"when"`
}

// TektonTask is an extension for executing Tekton Tasks.
type TektonTask struct {
	// Used to generate a matrix of configurations for parallel jobs.

	Jobs    []map[string]string `json:"jobs,omitempty"`
	TaskRef string              `json:"taskRef,omitempty"`
	Params  []TektonTaskParam   `json:"params,omitempty"`
}

// TektonTaskParam is passed into a Tekton task.
type TektonTaskParam struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// TektonConfig provides global configuration for the DSL script specifically
// for Tekton.
type TektonConfig struct {
	ServiceAccountName string `json:"serviceAccountName"`
}

// TasksForStage returns the named jobs for a specific stage.
func (c Pipeline) TasksForStage(n string) []string {
	s := []string{}
	for _, j := range c.Tasks {
		if j.Stage == n {
			s = append(s, j.Name)
		}
	}
	return s
}

// Task returns the named job or nil if it exists
func (c Pipeline) Task(n string) *Task {
	for _, j := range c.Tasks {
		if n == j.Name {
			return j
		}
	}
	return nil
}
