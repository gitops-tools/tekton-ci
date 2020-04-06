package git

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
)

// New creates and returns a new SCMClient.
func New(c *scm.Client) *SCMClient {
	return &SCMClient{client: c}
}

// SCMClient is a wrapper for the go-scm scm.Client with a simplified API.
type SCMClient struct {
	client *scm.Client
}

// ParseWebhookRequest parses an incoming hook request and returns a parsed
// hook response if one can be matched.
func (c *SCMClient) ParseWebhookRequest(req *http.Request) (scm.Webhook, error) {
	hook, err := c.client.Webhooks.Parse(req, c.Secret)
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
	if isErrorStatus(r.Status) {
		return nil, scmError{msg: fmt.Sprintf("failed to get file %s from repo %s ref %s", path, repo, ref), Status: r.Status}
	}

	if err != nil {
		return nil, err
	}
	return content.Data, nil
}

// Secret returns a string that can be compared to an incoming hook secret.
func (c *SCMClient) Secret(webhook scm.Webhook) (string, error) {
	return "", nil
}

func isErrorStatus(i int) bool {
	return i > 400
}
