package hetrixtools

import (
	"strings"
	"testing"
)

func TestValidateQueryRejectsInvalidPagination(t *testing.T) {
	t.Parallel()

	err := validateQuery(ListContactListsRequest{PaginationRequest: PaginationRequest{Page: -1, PerPage: 201}})
	if err == nil {
		t.Fatal("expected invalid pagination query")
	}
	if got := err.Error(); !strings.Contains(got, "Page") || !strings.Contains(got, "PerPage") {
		t.Fatalf("error = %q, want Page and PerPage validation details", got)
	}
}

func TestValidateQueryRejectsInvalidEnumsAndDates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query any
	}{
		{name: "blacklist type", query: ListBlacklistMonitorsRequest{Type: "http"}},
		{name: "blacklist cidr", query: ListBlacklistMonitorsRequest{CIDR: 20}},
		{name: "blacklist report date", query: GetBlacklistMonitorReportRequest{Date: "07-05-2026"}},
		{name: "uptime id", query: ListUptimeMonitorsRequest{ID: "up-1"}},
		{name: "uptime type", query: ListUptimeMonitorsRequest{Type: "http"}},
		{name: "uptime order", query: ListUptimeMonitorsRequest{Order: "newest"}},
		{name: "uptime report days", query: GetUptimeMonitorReportRequest{Days: -7}},
		{name: "uptime report month", query: GetUptimeMonitorReportRequest{Month: "2026/07"}},
		{name: "uptime fail-log minutes", query: GetUptimeMonitorLocationFailLogRequest{Minutes: -1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := validateQuery(test.query); err == nil {
				t.Fatal("expected query validation error")
			}
		})
	}
}

func TestValidateQueryAcceptsValidQueries(t *testing.T) {
	t.Parallel()

	tests := []any{
		ListContactListsRequest{PaginationRequest: PaginationRequest{Page: 1, PerPage: 100}},
		ListBlacklistMonitorsRequest{Type: "domain", CIDR: 24, Order: "asc"},
		GetBlacklistMonitorReportRequest{Date: "2026-07-05"},
		ListUptimeMonitorsRequest{ID: "31adab6c21406254efda58b0020b7e8e", Type: "website", Order: "desc"},
		GetUptimeMonitorReportRequest{Days: 30, Month: "2026-07"},
		GetUptimeMonitorLocationFailLogRequest{Minutes: 5},
		ListStatusPagesRequest{PaginationRequest: PaginationRequest{Page: 2}},
		ListScheduledMaintenancesRequest{PaginationRequest: PaginationRequest{PerPage: 50}},
	}

	for _, query := range tests {
		if err := validateQuery(query); err != nil {
			t.Fatalf("validateQuery(%#v) returned error: %s", query, err)
		}
	}
}

func TestValidateRequestRejectsInvalidBlacklistMonitorRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request BlacklistMonitorRequest
	}{
		{name: "missing target", request: BlacklistMonitorRequest{}},
		{name: "invalid target", request: BlacklistMonitorRequest{Target: "bad target"}},
		{name: "invalid label", request: BlacklistMonitorRequest{Target: "example.com", Label: "Bad/Label"}},
		{name: "invalid contact", request: BlacklistMonitorRequest{Target: "example.com", Contact: "contacts-1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := validateRequest(test.request); err == nil {
				t.Fatal("expected request validation error")
			}
		})
	}
}

func TestValidateRequestAcceptsValidBlacklistMonitorRequest(t *testing.T) {
	t.Parallel()

	request := BlacklistMonitorRequest{
		Target:  "example.com",
		Label:   "Example Monitor",
		Contact: "31adab6c21406254efda58b0020b7e8e",
	}
	if err := validateRequest(request); err != nil {
		t.Fatalf("validateRequest returned error: %s", err)
	}
}
