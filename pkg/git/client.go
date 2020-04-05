package git

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
)

func New(c *scm.Client) *SCMClient {
	return &SCMClient{client: c}
}

type SCMClient struct {
	client *scm.Client
}

func (c *SCMClient) ParseWebhookRequest(req *http.Request) (scm.Webhook, error) {
	hook, err := c.client.Webhooks.Parse(req, c.Secret)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

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

func (c *SCMClient) Secret(webhook scm.Webhook) (string, error) {
	return "", nil
}

func isErrorStatus(i int) bool {
	return i > 400
}
