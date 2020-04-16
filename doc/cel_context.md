# CEL Context Keys

Wherever CEL expressions are evaluated, there are contextual values available to
use in the expressions.

These are documented below:

## Hook

The `hook` key contains the incoming Webhook parsed by [go-scm](https://github.com/jenkins-x/go-scm).

For example, a [GitHub `push` event](https://developer.github.com/v3/activity/events/types/#pushevent) is parsed to a [`PushHook`](https://github.com/jenkins-x/go-scm/blob/master/scm/webhook.go#L78).

This means that this expression `hook.Before` will contain the `before` key from
the push event.

it's possible to nest the keys, so `hook.Repo.Clone` will be evaluated to the "clone URL" for the repository.

## Vars

Some hook fields are extracted and placed into a `vars` key in the expression
context.

 * CI_COMMIT_SHA
 * CI_COMMIT_SHORT_SHA
 * CI_COMMIT_BRANCH

This means that `vars.CI_COMMIT_SHA` will be the commit SHA of the commit referred to in the received event.

These are standardised across the supported `go-scm` event types, for example,
`CI_COMMIT_SHA` will reference the `PullRequest.Sha` field for `PullRequestHook`
events, but it'll be the `Commit.Sha` field for `PushHook` events.
