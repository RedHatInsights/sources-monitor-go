package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The following statuses are not part of the main file, so they need to be declared here. They will be used to test
// the output of the "availabilityStatusMatches" function.
const (
	availableStatus          = "available"
	inProgressStatus         = "in_progress"
	partiallyAvailableStatus = "partially_available"
)

// TestAvailabilityStatusMatches tests if the function under test returns "true" only when the source status matches
// the target status. It also tests that a "true" is returned when the target status is "unavailable" and the source's
// status is empty or the source's status is "in_progress".
func TestAvailabilityStatusMatches(t *testing.T) {
	testData := []struct {
		SourceStatus        string
		TargetStatus        string
		ExpectedReturnValue bool
	}{
		{availableStatus, availableStatus, true},
		{inProgressStatus, availableStatus, false},
		{partiallyAvailableStatus, availableStatus, false},
		{unavailableStatus, availableStatus, false},
		{availableStatus, unavailableStatus, false},
		{inProgressStatus, unavailableStatus, true},
		{partiallyAvailableStatus, unavailableStatus, false},
		{unavailableStatus, unavailableStatus, true},
		{"", unavailableStatus, true},
	}

	for _, td := range testData {
		want := td.ExpectedReturnValue
		got := availabilityStatusMatches(td.SourceStatus, td.TargetStatus)

		if want != got {
			t.Errorf(`unexpected result returned from the function. Want "%t", got "%t". %#v`, want, got, td)
		}
	}
}

// TestResolveInternalBasepath_NewPathAvailable tests that the new standard basepath is returned when the server
// responds with 200 on the new path.
func TestResolveInternalBasepath_NewPathAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == newInternalBasepath+"/sources" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[],"meta":{"count":0,"limit":1,"offset":0}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	result := resolveInternalBasepath(server.URL, "test-psk", client)

	if result != newInternalBasepath {
		t.Errorf("expected %s, got %s", newInternalBasepath, result)
	}
}

// TestResolveInternalBasepath_NewPathNotFound tests that the legacy basepath is returned when the server responds
// with 404 on the new path.
func TestResolveInternalBasepath_NewPathNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	result := resolveInternalBasepath(server.URL, "test-psk", client)

	if result != legacyInternalBasepath {
		t.Errorf("expected %s, got %s", legacyInternalBasepath, result)
	}
}

// TestResolveInternalBasepath_ConnectionError tests that the legacy basepath is returned when the probe request
// fails due to a connection error.
func TestResolveInternalBasepath_ConnectionError(t *testing.T) {
	client := &http.Client{}
	// Use an invalid host to trigger a connection error.
	result := resolveInternalBasepath("http://127.0.0.1:0", "test-psk", client)

	if result != legacyInternalBasepath {
		t.Errorf("expected %s, got %s", legacyInternalBasepath, result)
	}
}

// TestResolveInternalBasepath_ServerError tests that the legacy basepath is returned when the server responds
// with a 500 error on the new path.
func TestResolveInternalBasepath_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := server.Client()
	result := resolveInternalBasepath(server.URL, "test-psk", client)

	if result != legacyInternalBasepath {
		t.Errorf("expected %s, got %s", legacyInternalBasepath, result)
	}
}

// TestResolveInternalBasepath_SendsPSKHeader verifies that the probe request includes the PSK header.
func TestResolveInternalBasepath_SendsPSKHeader(t *testing.T) {
	var receivedPSK string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPSK = r.Header.Get("x-rh-sources-psk")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[],"meta":{"count":0,"limit":1,"offset":0}}`))
	}))
	defer server.Close()

	client := server.Client()
	resolveInternalBasepath(server.URL, "my-secret-psk", client)

	if receivedPSK != "my-secret-psk" {
		t.Errorf("expected PSK header 'my-secret-psk', got '%s'", receivedPSK)
	}
}
