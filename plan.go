package hetrixtools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	api "github.com/filippo-claude/hetrixtools-sync/internal/hetrixapi"
)

type apiClient interface {
	ListContactLists(context.Context, api.ListContactListsRequest) (*api.ListContactListsResponse, error)
	ListUptimeMonitors(context.Context, api.ListUptimeMonitorsRequest) (*api.ListUptimeMonitorsResponse, error)
	CreateUptimeMonitor(context.Context, api.UptimeMonitorRequest) (*api.ActionResponse, error)
	UpdateUptimeMonitor(context.Context, api.UptimeMonitorRequest) (*api.ActionResponse, error)
	DeleteUptimeMonitor(context.Context, string) error
	ListStatusPages(context.Context, api.ListStatusPagesRequest) (*api.ListStatusPagesResponse, error)
	AddStatusPageMonitors(context.Context, string, []string) error
	RemoveStatusPageMonitors(context.Context, string, []string) error
}

type operationKind int

const (
	opCreate operationKind = iota
	opUpdate
	opPageAdd
	opPageRemove
	opDelete
)

type operation struct {
	kind operationKind
	key  string
	name string
	id   string

	contactID string

	desired desiredMonitor
	actual  *remoteMonitor
	diffs   []fieldDiff

	pageID   string
	pageName string
	memberID string
}

type fieldDiff struct {
	name string
	old  string
	new  string
}

type plan struct {
	operations []operation
	warnings   []string
}

type remoteMonitor struct {
	key          string
	kind         MonitorKind
	id           string
	name         string
	contactList  string
	contactLists []string
	raw          api.UptimeMonitor
}

type remoteState struct {
	monitors     []remoteMonitor
	byID         map[string]*remoteMonitor
	statusPages  []api.StatusPage
	contactIDs   map[string]string
	contactNames map[string]string
}

