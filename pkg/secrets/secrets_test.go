package secrets

import (
	"testing"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/bigkevmcd/tekton-ci/test/hook"
	"github.com/bigkevmcd/tekton-ci/test/secret"
)

var _ SecretGetter = (*KubeSecretGetter)(nil)

func TestSecretForKnownRepository(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(secret.Create("Codertocat/Hello-World"))
	hook := hook.MakeHookFromFixture(t, "../testdata/github_push.json", "push")
	g := New("testing", "tekton-ci-auth", fakeClient)

	secret, err := g.Secret(hook)
	if err != nil {
		t.Fatal(err)
	}

	if secret != "secret-token" {
		t.Fatalf("got %s, want secret-token", secret)
	}
}

func TestSecretWithMissingSecret(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	hook := hook.MakeHookFromFixture(t, "../testdata/github_push.json", "push")
	g := New("testing", "tekton-ci-auth", fakeClient)

	_, err := g.Secret(hook)
	if err.Error() != `secrets "tekton-ci-auth" not found` {
		t.Fatal(err)
	}
}

func TestSecretForUnknownRepository(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(secret.Create("my-org/hello-world"))
	hook := hook.MakeHookFromFixture(t, "../testdata/github_push.json", "push")
	g := New("testing", "tekton-ci-auth", fakeClient)

	_, err := g.Secret(hook)
	if err.Error() != "no secret for repository: Codertocat/Hello-World" {
		t.Fatal(err)
	}
}
