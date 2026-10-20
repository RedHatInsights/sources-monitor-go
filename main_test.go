package main

import "testing"

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
