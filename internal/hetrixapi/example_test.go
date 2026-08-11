// Modified by the hetrixtools-sync project from the Apache-2.0 upstream source.
package hetrixtools_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	hetrixtools "github.com/filippo-claude/hetrixtools-sync/internal/hetrixapi"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func exampleClient(responses map[string]string) (*hetrixtools.Client, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		if response, ok := responses[key]; ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
			return
		}
		http.Error(w, key, http.StatusNotFound)
	}))
	client := hetrixtools.NewClientWithBaseURL(server.URL, "token", hetrixtools.WithMinimumRequestInterval(0))
	return client, server.Close
}

func ExampleNewClientWithBaseURL() {
	client, closeServer := exampleClient(map[string]string{
		"GET /v3/account/limits": `{"uptime":{"monitors":{"usage":1,"limit":2}}}`,
	})
	defer closeServer()

	limits, _ := client.GetAccountLimits(context.Background())
	fmt.Println(limits.Uptime.Monitors.Limit)
	// Output: 2
}

func ExampleWithHTTPClient() {
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"uptime":{"monitors":{"usage":3,"limit":7}}}`)),
		}, nil
	})}
	client := hetrixtools.NewClient("token", hetrixtools.WithHTTPClient(httpClient), hetrixtools.WithMinimumRequestInterval(0))

	limits, _ := client.GetAccountLimits(context.Background())
	fmt.Println(limits.Uptime.Monitors.Limit)
	// Output: 7
}

func ExampleWithV2BaseURL() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom-v2/token/blacklist-check/domain/example.com/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCESS","blacklisted_count":0}`))
	}))
	defer server.Close()

	client := hetrixtools.NewClientWithBaseURL("https://unused.invalid", "token",
		hetrixtools.WithV2BaseURL(server.URL+"/custom-v2"),
		hetrixtools.WithMinimumRequestInterval(0),
	)
	result, _ := client.CheckBlacklistDomain(context.Background(), "example.com")
	fmt.Println(result.BlacklistedCount)
	// Output: 0
}

func ExampleWithV3BaseURL() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom-v3/account/limits" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uptime":{"monitors":{"usage":2,"limit":5}}}`))
	}))
	defer server.Close()

	client := hetrixtools.NewClientWithBaseURL("https://unused.invalid", "token",
		hetrixtools.WithV3BaseURL(server.URL+"/custom-v3"),
		hetrixtools.WithMinimumRequestInterval(0),
	)
	limits, _ := client.GetAccountLimits(context.Background())
	fmt.Println(limits.Uptime.Monitors.Limit)
	// Output: 5
}

func ExampleClient_CreateBlacklistMonitor() {
	client, closeServer := exampleClient(map[string]string{
		"POST /v2/token/blacklist/add/": `{"status":"SUCCESS","monitor_id":"blacklist-1"}`,
	})
	defer closeServer()

	response, _ := client.CreateBlacklistMonitor(context.Background(), hetrixtools.BlacklistMonitorRequest{Target: "example.com"})
	fmt.Println(response.MonitorID)
	// Output: blacklist-1
}

func ExampleClient_CheckBlacklistDomain() {
	client, closeServer := exampleClient(map[string]string{
		"GET /v2/token/blacklist-check/domain/example.com/": `{"status":"SUCCESS","blacklisted_count":0}`,
	})
	defer closeServer()

	result, _ := client.CheckBlacklistDomain(context.Background(), "example.com")
	fmt.Println(result.BlacklistedCount)
	// Output: 0
}

func ExampleClient_ListUptimeMonitors() {
	client, closeServer := exampleClient(map[string]string{
		"GET /v3/uptime-monitors?page=1": `{"uptime_monitors":[{"id":"monitor-1","name":"Homepage","type":"website"}],"meta":{"pagination":{"current":1,"last":1}}}`,
	})
	defer closeServer()

	monitors, _ := client.ListUptimeMonitors(context.Background(), hetrixtools.ListUptimeMonitorsRequest{PaginationRequest: hetrixtools.PaginationRequest{Page: 1}})
	fmt.Println(monitors.UptimeMonitors[0].Type)
	// Output: http
}

func ExampleClient_AttachServerAgent() {
	client, closeServer := exampleClient(map[string]string{
		"POST /v3/uptime-monitors/monitor-1/server-agent": `{"agent_id":"agent-1"}`,
	})
	defer closeServer()

	agent, _ := client.AttachServerAgent(context.Background(), "monitor-1")
	fmt.Println(*agent.AgentID)
	// Output: agent-1
}

func ExampleUptimeMonitorRequest_MarshalJSON() {
	body, _ := json.Marshal(hetrixtools.UptimeMonitorRequest{
		Type:      "http",
		Name:      "Homepage",
		Target:    "https://example.com",
		Locations: []string{"new_york"},
	})
	fmt.Println(strings.Contains(string(body), `"Type":1`))
	// Output: true
}

func ExampleUptimeMonitorRequest_Validate() {
	err := hetrixtools.UptimeMonitorRequest{Type: "smtp", Target: "smtp.example.com"}.Validate()
	fmt.Println(err != nil)
	// Output: true
}

func ExampleError() {
	err := hetrixtools.Error{StatusCode: http.StatusForbidden, Body: `{"message":"forbidden"}`}

	var apiErr hetrixtools.Error
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode)
	}
	// Output: 403
}

func ExampleIsNotFound() {
	fmt.Println(hetrixtools.IsNotFound(hetrixtools.Error{StatusCode: http.StatusNotFound}))
	// Output: true
}
