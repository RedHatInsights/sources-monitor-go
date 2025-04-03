package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// skipEmptySourcesHeader defines the header's name that will allow the monitor to skip empty sources when fetching
// them from the API.
const skipEmptySourcesHeader = "x-rh-sources-skip-empty-sources"

// unavailableStatus holds the value used for unavailable statuses.
const unavailableStatus = "unavailable"

var (
	// where is sources-api?
	host = fmt.Sprintf("%v://%v:%v", os.Getenv("SOURCES_SCHEME"), os.Getenv("SOURCES_HOST"), os.Getenv("SOURCES_PORT"))
	// how can we talk to it?
	psk = os.Getenv("SOURCES_PSK")

	// global waitgroup to keep track of how many goroutines are running
	wg = sync.WaitGroup{}
	// channel to limit the number of requests running at once
	choke = make(chan struct{}, 3)
	// default client has no timeout, so use our own.
	httpClient = http.Client{Timeout: 10 * time.Second}
)

func main() {
	// usage: ./sources-monitor-go -status unavailable
	// defaults to "all"
	status := flag.String("status", "all", "which availability_status to check")
	flag.Parse()

	// Check whether we need to skip empty sources when fetching them or not.
	skipEmptySources := strings.ToLower(os.Getenv("SKIP_EMPTY_SOURCES")) == "true"

	if psk == "" {
		log.Fatalf("Need PSK to run availability checks.")
	}
	log.Printf("[host: %s][status: %s][skip_empty_sources: %t] Checking sources", host, *status, skipEmptySources)

	// a count of how many requests we do
	count := 0
	// first page of sources
	sources := listInternalSources(100, 0, skipEmptySources)
	for {
		// loop through all sources - requesting availability status updates
		// for those that match the `-status` flag.
		for _, s := range sources.Data {
			if *status == "all" || availabilityStatusMatches(s.AvailabilityStatus, *status) {
				count++
				// Add one to the "in-flight" waitgroup so we know what to wait for
				wg.Add(1)
				// send an empty struct onto the choke channel - this limits us to the
				// size of the channel as far as requests running at once
				choke <- struct{}{}
				go checkAvailability(s.ID, s.Tenant, s.OrgId, skipEmptySources)
			} else {
				log.Printf("[id: %s][account_number: %s][org_id: %s][availability_status: %s][requested_status: %s] Skipped source", s.ID, s.Tenant, s.OrgId, s.AvailabilityStatus, *status)
			}
		}
		// if we hit the last page, break out of the loop.
		if sources.Meta.Limit+sources.Meta.Offset > sources.Meta.Count {
			log.Printf("Requested availability for %v sources, waiting for all routines to complete...", count)
			break
		}

		// next page!
		sources = listInternalSources(sources.Meta.Limit, sources.Meta.Offset+sources.Meta.Limit, skipEmptySources)
	}

	// wait for all requests to complete at the end so the program doesn't
	// terminate and kill all running go-routines
	wg.Wait()
}

// GET /internal/v2.0/sources?limit=xx&offset=xx
// hit the internal sources api, parse it into a struct and return.
func listInternalSources(limit, offset int64, skipEmptySources bool) *SourceResponse {
	log.Printf("[limit: %d][offset: %d][host: %v][skip_empty_sources: %t] Requesting sources from internal API", limit, offset, host, skipEmptySources)

	url, _ := url.Parse(fmt.Sprintf("%v/internal/v2.0/sources?limit=%v&offset=%v", host, limit, offset))
	req := &http.Request{Method: http.MethodGet, URL: url, Header: map[string][]string{
		"x-rh-sources-account-number": {"sources_monitor"},
		"x-rh-sources-psk":            {psk},
	}}

	// Signal the Sources API that we might just be interested in fetching sources that do require an availability
	// check.
	if skipEmptySources {
		req.Header.Add(skipEmptySourcesHeader, "true")
	}

	resp, err := httpClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		log.Fatalf("Failed to list internal sources: %s", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	sources := &SourceResponse{}
	err = json.Unmarshal(data, sources)
	if err != nil {
		log.Fatalf("Failed to unmarshal sources: %s", err)
	}

	return sources
}

// POST /sources/:id/check_availability
// checking availability for a tenant's source
func checkAvailability(id, tenant, orgId string, skipEmptySources bool) {
	log.Printf("[source_id: %s][account_id: %s][org_id: %s][skip_empty_sources: %t] Requesting availability status for source", id, tenant, orgId, skipEmptySources)

	url, _ := url.Parse(fmt.Sprintf("%v/api/sources/v3.1/sources/%v/check_availability", host, id))

	requestHeaders := make(map[string][]string)
	requestHeaders["x-rh-sources-psk"] = []string{psk}

	if tenant != "" {
		requestHeaders["x-rh-sources-account-number"] = []string{tenant}
	}

	if orgId != "" {
		requestHeaders["x-rh-sources-org-id"] = []string{orgId}
	}

	// Signal the Sources API that we do not want to run availability checks for empty sources.
	if skipEmptySources {
		requestHeaders[skipEmptySourcesHeader] = []string{"true"}
	}

	req := &http.Request{Method: http.MethodPost, URL: url, Header: requestHeaders}
	resp, err := httpClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode != http.StatusAccepted) {
		log.Printf("[source_id: %s][account_number: %s][org_id: %s][skip_empty_sources: %t] Failed to request availability source", id, tenant, orgId, skipEmptySources)
		if resp != nil {
			log.Printf("Request status code: %v", resp.StatusCode)
		}
	}
	defer resp.Body.Close()

	// consume one value from the choke so another waiting routine can use it.
	<-choke
	// remove one from the waitgroup, since this routine is terminating.
	wg.Done()
}

// availabilityStatusMatches returns true if both the source status and the target status match, which implies that the
// current source should be checked for an availability status. In the case of having an "unavailable" target status,
// the empty string is also considered as "unavailable", and the "in_progress" one too.
func availabilityStatusMatches(sourceStatus string, targetStatus string) bool {
	if targetStatus == unavailableStatus {
		return sourceStatus == targetStatus || sourceStatus == "" || sourceStatus == "in_progress"
	}

	return sourceStatus == targetStatus
}
