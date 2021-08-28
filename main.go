package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var (
	host  string
	psk   string
	wg    = sync.WaitGroup{}
	choke chan struct{}
)

type SourceResponse struct {
	Data []struct {
		AvailabilityStatus string `json:"availability_status"`
		ID                 string `json:"id"`
		Tenant             string `json:"tenant"`
	} `json:"data"`
	Meta struct {
		Count  int64 `json:"count"`
		Limit  int64 `json:"limit"`
		Offset int64 `json:"offset"`
	} `json:"meta"`
}

func init() {
	// where is sources-api?
	host = fmt.Sprintf("%v://%v:%v", os.Getenv("SOURCES_SCHEME"), os.Getenv("SOURCES_HOST"), os.Getenv("SOURCES_PORT"))
	psk = os.Getenv("SOURCES_PSK")
	// only sending the buffered channel's size requests at once
	choke = make(chan struct{}, 3)
}

func main() {
	// usage: ./sources-monitor-go -status unavailable
	// defaults to "all"
	status := flag.String("status", "all", "which availability_status to check")
	flag.Parse()

	// first page of sources
	sources := listInternalSources(100, 0)
	for {
		// loop through all sources - requesting availability status updates
		// for those that match the `-status` flag.
		for _, s := range sources.Data {
			if *status == "all" || s.AvailabilityStatus == *status {
				// Add one to the "in-flight" waitgroup so we know what to wait for
				wg.Add(1)
				go checkAvailability(s.ID, s.Tenant)
			}
		}
		// if we hit the last page, break out of the loop.
		if sources.Meta.Limit+sources.Meta.Offset > sources.Meta.Count {
			break
		}

		// next page!
		sources = listInternalSources(sources.Meta.Limit, sources.Meta.Offset+sources.Meta.Limit)
	}

	// wait for all requests to complete at the end so the program doesn't
	// terminate and kill all running go-routines
	wg.Wait()
}

// hit the internal sources api, parse it into a struct and return.
func listInternalSources(limit, offset int64) *SourceResponse {
	url, _ := url.Parse(fmt.Sprintf("%v/internal/v2.0/sources?limit=%v&offset=%v", host, limit, offset))
	req := &http.Request{Method: "GET", URL: url, Header: map[string][]string{
		"x-rh-sources-account-number": {"sources_monitor"},
		"x-rh-sources-psk":            {psk},
	}}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Failed to list internal sources: %v", err)
		os.Exit(1)
	}

	data, _ := ioutil.ReadAll(resp.Body)
	sources := &SourceResponse{}
	err = json.Unmarshal(data, sources)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal sources: %v", err)
		os.Exit(1)
	}

	return sources
}

// POST /sources/:id/check_availability for a tenant's source
func checkAvailability(id, tenant string) {
	// send an empty struct onto the choke channel - this limits us to the
	// size of the channel as far as requests running at once
	choke <- struct{}{}

	fmt.Printf("Requesting availability status for tenant %v, source %v\n", tenant, id)

	url, _ := url.Parse(fmt.Sprintf("%v/api/sources/v3.1/sources/%v/check_availability", host, id))
	req := &http.Request{Method: "POST", URL: url, Header: map[string][]string{
		"x-rh-sources-account-number": {tenant},
		"x-rh-sources-psk":            {psk},
	}}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 202 {
		fmt.Fprintf(os.Stderr, "Failed to request availability for tenant %v source id: %v\n", tenant, id)
	}

	// consume one value from the choke so another waiting routine can use it.
	<-choke
	// remove one from the waitgroup, since we're done.
	wg.Done()
}
