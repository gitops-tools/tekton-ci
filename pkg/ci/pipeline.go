package ci

// Pipeline represents the parsed Pipeline.
type Pipeline struct {
	Image        string
	Variables    map[string]string
	BeforeScript []string
	AfterScript  []string
	Stages       []string
	Tasks        []*Task
}

// Task represents the parsed Task from the Pipeline.
type Task struct {
	Name      string
	Stage     string
	Tekton    *TektonTask
	Script    []string
	Artifacts Artifacts
	Rules     []Rule
}

// Artifacts represents a set of paths that should be treated as artifacts and
// archived in some way.
type Artifacts struct {
	Paths []string
}

// Rule represents a rule that determines when a PipelineRun is triggered.
type Rule struct {
	If   string
	When string
}

// TektonTask is an extension for executing Tekton Tasks.
type TektonTask struct {
	TaskRef string
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