func buildPlan(ctx context.Context, h *Hetrix, client apiClient) (*plan, error) {
	if err := h.validateDefinitions(); err != nil {
		return nil, err
	}
	if h.ignoreExisting == nil {
		return nil, fmt.Errorf("IgnoreExisting must be configured to establish an ownership boundary")
	}

	state, err := loadRemoteState(ctx, client, len(h.statusPages) > 0)
	if err != nil {
		return nil, err
	}

	managedByKey := make(map[string][]*remoteMonitor)
	for i := range state.monitors {
		remote := &state.monitors[i]
		existing := ExistingMonitor{ID: remote.id, Kind: remote.kind, Name: remote.name, ContactLists: append([]string(nil), remote.contactLists...)}
		if h.ignoreExisting(existing) {
			if _, collision := h.monitors[remote.key]; collision {
				return nil, fmt.Errorf("desired monitor %q collides with an ignored remote monitor", remote.name)
			}
			continue
		}
		if err := validateRemoteCompleteness(remote); err != nil {
			return nil, err
		}
		if len(remote.raw.UnknownFields) > 0 {
			return nil, fmt.Errorf("managed remote monitor %q contains unknown API fields: %s", remote.name, strings.Join(remote.raw.UnknownFields, ", "))
		}
		if remote.kind != WebsiteMonitor && remote.kind != CronMonitor {
			return nil, fmt.Errorf("managed remote monitor %q has unsupported type %q; ignore it explicitly or add support", remote.name, remote.raw.Type)
		}
		if len(remote.contactLists) != 1 {
			return nil, fmt.Errorf("managed remote monitor %q has %d contact lists; safe updates require exactly one", remote.name, len(remote.contactLists))
		}
		if remote.raw.MonitorStatus != "active" {
			return nil, fmt.Errorf("managed remote monitor %q has unsupported status %q", remote.name, remote.raw.MonitorStatus)
		}
		managedByKey[remote.key] = append(managedByKey[remote.key], remote)
	}

	for _, desired := range h.monitors {
		contact := desiredContactList(desired)
		probe := ExistingMonitor{Kind: desired.kind, Name: monitorName(desired), ContactLists: []string{contact}}
		if h.ignoreExisting(probe) {
			return nil, fmt.Errorf("desired monitor %q is excluded by IgnoreExisting", monitorName(desired))
		}
		if _, ok := state.contactIDs[contact]; !ok {
			return nil, fmt.Errorf("desired monitor %q references missing contact list %q", monitorName(desired), contact)
		}
	}

	p := &plan{}
	chosen := make(map[string]*remoteMonitor)
	keys := append([]string(nil), h.monitorOrder...)
	sort.Slice(keys, func(i, j int) bool { return monitorName(h.monitors[keys[i]]) < monitorName(h.monitors[keys[j]]) })
	for _, key := range keys {
		desired := h.monitors[key]
		remotes := managedByKey[key]
		switch len(remotes) {
		case 0:
			p.operations = append(p.operations, operation{kind: opCreate, key: key, name: monitorName(desired), contactID: state.contactIDs[desiredContactList(desired)], desired: desired})
		case 1:
			remote := remotes[0]
			chosen[key] = remote
			diffs, err := compareMonitor(desired, remote)
			if err != nil {
				return nil, err
			}
			if len(diffs) > 0 {
				if desired.kind == CronMonitor {
					return nil, fmt.Errorf("cron monitor %q differs, but HetrixTools does not expose all heartbeat fields needed for a safe update: %s", remote.name, formatDiffSummary(diffs))
				}
				if err := validateSafeWebsiteUpdate(desired.website, remote.raw, diffs); err != nil {
					return nil, err
				}
				p.operations = append(p.operations, operation{kind: opUpdate, key: key, name: remote.name, id: remote.id, contactID: state.contactIDs[desiredContactList(desired)], desired: desired, actual: remote, diffs: diffs})
			}
		default:
			sort.Slice(remotes, func(i, j int) bool { return remotes[i].id < remotes[j].id })
			var baseline []fieldDiff
			allExact := true
			for _, remote := range remotes {
				diffs, err := compareMonitor(desired, remote)
				if err != nil {
					return nil, err
				}
				if len(diffs) != 0 {
					allExact = false
					baseline = diffs
					break
				}
			}
			if !allExact {
				return nil, fmt.Errorf("managed monitor %q exists %d times with conflicting state; refusing to choose one (%s)", monitorName(desired), len(remotes), formatDiffSummary(baseline))
			}
			chosen[key] = remotes[0]
			p.warnings = append(p.warnings, fmt.Sprintf("monitor %q exists %d times with identical managed settings; leaving duplicates untouched", monitorName(desired), len(remotes)))
		}
	}

	for key, remotes := range managedByKey {
		if _, ok := h.monitors[key]; ok {
			continue
		}
		for _, remote := range remotes {
			p.operations = append(p.operations, operation{kind: opDelete, key: key, name: remote.name, id: remote.id, actual: remote})
		}
	}

	if err := planStatusPages(h, state, chosen, p); err != nil {
		return nil, err
	}
	sort.Strings(p.warnings)
	sortOperations(p.operations)
	for _, op := range p.operations {
		var request api.UptimeMonitorRequest
		switch op.kind {
		case opCreate:
			request = requestForCreate(op.desired, op.contactID)
		case opUpdate:
			request = requestForUpdate(op.desired, op.id, op.contactID, &op.actual.raw)
		default:
			continue
		}
		if _, err := json.Marshal(request); err != nil {
			return nil, fmt.Errorf("planned %s %q cannot be encoded safely: %w", map[operationKind]string{opCreate: "create", opUpdate: "update"}[op.kind], op.name, err)
		}
	}
	return p, nil
}

