package hetrixtools

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	api "github.com/filippo-claude/hetrixtools-sync/internal/hetrixapi"
)

type fakeAPI struct {
	contacts []api.ContactList
	monitors []api.UptimeMonitor
	pages    []api.StatusPage

	creates, updates, deletes, pageAdds, pageRemoves int
	createRequests                                   []api.UptimeMonitorRequest
	updateRequests                                   []api.UptimeMonitorRequest
}

func (f *fakeAPI) ListContactLists(context.Context, api.ListContactListsRequest) (*api.ListContactListsResponse, error) {
	return &api.ListContactListsResponse{ContactLists: f.contacts, Meta: onePageMeta()}, nil
}
func (f *fakeAPI) ListUptimeMonitors(context.Context, api.ListUptimeMonitorsRequest) (*api.ListUptimeMonitorsResponse, error) {
	return &api.ListUptimeMonitorsResponse{UptimeMonitors: f.monitors, Meta: onePageMeta(), MonitorsPresent: true}, nil
}
func (f *fakeAPI) CreateUptimeMonitor(_ context.Context, r api.UptimeMonitorRequest) (*api.ActionResponse, error) {
	f.creates++
	f.createRequests = append(f.createRequests, r)
	return &api.ActionResponse{Status: "SUCCESS", MonitorID: "created-" + r.Name, ServerID: "server-" + r.Name}, nil
}
func (f *fakeAPI) UpdateUptimeMonitor(_ context.Context, r api.UptimeMonitorRequest) (*api.ActionResponse, error) {
	f.updates++
	f.updateRequests = append(f.updateRequests, r)
	return &api.ActionResponse{Status: "SUCCESS"}, nil
}
func (f *fakeAPI) DeleteUptimeMonitor(context.Context, string) error { f.deletes++; return nil }
func (f *fakeAPI) ListStatusPages(context.Context, api.ListStatusPagesRequest) (*api.ListStatusPagesResponse, error) {
	return &api.ListStatusPagesResponse{StatusPages: f.pages, Meta: onePageMeta()}, nil
}
func (f *fakeAPI) AddStatusPageMonitors(context.Context, string, []string) error {
	f.pageAdds++
	return nil
}
func (f *fakeAPI) RemoveStatusPageMonitors(context.Context, string, []string) error {
	f.pageRemoves++
	return nil
}

func onePageMeta() api.Meta { return api.Meta{Pagination: api.Pagination{Current: 1, Last: 1}} }

func standardDefinitions(h *Hetrix) {
	h.WebsiteDefaults(Website{
		Locations: []string{"new_york"}, Method: "GET", AcceptedHTTPStatuses: []int{200},
		Timeout: 10 * time.Second, Frequency: time.Minute, Tries: 3,
		TriggeringLocations: 1, RepeatTimes: 3, RepeatEvery: 20 * time.Minute, MaxRedirects: 5,
	})
	h.IgnoreExisting(func(m ExistingMonitor) bool { return !contains(m.ContactLists, "managed") })
}

func exactWebsite() api.UptimeMonitor {
	public, show, yes, no := false, false, true, false
	return api.UptimeMonitor{
		ID: "m1", Type: "http", Name: "example", Target: "https://example.com", HTTPMethod: "GET",
		MaxRedirects: 5, Timeout: 10, Frequency: 1, FailsBeforeAlert: 3, FailedLocations: 1,
		ContactListID: "c1", ContactListIDs: []string{"c1"}, AlertAfter: "", RepeatTimes: 3,
		RepeatEvery: "20m", Public: &public, ShowTarget: &show, VerSSLCert: &yes, VerSSLHost: &yes,
		Locations: []string{"new_york"}, HTTPCodes: []int64{200}, NSChangeAlert: &no, MonitorStatus: "active",
		PresentFields: present("id", "name", "type", "target", "keyword", "category", "timeout", "check_frequency", "contact_lists", "monitor_status", "public_report", "public_target", "alert_after_minutes", "repeat_alert_times", "repeat_alert_frequency", "max_redirects", "http_method", "accepted_http_codes", "locations", "number_of_tries", "triggering_locations", "verify_ssl_certificate", "verify_ssl_hostname", "ssl_expiration_warn", "ssl_expiration_warn_days", "domain_expiration_warn", "domain_expiration_warn_days", "nameservers_change_warn"),
	}
}

