package secrets

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/bigkevmcd/tekton-ci/test/hook"
)

var _ SecretGetter = (*KubeSecretGetter)(nil)

func TestSecretForKnownRepository(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(createSecret("Codertocat/Hello-World"))
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
	fakeClient := fake.NewSimpleClientset(createSecret("my-org/hello-world"))
	hook := hook.MakeHookFromFixture(t, "../testdata/github_push.json", "push")
	g := New("testing", "tekton-ci-auth", fakeClient)

	_, err := g.Secret(hook)
	if err.Error() != "no secret for repository: Codertocat/Hello-World" {
		t.Fatal(err)
	}
}

func createSecret(f string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tekton-ci-auth",
			Namespace: "testing",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			f: []byte(`secret-token`),
		},
	}
}
