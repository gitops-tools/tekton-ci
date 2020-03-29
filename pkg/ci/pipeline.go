package ci

// Pipeline represents the parsed Pipeline.
type Pipeline struct {
	Image        string
	Variables    map[string]string
	BeforeScript []string
	Stages       []string
	Jobs         []*Job
}

// Job represents the parsed Job from the Pipeline.
type Job struct {
	Name   string
	Stage  string
	Script []string
}

// JobsForStage returns the named jobs for a specific stage.
func (c Pipeline) JobsForStage(n string) []string {
	s := []string{}
	for _, j := range c.Jobs {
		if j.Stage == n {
			s = append(s, j.Name)
		}
	}
	return s
}

// Job returns the named job or nil if it exists
func (c Pipeline) Job(n string) *Job {
	for _, j := range c.Jobs {
		if n == j.Name {
			return j
		}
	}
	return nil
}
