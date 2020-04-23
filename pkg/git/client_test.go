package git

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/bigkevmcd/tekton-ci/pkg/secrets"
	"github.com/bigkevmcd/tekton-ci/test"
	"github.com/bigkevmcd/tekton-ci/test/secret"
)

func TestFileContents(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient, nil)

	body, err := client.FileContents(context.TODO(), "Codertocat/Hello-World", ".tekton_ci.yaml", "master")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("testing service\n")
	if diff := cmp.Diff(want, body); diff != "" {
		t.Fatalf("got a different body back: %s\n", diff)
	}
}

func TestFileContentsWithNotFoundResponse(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient, nil)

	_, err = client.FileContents(context.TODO(), "Codertocat/Hello-World", ".tekton_ci.yaml", "master")
	if !IsNotFound(err) {
		t.Fatal(err)
	}
}

func TestParseWebhook(t *testing.T) {
	hookSecret := secret.Create("Codertocat/Hello-World")
	fakeClient := fake.NewSimpleClientset(hookSecret)
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "")
	defer as.Close()
	req := test.MakeHookRequest(t, "testdata/push_hook.json", "push")
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient, secrets.New(hookSecret.ObjectMeta.Namespace, hookSecret.ObjectMeta.Name, fakeClient))
	hook, err := client.ParseWebhookRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = hook.(*scm.PushHook)
}

func TestParseWebhookWithInvalidSignature(t *testing.T) {
	hookSecret := secret.Create("Codertocat/Hello-World")
	fakeClient := fake.NewSimpleClientset(hookSecret)
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "")
	defer as.Close()
	req := test.MakeHookRequest(t, "testdata/push_hook.json", "push")
	req.Header.Set("X-Hub-Signature", "sha1=testing")
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient, secrets.New(hookSecret.ObjectMeta.Namespace, hookSecret.ObjectMeta.Name, fakeClient))
	_, err = client.ParseWebhookRequest(req)
	if err != scm.ErrSignatureInvalid {
		t.Fatal(err)
	}
}

func makeAPIServer(t *testing.T, urlPath, ref, fixture string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != urlPath {
			http.NotFound(w, r)
			t.Fatalf("request path got %s, want %s", r.URL.Path, urlPath)
		}
		if queryRef := r.URL.Query().Get("ref"); queryRef != ref {
			t.Fatalf("failed to match ref, got %s, want %s", queryRef, ref)
		}
		if fixture == "" {
			http.NotFound(w, r)
			return
		}
		b, err := ioutil.ReadFile(fixture)
		if err != nil {
			t.Fatalf("failed to read %s: %s", fixture, err)
		}
		_, err = w.Write(b)
		if err != nil {
			t.Fatalf("failed to write: %s", err)
		}
	}))
}
