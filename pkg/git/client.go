package git

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gitops-tools/tekton-ci/pkg/metrics"
	"github.com/gitops-tools/tekton-ci/pkg/secrets"
	"github.com/jenkins-x/go-scm/scm"
)

// New creates and returns a new SCMClient.
func New(c *scm.Client, s secrets.SecretGetter, m metrics.Interface) *SCMClient {
	return &SCMClient{client: c, secrets: s, m: m}
}

// SCMClient is a wrapper for the go-scm scm.Client with a simplified API.
type SCMClient struct {
	client  *scm.Client
	secrets secrets.SecretGetter
	m       metrics.Interface
}

// ParseWebhookRequest parses an incoming hook request and returns a parsed
// hook response if one can be matched.
func (c *SCMClient) ParseWebhookRequest(req *http.Request) (scm.Webhook, error) {
	hook, err := c.client.Webhooks.Parse(req, func(hook scm.Webhook) (string, error) {
		return c.secrets.Secret(req.Context(), hook)
	})
	if err != nil {
		return nil, err
	}
	return hook, nil
}

// FileContents reads the specific revision of a file from a repository.
//
// If an HTTP error is returned by the upstream service, an error with the
// response status code is returned.
func (c *SCMClient) FileContents(ctx context.Context, repo, path, ref string) ([]byte, error) {
	content, r, err := c.client.Contents.Find(ctx, repo, path, ref)
	c.m.CountAPICall("file_contents")
	if isErrorStatus(r.Status) {
		return nil, scmError{msg: fmt.Sprintf("failed to get file %s from repo %s ref %s", path, repo, ref), Status: r.Status}
	}
	if err != nil {
		return nil, err
	}
	return content.Data, nil
}

// CreateStatus creates a commit status.
//
// If an HTTP error is returned by the upstream service, an error with the
// response status code is returned.
func (c *SCMClient) CreateStatus(ctx context.Context, repo, commit string, s *scm.StatusInput) error {
	c.m.CountAPICall("create_status")
	_, r, err := c.client.Repositories.CreateStatus(ctx, repo, commit, s)
	errResponse := isErrorStatus(r.Status)
	if errResponse || err != nil {
		c.m.CountFailedAPICall("file_contents")
	}
	if errResponse {
		return scmError{msg: fmt.Sprintf("failed to create commitstatus in repo %s commit %s", repo, commit), Status: r.Status}
	}
	return err
}

func isErrorStatus(i int) bool {
	return i >= 400
}
