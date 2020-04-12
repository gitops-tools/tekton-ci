package test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm/factory"

	"github.com/bigkevmcd/tekton-ci/pkg/git"
)

func MakeHookFromFixture(t *testing.T, filename, eventType string) interface{} {
	t.Helper()
	req := MakeHookRequest(t, filename, eventType)
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
