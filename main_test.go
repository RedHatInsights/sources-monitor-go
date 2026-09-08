package main

import (
	"os"
	"testing"
)

// The following statuses are not part of the main file, so they need to be declared here. They will be used to test
// the output of the "availabilityStatusMatches" function.
const (
	availableStatus          = "available"
	inProgressStatus         = "in_progress"
	partiallyAvailableStatus = "partially_available"
)

// TestNormalizeScheme verifies that normalizeScheme handles empty, uppercase,
// and whitespace-padded values correctly.
func TestNormalizeScheme(t *testing.T) {
	testData := []struct {
		input    string
		expected string
	}{
		{"", "http"},
		{"http", "http"},
		{"https", "https"},
		{"HTTP", "http"},
		{"HTTPS", "https"},
		{"  https  ", "https"},
		{"Http", "http"},
	}

	for _, td := range testData {
		got := normalizeScheme(td.input)
		if got != td.expected {
			t.Errorf("normalizeScheme(%q) = %q, want %q", td.input, got, td.expected)
		}
	}
}

// TestConfigureTLSTransport tests the TLS transport configuration function.
func TestConfigureTLSTransport(t *testing.T) {
	t.Run("empty CA path uses system certificate pool", func(t *testing.T) {
		transport, err := configureTLSTransport("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if transport == nil {
			t.Fatal("expected non-nil transport")
		}

		if transport.TLSClientConfig.RootCAs != nil {
			t.Error("expected nil RootCAs (system pool) when no CA path specified")
		}

		if !transport.ForceAttemptHTTP2 {
			t.Error("expected ForceAttemptHTTP2 to be true")
		}

		if transport.MaxConnsPerHost != 3 {
			t.Errorf("expected MaxConnsPerHost=3, got %d", transport.MaxConnsPerHost)
		}
	})

	t.Run("nonexistent CA path returns error", func(t *testing.T) {
		_, err := configureTLSTransport("/nonexistent/ca.crt")
		if err == nil {
			t.Error("expected error for nonexistent CA path")
		}
	})

	t.Run("invalid PEM content returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		f, err := os.CreateTemp(tmpDir, "test-ca-*.pem")
		if err != nil {
			t.Fatal(err)
		}

		_, writeErr := f.WriteString("not-a-valid-pem")
		if writeErr != nil {
			t.Fatal(writeErr)
		}

		f.Close()

		_, err = configureTLSTransport(f.Name())
		if err == nil {
			t.Error("expected error for invalid PEM content")
		}
	})
}

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
