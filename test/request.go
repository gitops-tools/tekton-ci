package test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fixtureFunc func(map[string]interface{})

const secret = "secret-token"

// MakeHookRequest creates and returns a new http.Request that resembles a
// GitHub hook request, including the correct event type and reading and sending
// a fixture as a JSON body.
//
// Optionally changes can be applied to the fixture that is read, before it's
// sent.
// TODO use uuid to generate the Delivery ID.
func MakeHookRequest(t *testing.T, fixture, eventType string, changes ...fixtureFunc) *http.Request {
	body := ReadJSONFixture(t, fixture)
	for _, c := range changes {
		c(body)
	}

	serialisedBody := serialiseToJSON(t, body)
	mac := hmac.New(sha1.New, []byte(secret))
	_, err := mac.Write(serialisedBody.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	sig := hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest("POST", "/", serialisedBody)
	req.Header.Add("X-GitHub-Delivery", "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Event", eventType)
	req.Header.Add("X-Hub-Signature", fmt.Sprintf("sha1=%s", sig))
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