func loadRemoteState(ctx context.Context, client apiClient, includeStatusPages bool) (*remoteState, error) {
	state := &remoteState{byID: make(map[string]*remoteMonitor), contactIDs: make(map[string]string), contactNames: make(map[string]string)}
	for page := 1; ; page++ {
		response, err := client.ListContactLists(ctx, api.ListContactListsRequest{PaginationRequest: api.PaginationRequest{Page: page, PerPage: 200}})
		if err != nil {
			return nil, fmt.Errorf("list contact lists: %w", err)
		}
		if err := validatePagination("contact lists", page, response.Meta.Pagination); err != nil {
			return nil, err
		}
		for _, contact := range response.ContactLists {
			if strings.TrimSpace(contact.ID) == "" || strings.TrimSpace(contact.Name) == "" {
				return nil, fmt.Errorf("contact-list response contains an empty ID or name")
			}
			if old, ok := state.contactIDs[contact.Name]; ok && old != contact.ID {
				return nil, fmt.Errorf("contact list name %q is ambiguous", contact.Name)
			}
			if old, ok := state.contactNames[contact.ID]; ok && old != contact.Name {
				return nil, fmt.Errorf("contact list ID %q has conflicting names", contact.ID)
			}
			state.contactIDs[contact.Name] = contact.ID
			state.contactNames[contact.ID] = contact.Name
		}
		if page >= response.Meta.Pagination.Last {
			break
		}
	}
	for page := 1; ; page++ {
		response, err := client.ListUptimeMonitors(ctx, api.ListUptimeMonitorsRequest{PaginationRequest: api.PaginationRequest{Page: page, PerPage: 200}})
		if err != nil {
			return nil, fmt.Errorf("list uptime monitors: %w", err)
		}
		if err := validatePagination("uptime monitors", page, response.Meta.Pagination); err != nil {
			return nil, err
		}
		if !response.MonitorsPresent {
			return nil, fmt.Errorf("uptime-monitor response omitted the monitor list")
		}
		for _, monitor := range response.UptimeMonitors {
			kind := remoteKind(monitor.Type)
			contacts := remoteContactNames(monitor, state.contactNames)
			remote := remoteMonitor{kind: kind, id: monitor.ID, name: monitor.Name, contactLists: contacts, raw: monitor}
			if len(contacts) == 1 {
				remote.contactList = contacts[0]
			}
			remote.key = monitorKey(kind, monitor.Name)
			state.monitors = append(state.monitors, remote)
		}
		if page >= response.Meta.Pagination.Last {
			break
		}
	}
	for i := range state.monitors {
		id := state.monitors[i].id
		if id == "" {
			continue
		} // Managed empty IDs are rejected after ownership filtering.
		if _, duplicate := state.byID[id]; duplicate {
			return nil, fmt.Errorf("uptime monitor ID %q appears more than once", id)
		}
		state.byID[id] = &state.monitors[i]
	}
	if includeStatusPages {
		for page := 1; ; page++ {
			response, err := client.ListStatusPages(ctx, api.ListStatusPagesRequest{PaginationRequest: api.PaginationRequest{Page: page, PerPage: 100}})
			if err != nil {
				return nil, fmt.Errorf("list status pages: %w", err)
			}
			if err := validatePagination("status pages", page, response.Meta.Pagination); err != nil {
				return nil, err
			}
			state.statusPages = append(state.statusPages, response.StatusPages...)
			if page >= response.Meta.Pagination.Last {
				break
			}
		}
	}
	return state, nil
}

func validatePagination(resource string, requested int, p api.Pagination) error {
	if p.Current != requested || p.Current < 1 || p.Last < p.Current {
		return fmt.Errorf("%s returned inconsistent pagination: requested=%d current=%d last=%d", resource, requested, p.Current, p.Last)
	}
	if p.Current < p.Last {
		if p.Next == nil || *p.Next != p.Current+1 {
			return fmt.Errorf("%s pagination omitted or misreported next page", resource)
		}
	} else if p.Next != nil {
		return fmt.Errorf("%s pagination reported a next page after the last page", resource)
	}
	return nil
}

