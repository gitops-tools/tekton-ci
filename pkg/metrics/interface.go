package metrics

import "github.com/jenkins-x/go-scm/scm"

// Interface implementations provide metrics for the system.
type Interface interface {
	// CountHook records this hook as having been received, along with it's kind.
	CountHook(h scm.Webhook)

	// CountInvalidHook records "bad" hooks, probably due to non-matching secrets.
	CountInvalidHook()

	// CountAPICall records API calls to the upstream hosting service.
	CountAPICall(name string)

	// CountFailedAPICall records failed API calls to the upstream hosting service.
	CountFailedAPICall(name string)
}
