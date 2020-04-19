package hook

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
	"github.com/bigkevmcd/tekton-ci/test"
)

// MakeHookFromFixture creates and returns a WebHook parsed from the provided
// fixture file, with the correct X-GitHub-Event type etc.
func MakeHookFromFixture(t *testing.T, filename, eventType string) scm.Webhook {
	t.Helper()
	req := test.MakeHookRequest(t, filename, eventType)
	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := git.New(scmClient)
	hook, err := client.ParseWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	return hook
}
