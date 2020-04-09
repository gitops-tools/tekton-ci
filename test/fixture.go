package test

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

// ReadJSONFixture reads a filename into a map, and fails the test if it is
// unable to open or parse the file.
func ReadJSONFixture(t *testing.T, filename string) map[string]interface{} {
	t.Helper()
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
