package dsl

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/gitops-tools/tekton-ci/pkg/git"
	"github.com/gitops-tools/tekton-ci/pkg/metrics"
	"github.com/gitops-tools/tekton-ci/pkg/secrets"
	"github.com/gitops-tools/tekton-ci/pkg/volumes"
	"github.com/gitops-tools/tekton-ci/test"
)

const testNS = "testing"

func TestHandlePushEvent(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "6113728f27ae82c7b1a177c8d03f9e96e0adf246", "testdata/content.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeTektonClient := fakeclientset.NewSimpleClientset()
	fakeClient := fake.NewSimpleClientset()
	vc := volumes.New(fakeClient)
	cfg := testConfiguration()
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	converter := NewDSLConverter(gitClient, fakeTektonClient, vc, metrics.NewMock(), cfg, testNS, logger.Sugar())
	h := New(gitClient, logger.Sugar(), metrics.NewMock(), converter)
	req := test.MakeHookRequest(t, "../testdata/github_push.json", "push")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusNotFound, mustReadBody(t, w))
	}
	claim, err := fakeClient.CoreV1().PersistentVolumeClaims(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO: This should probably be a call to a function in volumes.
	wantClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-volume-",
			Namespace:    testNS,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": cfg.VolumeSize,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			VolumeMode: &volumes.SimpleVolumeMode,
		},
	}

	if diff := cmp.Diff(wantClaim, claim, cmpopts.IgnoreFields(corev1.PersistentVolumeClaim{}, "TypeMeta")); diff != "" {
		t.Fatalf("persistent volume claim incorrect, diff\n%s", diff)
	}
	pr, err := fakeTektonClient.TektonV1().PipelineRuns(testNS).Get(
		context.TODO(), "", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if l := len(pr.Spec.PipelineSpec.Tasks); l != 4 {
		t.Fatalf("got %d tasks, want 4", l)
	}
	// check that it picked up the correct source URL and branch from the
	// fixture file.
	want := []string{
		"/ko-app/git-init",
		"-url", "https://github.com/Codertocat/Hello-World.git",
		"-revision", "6113728f27ae82c7b1a177c8d03f9e96e0adf246",
		"-path", "$(workspaces.source.path)",
	}
	if diff := cmp.Diff(want, pr.Spec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Command); diff != "" {
		t.Fatalf("git command incorrect, diff\n%s", diff)
	}
	prUUID := pr.ObjectMeta.Annotations[ciHookIDAnnotation]
	if deliveryID := req.Header.Get("X-GitHub-Delivery"); prUUID != deliveryID {
		t.Fatalf("PR UUID got %s, want %s", prUUID, deliveryID)
	}
}

func TestHandlePushEventNoPipeline(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "6113728f27ae82c7b1a177c8d03f9e96e0adf246", "")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeTektonClient := fakeclientset.NewSimpleClientset()
	fakeClient := fake.NewSimpleClientset()
	vc := volumes.New(fakeClient)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	converter := NewDSLConverter(gitClient, fakeTektonClient, vc, metrics.NewMock(), testConfiguration(), testNS, logger.Sugar())
	h := New(gitClient, logger.Sugar(), metrics.NewMock(), converter)

	req := test.MakeHookRequest(t, "../testdata/github_push.json", "push")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusOK, mustReadBody(t, w))
	}
	_, err = fakeTektonClient.TektonV1().PipelineRuns(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatal("pipelinerun was created when no pipeline definition exists")
	}
}

func TestHandlePushEventNoMatchingRules(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "6113728f27ae82c7b1a177c8d03f9e96e0adf246", "testdata/content_match_only_master.json")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeTektonClient := fakeclientset.NewSimpleClientset()
	fakeClient := fake.NewSimpleClientset()
	vc := volumes.New(fakeClient)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	converter := NewDSLConverter(gitClient, fakeTektonClient, vc, metrics.NewMock(), testConfiguration(), testNS, logger.Sugar())
	h := New(gitClient, logger.Sugar(), metrics.NewMock(), converter)
	req := test.MakeHookRequest(t, "../testdata/github_push.json", "push")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusOK, mustReadBody(t, w))
	}
	_, err = fakeTektonClient.TektonV1().PipelineRuns(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatal("pipelinerun was created with no matching rules")
	}
}

func TestHandlePushEventWithSkippableMessage(t *testing.T) {
	as := test.MakeAPIServer(t, "/api/v3/repos/Codertocat/Hello-World/contents/.tekton_ci.yaml", "6113728f27ae82c7b1a177c8d03f9e96e0adf246", "")
	defer as.Close()
	scmClient, err := factory.NewClient("github", as.URL, "", factory.Client(as.Client()))
	if err != nil {
		t.Fatal(err)
	}
	gitClient := git.New(scmClient, secrets.NewMock(), metrics.NewMock())
	fakeTektonClient := fakeclientset.NewSimpleClientset()
	fakeClient := fake.NewSimpleClientset()
	vc := volumes.New(fakeClient)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	converter := NewDSLConverter(gitClient, fakeTektonClient, vc, metrics.NewMock(), testConfiguration(), testNS, logger.Sugar())
	h := New(gitClient, logger.Sugar(), metrics.NewMock(), converter)
	req := test.MakeHookRequest(t, "../testdata/github_push.json", "push", func(b map[string]interface{}) {
		b["head_commit"].(map[string]interface{})["message"] = "This is a [skip ci] commit"
	})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	w := rec.Result()
	if w.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d: %s", w.StatusCode, http.StatusOK, mustReadBody(t, w))
	}
	_, err = fakeTektonClient.TektonV1().PipelineRuns(testNS).Get(context.TODO(), "", metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatalf("pipelinerun was created when the message indicated a skip")
	}
}

func TestSkip(t *testing.T) {
	skipTests := []struct {
		message string
		skip    bool
	}{
		{"this is a message\n", false},
		{"this is [ci skip]a message\n", true},
		{"this is [skip ci]a message\n", true},
	}

	for i, tt := range skipTests {
		h := &scm.PushHook{
			Commit: scm.Commit{
				Message: tt.message,
			},
		}
		if b := skip(h); b != tt.skip {
			t.Errorf("%d failed, got %v, want %v", i, b, tt.skip)
		}
	}

}

func mustReadBody(t *testing.T, req *http.Response) []byte {
	t.Helper()
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