func validateRemoteCompleteness(remote *remoteMonitor) error {
	if remote.id == "" {
		return fmt.Errorf("managed remote monitor %q has an empty ID", remote.name)
	}
	if strings.TrimSpace(remote.name) == "" {
		return fmt.Errorf("managed remote monitor %s has an empty name", remote.id)
	}
	if remote.raw.PresentFields == nil {
		return fmt.Errorf("managed remote monitor %q has no API field-presence metadata", remote.name)
	}
	required := []string{"id", "name", "type", "contact_lists", "category", "timeout", "monitor_status", "public_report", "public_target", "alert_after_minutes", "repeat_alert_times", "repeat_alert_frequency"}
	if remote.kind == WebsiteMonitor {
		required = append(required, "target", "keyword", "max_redirects", "http_method", "accepted_http_codes", "locations", "check_frequency", "number_of_tries", "triggering_locations", "verify_ssl_certificate", "verify_ssl_hostname", "ssl_expiration_warn", "ssl_expiration_warn_days", "domain_expiration_warn", "domain_expiration_warn_days", "nameservers_change_warn")
	} else if remote.kind == CronMonitor {
		required = append(required, "grace", "agent_id")
	}
	var missing []string
	for _, field := range required {
		if !remote.raw.PresentFields[field] {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("managed remote monitor %q omitted required API fields: %s", remote.name, strings.Join(missing, ", "))
	}
	return nil
}

func remoteKind(kind string) MonitorKind {
	switch kind {
	case "http", "website":
		return WebsiteMonitor
	case "heartbeat":
		return CronMonitor
	default:
		return MonitorKind(kind)
	}
}

func remoteContactNames(m api.UptimeMonitor, names map[string]string) []string {
	ids := append([]string(nil), m.ContactListIDs...)
	if len(ids) == 0 && m.ContactListID != "" {
		ids = []string{m.ContactListID}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := names[id]; ok {
			out = append(out, name)
		} else {
			out = append(out, "<unknown:"+id+">")
		}
	}
	sort.Strings(out)
	return out
}

func compareMonitor(desired desiredMonitor, remote *remoteMonitor) ([]fieldDiff, error) {
	if desired.kind != remote.kind {
		return nil, fmt.Errorf("monitor %q changed type from %s to %s; rename or remove it explicitly", remote.name, remote.kind, desired.kind)
	}
	var diffs []fieldDiff
	add := func(name string, old, new any) {
		oldS, newS := fmt.Sprint(old), fmt.Sprint(new)
		if oldS != newS {
			diffs = append(diffs, fieldDiff{name: name, old: oldS, new: newS})
		}
	}
	add("contact_list", remote.contactList, desiredContactList(desired))
	if desired.kind == WebsiteMonitor {
		w, a := desired.website, remote.raw
		if a.VerSSLCert == nil || a.VerSSLHost == nil || a.Public == nil || a.ShowTarget == nil {
			return nil, fmt.Errorf("website monitor %q omitted mutable fields in the API response; refusing an incomplete comparison", remote.name)
		}
		add("target", a.Target, w.Target)
		add("category", a.Category, w.Category)
		add("method", strings.ToUpper(a.HTTPMethod), strings.ToUpper(w.Method))
		add("keyword", a.Keyword, w.Keyword)
		add("accepted_http_statuses", ints64String(a.HTTPCodes), intsString(w.AcceptedHTTPStatuses))
		add("locations", strings.Join(sortedStrings(a.Locations), ","), strings.Join(sortedStrings(w.Locations), ","))
		add("timeout", secondsString(a.Timeout), w.Timeout.String())
		add("frequency", minutesString(a.Frequency), w.Frequency.String())
		add("tries", a.FailsBeforeAlert, w.Tries)
		add("triggering_locations", a.FailedLocations, w.TriggeringLocations)
		add("alert_after", normalizeAPIDuration(a.AlertAfter), durationMinutesString(w.AlertAfter))
		add("repeat_times", a.RepeatTimes, w.RepeatTimes)
		add("repeat_every", normalizeAPIDuration(a.RepeatEvery), durationMinutesString(w.RepeatEvery))
		add("max_redirects", a.MaxRedirects, w.MaxRedirects)
		add("public", *a.Public, w.Public)
		add("show_target", *a.ShowTarget, w.ShowTarget)
		add("verify_ssl_certificate", *a.VerSSLCert, !w.DisableSSLCertificateVerification)
		add("verify_ssl_hostname", *a.VerSSLHost, !w.DisableSSLHostnameVerification)
	} else {
		c, a := desired.cron, remote.raw
		if a.Public == nil || a.ShowTarget == nil {
			return nil, fmt.Errorf("cron monitor %q omitted mutable fields in the API response; refusing an incomplete comparison", remote.name)
		}
		add("category", a.Category, c.Category)
		add("interval", secondsString(a.Timeout), c.Interval.String())
		add("grace", secondsString(a.Grace), c.Grace.String())
		add("alert_after", normalizeAPIDuration(a.AlertAfter), durationMinutesString(c.AlertAfter))
		add("repeat_times", a.RepeatTimes, c.RepeatTimes)
		add("repeat_every", normalizeOptionalAPIDuration(a.RepeatEvery), optionalDurationMinutesString(c.RepeatEvery))
		add("public", *a.Public, c.Public)
		add("show_target", *a.ShowTarget, c.ShowTarget)
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].name < diffs[j].name })
	return diffs, nil
}

