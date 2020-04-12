package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fixtureFunc func(map[string]interface{})

// TODO use uuid to generate the Delivery ID.
// MakeHookRequest creates and returns a new http.Request that resembles a
// GitHub hook request, including the correct event type and reading and sending
// a fixture as a JSON body.
//
// Optionally changes can be applied to the fixture that is read, before it's
// sent.
func MakeHookRequest(t *testing.T, fixture, eventType string, changes ...fixtureFunc) *http.Request {
	body := ReadJSONFixture(t, fixture)
	for _, c := range changes {
		c(body)
	}
	req := httptest.NewRequest("POST", "/", serialiseToJSON(t, body))
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	return req
}

func serialiseToJSON(t *testing.T, e interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal %#v to JSON: %s", e, err)
	}
	return bytes.NewBuffer(b)
}
