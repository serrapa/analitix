package dnslytics

import "testing"

func TestParseResponse(t *testing.T) {
	body := []byte(`{
		"status":"success",
		"data":{
			"domains":[
				{"domain":"www.example.com","firstseen":"2024-01-01","lastseen":"2024-03-01"},
				"blog.example.com"
			]
		}
	}`)

	records, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("unexpected record count: %d", len(records))
	}
	if records[0].Domain == "" || records[1].Domain == "" {
		t.Fatalf("unexpected empty domain in records: %+v", records)
	}
}