func validateSafeWebsiteUpdate(desired Website, actual api.UptimeMonitor, diffs []fieldDiff) error {
	for _, diff := range diffs {
		switch diff.name {
		case "keyword":
			if desired.Keyword == "" {
				return fmt.Errorf("website monitor %q requires clearing keyword, which the HetrixTools update API cannot represent safely", desired.Name)
			}
		case "category":
			if desired.Category == "" {
				return fmt.Errorf("website monitor %q requires clearing category, which the HetrixTools update API cannot represent safely", desired.Name)
			}
		}
	}
	_ = actual
	return nil
}

func planStatusPages(h *Hetrix, state *remoteState, chosen map[string]*remoteMonitor, p *plan) error {
	pageNames := make([]string, 0, len(h.statusPages))
	for name := range h.statusPages {
		pageNames = append(pageNames, name)
	}
	sort.Strings(pageNames)
	for _, name := range pageNames {
		decl := h.statusPages[name]
		var matches []api.StatusPage
		for _, page := range state.statusPages {
			if page.Name == name {
				matches = append(matches, page)
			}
		}
		if len(matches) == 0 {
			return fmt.Errorf("status page %q does not exist", name)
		}
		if len(matches) > 1 {
			return fmt.Errorf("status page name %q is ambiguous", name)
		}
		page := matches[0]
		if strings.TrimSpace(page.ID) == "" {
			return fmt.Errorf("status page %q has an empty ID", name)
		}
		if !page.MonitorsPresent {
			return fmt.Errorf("status page %q response omitted monitor membership", name)
		}
		current := make(map[string]struct{}, len(page.Monitors))
		for _, id := range page.Monitors {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("status page %q contains an empty monitor ID", name)
			}
			current[id] = struct{}{}
		}

		for key := range decl.members {
			if remote := chosen[key]; remote != nil {
				if _, ok := current[remote.id]; !ok {
					p.operations = append(p.operations, operation{kind: opPageAdd, key: key, name: monitorName(h.monitors[key]), pageID: page.ID, pageName: page.Name, memberID: remote.id})
				}
				continue
			}
			// A create operation will provide the ID during push.
			p.operations = append(p.operations, operation{kind: opPageAdd, key: key, name: monitorName(h.monitors[key]), pageID: page.ID, pageName: page.Name})
		}

		for id := range current {
			remote := state.byID[id]
			if remote == nil {
				p.warnings = append(p.warnings, fmt.Sprintf("status page %q contains dangling monitor ID %s; preserving it", page.Name, id))
				continue
			}
			existing := ExistingMonitor{ID: remote.id, Kind: remote.kind, Name: remote.name, ContactLists: remote.contactLists}
			if h.ignoreExisting(existing) {
				continue
			}
			if _, wanted := decl.members[remote.key]; !wanted {
				p.operations = append(p.operations, operation{kind: opPageRemove, key: remote.key, name: remote.name, pageID: page.ID, pageName: page.Name, memberID: id})
			}
		}
	}
	return nil
}

func desiredContactList(m desiredMonitor) string {
	if m.kind == WebsiteMonitor {
		return m.website.ContactList
	}
	return m.cron.ContactList
}

