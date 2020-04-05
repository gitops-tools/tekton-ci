package git

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
)

func TestFileContents(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

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
	client := New(scmClient)

	_, err = client.FileContents(context.TODO(), "Codertocat/Hello-World", ".tekton_ci.yaml", "master")
	if !IsNotFound(err) {
		t.Fatal(err)
	}
}

func TestParsewebhook(t *testing.T) {
	as := makeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "master", "")
	defer as.Close()
	req := makeHookRequest(t, "testdata/push_hook.json", "push")
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)
	hook, err := client.ParseWebhookRequest(req)
	_ = hook.(*scm.PushHook)
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
		w.Write(b)
	}))
}

func readFixture(t *testing.T, filename string) map[string]interface{} {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read %s: %s", filename, err)
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		t.Fatalf("failed to unmarshal %s: %s", filename, err)
	}
	return result
}

// TODO use uuid to generate the Delivery ID.
func makeHookRequest(t *testing.T, fixture, eventType string) *http.Request {
	req := httptest.NewRequest("POST", "/", serialiseToJSON(t, readFixture(t, fixture)))
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	return req
}

func makeClient(t *testing.T, as *httptest.Server) *SCMClient {
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	return New(scmClient)
}

func serialiseToJSON(t *testing.T, e interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal %#v to JSON: %s", e, err)
	}
	return bytes.NewBuffer(b)
}