func TestPushWithNoPreviewChangesMakesNoMutations(t *testing.T) {
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{exactWebsite()}}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"})
	}
	var out, errOut bytes.Buffer
	if err := runCLI(context.Background(), []string{"test", "push"}, &out, &errOut, defs, f); err != nil {
		t.Fatal(err)
	}
	if got := f.creates + f.updates + f.deletes + f.pageAdds + f.pageRemoves; got != 0 {
		t.Fatalf("push made %d mutations after an empty plan", got)
	}
	if !strings.Contains(out.String(), "No changes.") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPreviewAndPushPlanTheSameCreate(t *testing.T) {
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"})
	}
	previewAPI := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}}
	var preview bytes.Buffer
	if err := runCLI(context.Background(), []string{"test", "preview"}, &preview, &bytes.Buffer{}, defs, previewAPI); err != nil {
		t.Fatal(err)
	}
	if previewAPI.creates != 0 {
		t.Fatal("preview mutated remote state")
	}
	if !strings.Contains(preview.String(), "+ website example") {
		t.Fatalf("preview = %q", preview.String())
	}

	pushAPI := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}}
	var push bytes.Buffer
	if err := runCLI(context.Background(), []string{"test", "push"}, &push, &bytes.Buffer{}, defs, pushAPI); err != nil {
		t.Fatal(err)
	}
	if pushAPI.creates != 1 {
		t.Fatalf("creates = %d", pushAPI.creates)
	}
	if !strings.Contains(push.String(), "+ website example") {
		t.Fatalf("push plan = %q", push.String())
	}
}

func TestIgnoredExistingMonitorIsUntouched(t *testing.T) {
	m := exactWebsite()
	m.ContactListID = "other"
	m.ContactListIDs = []string{"other"}
	f := &fakeAPI{contacts: []api.ContactList{{ID: "other", Name: "other"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) { standardDefinitions(h) }
	var out bytes.Buffer
	if err := runCLI(context.Background(), []string{"test", "push"}, &out, &bytes.Buffer{}, defs, f); err != nil {
		t.Fatal(err)
	}
	if f.deletes != 0 {
		t.Fatal("ignored monitor was deleted")
	}
}

type malformedPaginationAPI struct{ *fakeAPI }

func (f *malformedPaginationAPI) ListUptimeMonitors(context.Context, api.ListUptimeMonitorsRequest) (*api.ListUptimeMonitorsResponse, error) {
	return &api.ListUptimeMonitorsResponse{MonitorsPresent: true, Meta: api.Meta{Pagination: api.Pagination{Current: 1, Last: 2}}}, nil
}

func TestMalformedPaginationFailsClosed(t *testing.T) {
	f := &malformedPaginationAPI{fakeAPI: &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}}}
	defs := func(h *Hetrix) { standardDefinitions(h) }
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "pagination") {
		t.Fatalf("error = %v", err)
	}
}

func TestMissingManagedRemoteIDFailsClosed(t *testing.T) {
	m := exactWebsite()
	m.ID = ""
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"})
	}
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "empty ID") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnknownManagedTypeFailsClosed(t *testing.T) {
	m := exactWebsite()
	m.Type = "smtp"
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) { standardDefinitions(h) }
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnknownManagedFieldsFailClosed(t *testing.T) {
	m := exactWebsite()
	m.UnknownFields = []string{"new_setting"}
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"})
	}
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "unknown API fields") {
		t.Fatalf("error = %v", err)
	}
}

func TestCronDifferenceFailsRatherThanUnsafeUpsert(t *testing.T) {
	public, show := false, false
	m := api.UptimeMonitor{ID: "cron1", Type: "heartbeat", Name: "job", Timeout: 900, ContactListID: "c1", ContactListIDs: []string{"c1"}, Public: &public, ShowTarget: &show, MonitorStatus: "active", PresentFields: present("id", "name", "type", "contact_lists", "category", "timeout", "monitor_status", "public_report", "public_target", "alert_after_minutes", "repeat_alert_times", "repeat_alert_frequency", "grace", "agent_id")}
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) {
		h.IgnoreExisting(func(m ExistingMonitor) bool { return !contains(m.ContactLists, "managed") })
		h.CronDefaults(Cron{Interval: 15 * time.Minute})
		h.Cron(Cron{Name: "job", ContactList: "managed", Category: "changed"})
	}
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "safe update") {
		t.Fatalf("error = %v", err)
	}
	if f.updates != 0 {
		t.Fatal("unsafe cron update was attempted")
	}
}

