package dsl

// Configuration provides options for the conversion to PipelineRuns.
type Configuration struct {
	ArchiverImage     string // Executed for tasks that have artifacts to archive.
	ArchiveURL        string // Passed to the archiver along with the artifact paths.
	PipelineRunPrefix string // Used in the generateName property of the created PipelineRun.
}