func requestForCreate(m desiredMonitor, contactID string) api.UptimeMonitorRequest {
	return requestForMonitor(m, "", contactID, nil)
}
func requestForUpdate(m desiredMonitor, id, contactID string, actual *api.UptimeMonitor) api.UptimeMonitorRequest {
	return requestForMonitor(m, id, contactID, actual)
}
func requestForMonitor(m desiredMonitor, id, contactID string, actual *api.UptimeMonitor) api.UptimeMonitorRequest {
	if m.kind == WebsiteMonitor {
		w := m.website
		public, showTarget := w.Public, w.ShowTarget
		verifyCert, verifyHost := !w.DisableSSLCertificateVerification, !w.DisableSSLHostnameVerification
		r := api.UptimeMonitorRequest{
			MID: id, Type: "http", Name: w.Name, Target: w.Target, HTTPMethod: w.Method,
			MaxRedirects: int64(w.MaxRedirects), Timeout: int64(w.Timeout / time.Second),
			Frequency: int64(w.Frequency / time.Minute), FailsBeforeAlert: int64(w.Tries),
			FailedLocations: int64(w.TriggeringLocations), ContactList: contactID, Category: w.Category,
			AlertAfter: durationMinutesString(w.AlertAfter), RepeatTimes: int64(w.RepeatTimes),
			RepeatEvery: durationMinutesString(w.RepeatEvery), Public: &public, ShowTarget: &showTarget,
			VerSSLCert: &verifyCert, VerSSLHost: &verifyHost, Locations: sortedStrings(w.Locations),
			Keyword: w.Keyword, HTTPCodes: intsToInt64s(w.AcceptedHTTPStatuses),
		}
		if actual != nil {
			r.SSLExpirationReminder = actual.SSLExpirationReminder
			r.DomainExpirationReminder = actual.DomainExpirationReminder
			r.NSChangeAlert = actual.NSChangeAlert
		}
		return r
	}
	c := m.cron
	public, showTarget := c.Public, c.ShowTarget
	return api.UptimeMonitorRequest{
		MID: id, Type: "heartbeat", Name: c.Name, Timeout: int64(c.Interval / time.Second),
		Grace: int64(c.Grace / time.Second), ContactList: contactID, Category: c.Category,
		AlertAfter: durationMinutesString(c.AlertAfter), RepeatTimes: int64(c.RepeatTimes),
		RepeatEvery: optionalDurationMinutesString(c.RepeatEvery), Public: &public, ShowTarget: &showTarget,
	}
}

func sortOperations(ops []operation) {
	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].kind != ops[j].kind {
			return ops[i].kind < ops[j].kind
		}
		if ops[i].pageName != ops[j].pageName {
			return ops[i].pageName < ops[j].pageName
		}
		return ops[i].name < ops[j].name
	})
}

func formatDiffSummary(diffs []fieldDiff) string {
	parts := make([]string, len(diffs))
	for i, diff := range diffs {
		parts[i] = diff.name + ": " + diff.old + " -> " + diff.new
	}
	return strings.Join(parts, ", ")
}
func durationMinutesString(d time.Duration) string {
	return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
}
func optionalDurationMinutesString(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return durationMinutesString(d)
}
func normalizeAPIDuration(s string) string {
	if s == "" {
		return "0m"
	}
	return s
}
func normalizeOptionalAPIDuration(s string) string {
	if s == "0m" {
		return ""
	}
	return s
}
func secondsString(v int64) string { return (time.Duration(v) * time.Second).String() }
func minutesString(v int64) string { return (time.Duration(v) * time.Minute).String() }
func intsString(v []int) string {
	x := cloneInts(v)
	sort.Ints(x)
	parts := make([]string, len(x))
	for i, n := range x {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}
func ints64String(v []int64) string {
	x := append([]int64(nil), v...)
	sort.Slice(x, func(i, j int) bool { return x[i] < x[j] })
	parts := make([]string, len(x))
	for i, n := range x {
		parts[i] = strconv.FormatInt(n, 10)
	}
	return strings.Join(parts, ",")
}
func intsToInt64s(v []int) []int64 {
	out := make([]int64, len(v))
	for i, n := range v {
		out[i] = int64(n)
	}
	return out
}
func sortedStrings(v []string) []string {
	out := append([]string(nil), v...)
	sort.Strings(out)
	return out
}
