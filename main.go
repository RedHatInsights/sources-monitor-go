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
	"sync"
	"time"
)

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

	if psk == "" {
		log.Fatalf("Need PSK to run availability checks.")
	}
	log.Printf("Checking sources with [%v] status from [%v]", *status, host)

	// a count of how many requests we do
	count := 0
	// first page of sources
	sources := listInternalSources(100, 0)
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
				go checkAvailability(s.ID, s.Tenant)
			}
		}
		// if we hit the last page, break out of the loop.
		if sources.Meta.Limit+sources.Meta.Offset > sources.Meta.Count {
			log.Printf("Requested availability for %v sources, waiting for all routines to complete...", count)
			break
		}

		// next page!
		sources = listInternalSources(sources.Meta.Limit, sources.Meta.Offset+sources.Meta.Limit)
	}

	// wait for all requests to complete at the end so the program doesn't
	// terminate and kill all running go-routines
	wg.Wait()
}

// GET /internal/v2.0/sources?limit=xx&offset=xx
// hit the internal sources api, parse it into a struct and return.
func listInternalSources(limit, offset int64) *SourceResponse {
	log.Printf("Requesting [limit %v] + [offset %v] sources from internal API at [%v]", limit, offset, host)

	url, _ := url.Parse(fmt.Sprintf("%v/internal/v2.0/sources?limit=%v&offset=%v", host, limit, offset))
	req := &http.Request{Method: http.MethodGet, URL: url, Header: map[string][]string{
		"x-rh-sources-account-number": {"sources_monitor"},
		"x-rh-sources-psk":            {psk},
	}}
	resp, err := httpClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		log.Fatalf("Failed to list internal sources: %v", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	sources := &SourceResponse{}
	err = json.Unmarshal(data, sources)
	if err != nil {
		log.Fatalf("Failed to unmarshal sources: %v", err)
	}

	return sources
}

// POST /sources/:id/check_availability
// checking availability for a tenant's source
func checkAvailability(id, tenant string) {
	log.Printf("Requesting availability status for [tenant %v], [source %v]", tenant, id)

	url, _ := url.Parse(fmt.Sprintf("%v/api/sources/v3.1/sources/%v/check_availability", host, id))
	req := &http.Request{Method: http.MethodPost, URL: url, Header: map[string][]string{
		"x-rh-sources-account-number": {tenant},
		"x-rh-sources-psk":            {psk},
	}}
	resp, err := httpClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode != http.StatusAccepted) {
		log.Printf("Failed to request availability for [tenant %v], [source %v]", tenant, id)
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
// the empty string is also considered as "unavailable".
func availabilityStatusMatches(sourceStatus string, targetStatus string) bool {
	if targetStatus == unavailableStatus {
		return sourceStatus == targetStatus || sourceStatus == ""
	}

	return sourceStatus == targetStatus
}
