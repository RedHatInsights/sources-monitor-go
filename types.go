package main

type SourceResponse struct {
	Data []struct {
		AvailabilityStatus string `json:"availability_status"`
		ID                 string `json:"id"`
		Tenant             string `json:"tenant"`
		OrgId              string `json:"org_id"`
	} `json:"data"`
	Meta struct {
		Count  int64 `json:"count"`
		Limit  int64 `json:"limit"`
		Offset int64 `json:"offset"`
	} `json:"meta"`
}
