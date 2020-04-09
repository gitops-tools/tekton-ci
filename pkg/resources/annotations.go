package resources

// Annotations returns a standard set of annotations.
func Annotations(component string) map[string]string {
	return map[string]string{
		"tekton.dev/git-status":        "true",
		"tekton.dev/status-context":    "tekton-ci",
		"app.kubernetes.io/managed-by": component,
		"app.kubernetes.io/part-of":    "Tekton-CI",
	}
}