func TestWebsiteUpdatePreservesOutOfSurfaceSettings(t *testing.T) {
	m := exactWebsite()
	m.Target = "https://old.example.com"
	m.SSLExpirationReminder = 5
	m.DomainExpirationReminder = 30
	yes := true
	m.NSChangeAlert = &yes
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{m}}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		h.Website(Website{Name: "example", Target: "https://new.example.com", ContactList: "managed"})
	}
	if err := runCLI(context.Background(), []string{"test", "push"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f); err != nil {
		t.Fatal(err)
	}
	if len(f.updateRequests) != 1 {
		t.Fatalf("updates = %d", len(f.updateRequests))
	}
	r := f.updateRequests[0]
	if r.SSLExpirationReminder != 5 || r.DomainExpirationReminder != 30 || r.NSChangeAlert == nil || !*r.NSChangeAlert {
		t.Fatalf("preserved settings = SSL %d domain %d NS %#v", r.SSLExpirationReminder, r.DomainExpirationReminder, r.NSChangeAlert)
	}
}

func TestInvalidLocationFailsDuringPreview(t *testing.T) {
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}}
	defs := func(h *Hetrix) {
		h.IgnoreExisting(func(ExistingMonitor) bool { return false })
		h.Website(Website{Name: "x", Target: "https://example.com", ContactList: "managed", Method: "GET", Locations: []string{"moon"}, AcceptedHTTPStatuses: []int{200}, Timeout: time.Second, Frequency: time.Minute, Tries: 1, TriggeringLocations: 1, RepeatEvery: time.Minute, MaxRedirects: 1})
	}
	err := runCLI(context.Background(), []string{"test", "preview"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f)
	if err == nil || !strings.Contains(err.Error(), "unsupported location") {
		t.Fatalf("error = %v", err)
	}
	if f.creates != 0 {
		t.Fatal("invalid preview mutated remote state")
	}
}

func TestDanglingStatusPageIDIsPreserved(t *testing.T) {
	f := &fakeAPI{
		contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, monitors: []api.UptimeMonitor{exactWebsite()},
		pages: []api.StatusPage{{ID: "p1", Name: "Status", Monitors: []string{"m1", "dangling"}, MonitorsPresent: true}},
	}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		page := h.StatusPage(StatusPage{Name: "Status"})
		page.Add(h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"}))
	}
	var out bytes.Buffer
	if err := runCLI(context.Background(), []string{"test", "push"}, &out, &bytes.Buffer{}, defs, f); err != nil {
		t.Fatal(err)
	}
	if f.pageRemoves != 0 {
		t.Fatal("dangling ID was removed")
	}
	if !strings.Contains(out.String(), "preserving it") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCreatedMonitorIsAddedToStatusPage(t *testing.T) {
	f := &fakeAPI{contacts: []api.ContactList{{ID: "c1", Name: "managed"}}, pages: []api.StatusPage{{ID: "p1", Name: "Status", MonitorsPresent: true}}}
	defs := func(h *Hetrix) {
		standardDefinitions(h)
		page := h.StatusPage(StatusPage{Name: "Status"})
		page.Add(h.Website(Website{Name: "example", Target: "https://example.com", ContactList: "managed"}))
	}
	if err := runCLI(context.Background(), []string{"test", "push"}, &bytes.Buffer{}, &bytes.Buffer{}, defs, f); err != nil {
		t.Fatal(err)
	}
	if f.creates != 1 || f.pageAdds != 1 {
		t.Fatalf("creates=%d pageAdds=%d", f.creates, f.pageAdds)
	}
}

func TestPrintPlanRefusesUnknownOperationKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for an operation preview cannot describe")
		}
	}()
	printPlan(&bytes.Buffer{}, &plan{operations: []operation{{kind: operationKind(99)}}})
}

func present(fields ...string) map[string]bool {
	out := make(map[string]bool, len(fields))
	for _, field := range fields {
		out[field] = true
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
