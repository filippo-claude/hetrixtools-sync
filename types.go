// Package hetrixtools provides a small declarative interface for HetrixTools.
//
// Programs define monitors in ordinary Go, then call Main to expose preview and
// push commands. Preview and push share the same planner: push only executes
// operations emitted by that planner.
package hetrixtools

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// MonitorKind is a supported HetrixTools monitor type.
type MonitorKind string

const (
	WebsiteMonitor MonitorKind = "website"
	CronMonitor    MonitorKind = "cron"
)

// Website describes an HTTP website monitor. Zero fields inherit from the
// WebsiteDefaults value. Boolean fields are deliberately phrased so their zero
// value is useful; TLS verification is enabled unless explicitly disabled.
type Website struct {
	Name        string
	Target      string
	ContactList string
	Category    string

	Method               string
	Keyword              string
	AcceptedHTTPStatuses []int
	Locations            []string
	Timeout              time.Duration
	Frequency            time.Duration
	Tries                int
	TriggeringLocations  int
	AlertAfter           time.Duration
	RepeatTimes          int
	RepeatEvery          time.Duration
	MaxRedirects         int

	Public                            bool
	ShowTarget                        bool
	DisableSSLCertificateVerification bool
	DisableSSLHostnameVerification    bool
}

// Cron describes a plain HetrixTools heartbeat/dead-man-switch monitor.
// Interval is the expected time between heartbeats; Grace is the additional
// grace period. Update requests keep server-agent detail sections private.
type Cron struct {
	Name        string
	ContactList string
	Category    string

	Interval    time.Duration
	Grace       time.Duration
	AlertAfter  time.Duration
	RepeatTimes int
	RepeatEvery time.Duration

	Public     bool
	ShowTarget bool
}

// ExistingMonitor is passed to IgnoreExisting. ContactLists are resolved to
// their human-readable names before the callback runs.
type ExistingMonitor struct {
	ID           string
	Kind         MonitorKind
	Name         string
	ContactLists []string
}

// StatusPage identifies an existing status page whose managed membership will
// be reconciled. Status-page creation and presentation settings are outside the
// API surface; only monitor membership is changed.
type StatusPage struct {
	Name string
}

// MonitorRef refers to a monitor declared by Website or Cron.
type MonitorRef struct {
	owner *Hetrix
	key   string
}

// StatusPageRef collects desired monitor membership.
type StatusPageRef struct {
	owner   *Hetrix
	name    string
	members map[string]struct{}
}

// Add includes monitors in this status page. Membership is exact for live,
// managed monitors; dangling remote IDs are preserved because they cannot be
// safely identified.
func (p *StatusPageRef) Add(monitors ...MonitorRef) {
	for _, monitor := range monitors {
		if monitor.owner != p.owner {
			p.owner.addDefinitionError(fmt.Errorf("status page %q: monitor reference belongs to another definition set", p.name))
			continue
		}
		if monitor.key == "" {
			p.owner.addDefinitionError(fmt.Errorf("status page %q: empty monitor reference", p.name))
			continue
		}
		p.members[monitor.key] = struct{}{}
	}
}

type desiredMonitor struct {
	kind    MonitorKind
	website Website
	cron    Cron
}

// Hetrix accumulates declarations. Its methods do no network I/O.
type Hetrix struct {
	websiteDefaults Website
	cronDefaults    Cron
	ignoreExisting  func(ExistingMonitor) bool
	monitors        map[string]desiredMonitor
	monitorOrder    []string
	statusPages     map[string]*StatusPageRef
	definitionErrs  []error
}

func newHetrix() *Hetrix {
	return &Hetrix{
		monitors:    make(map[string]desiredMonitor),
		statusPages: make(map[string]*StatusPageRef),
	}
}

// WebsiteDefaults sets defaults applied to subsequently declared websites.
func (h *Hetrix) WebsiteDefaults(defaults Website) {
	h.websiteDefaults = defaults
}

// CronDefaults sets defaults applied to subsequently declared cron monitors.
func (h *Hetrix) CronDefaults(defaults Cron) {
	h.cronDefaults = defaults
}

// IgnoreExisting excludes remote monitors from reconciliation. A desired
// monitor that would match this predicate is rejected during planning.
func (h *Hetrix) IgnoreExisting(ignore func(ExistingMonitor) bool) {
	if h.ignoreExisting != nil {
		h.addDefinitionError(fmt.Errorf("IgnoreExisting called more than once"))
		return
	}
	h.ignoreExisting = ignore
}

// Website declares a website monitor and returns a reference usable by a
// status page.
func (h *Hetrix) Website(m Website) MonitorRef {
	m = mergeWebsite(h.websiteDefaults, m)
	key := monitorKey(WebsiteMonitor, m.Name)
	h.addMonitor(key, desiredMonitor{kind: WebsiteMonitor, website: m})
	return MonitorRef{owner: h, key: key}
}

// Cron declares a cron/heartbeat monitor and returns a reference usable by a
// status page.
func (h *Hetrix) Cron(m Cron) MonitorRef {
	m = mergeCron(h.cronDefaults, m)
	key := monitorKey(CronMonitor, m.Name)
	h.addMonitor(key, desiredMonitor{kind: CronMonitor, cron: m})
	return MonitorRef{owner: h, key: key}
}

// StatusPage declares managed membership for an existing status page.
func (h *Hetrix) StatusPage(page StatusPage) *StatusPageRef {
	name := strings.TrimSpace(page.Name)
	if name == "" {
		h.addDefinitionError(fmt.Errorf("status page name is required"))
		name = "<invalid>"
	}
	if existing, ok := h.statusPages[name]; ok {
		h.addDefinitionError(fmt.Errorf("status page %q declared more than once", name))
		return existing
	}
	ref := &StatusPageRef{owner: h, name: name, members: make(map[string]struct{})}
	h.statusPages[name] = ref
	return ref
}

func (h *Hetrix) addMonitor(key string, monitor desiredMonitor) {
	if _, ok := h.monitors[key]; ok {
		h.addDefinitionError(fmt.Errorf("monitor %q declared more than once", monitorName(monitor)))
		return
	}
	h.monitors[key] = monitor
	h.monitorOrder = append(h.monitorOrder, key)
}

func (h *Hetrix) addDefinitionError(err error) {
	h.definitionErrs = append(h.definitionErrs, err)
}

func (h *Hetrix) validateDefinitions() error {
	if len(h.definitionErrs) > 0 {
		return joinErrors("invalid definitions", h.definitionErrs)
	}
	for _, key := range h.monitorOrder {
		m := h.monitors[key]
		if err := validateDesired(m); err != nil {
			return fmt.Errorf("monitor %q: %w", monitorName(m), err)
		}
	}
	for _, page := range h.statusPages {
		for key := range page.members {
			if _, ok := h.monitors[key]; !ok {
				return fmt.Errorf("status page %q references undeclared monitor %q", page.name, key)
			}
		}
	}
	return nil
}

func mergeWebsite(defaults, value Website) Website {
	out := value
	if out.ContactList == "" {
		out.ContactList = defaults.ContactList
	}
	if out.Category == "" {
		out.Category = defaults.Category
	}
	if out.Method == "" {
		out.Method = defaults.Method
	}
	if out.Keyword == "" {
		out.Keyword = defaults.Keyword
	}
	if out.AcceptedHTTPStatuses == nil {
		out.AcceptedHTTPStatuses = cloneInts(defaults.AcceptedHTTPStatuses)
	}
	if out.Locations == nil {
		out.Locations = append([]string(nil), defaults.Locations...)
	}
	if out.Timeout == 0 {
		out.Timeout = defaults.Timeout
	}
	if out.Frequency == 0 {
		out.Frequency = defaults.Frequency
	}
	if out.Tries == 0 {
		out.Tries = defaults.Tries
	}
	if out.TriggeringLocations == 0 {
		out.TriggeringLocations = defaults.TriggeringLocations
	}
	if out.AlertAfter == 0 {
		out.AlertAfter = defaults.AlertAfter
	}
	if out.RepeatTimes == 0 {
		out.RepeatTimes = defaults.RepeatTimes
	}
	if out.RepeatEvery == 0 {
		out.RepeatEvery = defaults.RepeatEvery
	}
	if out.MaxRedirects == 0 {
		out.MaxRedirects = defaults.MaxRedirects
	}
	out.Public = out.Public || defaults.Public
	out.ShowTarget = out.ShowTarget || defaults.ShowTarget
	out.DisableSSLCertificateVerification = out.DisableSSLCertificateVerification || defaults.DisableSSLCertificateVerification
	out.DisableSSLHostnameVerification = out.DisableSSLHostnameVerification || defaults.DisableSSLHostnameVerification
	out.Name = strings.TrimSpace(out.Name)
	out.Target = strings.TrimSpace(out.Target)
	out.ContactList = strings.TrimSpace(out.ContactList)
	out.Category = strings.TrimSpace(out.Category)
	out.Method = strings.ToUpper(strings.TrimSpace(out.Method))
	return out
}

func mergeCron(defaults, value Cron) Cron {
	out := value
	if out.ContactList == "" {
		out.ContactList = defaults.ContactList
	}
	if out.Category == "" {
		out.Category = defaults.Category
	}
	if out.Interval == 0 {
		out.Interval = defaults.Interval
	}
	if out.Grace == 0 {
		out.Grace = defaults.Grace
	}
	if out.AlertAfter == 0 {
		out.AlertAfter = defaults.AlertAfter
	}
	if out.RepeatTimes == 0 {
		out.RepeatTimes = defaults.RepeatTimes
	}
	if out.RepeatEvery == 0 {
		out.RepeatEvery = defaults.RepeatEvery
	}
	out.Public = out.Public || defaults.Public
	out.ShowTarget = out.ShowTarget || defaults.ShowTarget
	out.Name = strings.TrimSpace(out.Name)
	out.ContactList = strings.TrimSpace(out.ContactList)
	out.Category = strings.TrimSpace(out.Category)
	return out
}

func validateDesired(m desiredMonitor) error {
	switch m.kind {
	case WebsiteMonitor:
		w := m.website
		if strings.TrimSpace(w.Name) == "" {
			return fmt.Errorf("name is required")
		}
		if strings.TrimSpace(w.Target) == "" {
			return fmt.Errorf("target is required")
		}
		u, err := url.Parse(w.Target)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("target must be an absolute HTTP(S) URL")
		}
		if strings.TrimSpace(w.ContactList) == "" {
			return fmt.Errorf("contact list is required")
		}
		if w.Method != "GET" && w.Method != "HEAD" {
			return fmt.Errorf("method must be GET or HEAD")
		}
		if len(w.Locations) == 0 {
			return fmt.Errorf("at least one location is required")
		}
		seenLocations := make(map[string]bool)
		for _, location := range w.Locations {
			if !supportedLocations[location] {
				return fmt.Errorf("unsupported location %q", location)
			}
			if seenLocations[location] {
				return fmt.Errorf("duplicate location %q", location)
			}
			seenLocations[location] = true
		}
		if w.TriggeringLocations > len(w.Locations) {
			return fmt.Errorf("triggering locations cannot exceed configured locations")
		}
		if len(w.AcceptedHTTPStatuses) == 0 {
			return fmt.Errorf("at least one accepted HTTP status is required")
		}
		for _, status := range w.AcceptedHTTPStatuses {
			if status < 100 || status > 599 {
				return fmt.Errorf("invalid accepted HTTP status %d", status)
			}
		}
		if err := wholeSeconds("timeout", w.Timeout); err != nil {
			return err
		}
		if err := wholeMinutes("frequency", w.Frequency); err != nil {
			return err
		}
		if err := nonnegativeMinutes("alert after", w.AlertAfter); err != nil {
			return err
		}
		if err := wholeMinutes("repeat every", w.RepeatEvery); err != nil {
			return err
		}
		if w.Tries < 1 || w.TriggeringLocations < 1 || w.RepeatTimes < 0 || w.MaxRedirects < 1 {
			return fmt.Errorf("invalid numeric settings")
		}
	case CronMonitor:
		c := m.cron
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("name is required")
		}
		if strings.TrimSpace(c.ContactList) == "" {
			return fmt.Errorf("contact list is required")
		}
		if err := wholeSeconds("interval", c.Interval); err != nil {
			return err
		}
		if err := nonnegativeSeconds("grace", c.Grace); err != nil {
			return err
		}
		if err := nonnegativeMinutes("alert after", c.AlertAfter); err != nil {
			return err
		}
		if c.RepeatTimes < 0 {
			return fmt.Errorf("repeat times must not be negative")
		}
		if c.RepeatEvery != 0 {
			if err := wholeMinutes("repeat every", c.RepeatEvery); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported monitor type %q", m.kind)
	}
	return nil
}

func wholeSeconds(name string, d time.Duration) error {
	if d <= 0 || d%time.Second != 0 {
		return fmt.Errorf("%s must be a positive whole number of seconds", name)
	}
	return nil
}
func nonnegativeSeconds(name string, d time.Duration) error {
	if d < 0 || d%time.Second != 0 {
		return fmt.Errorf("%s must be a non-negative whole number of seconds", name)
	}
	return nil
}
func wholeMinutes(name string, d time.Duration) error {
	if d <= 0 || d%time.Minute != 0 {
		return fmt.Errorf("%s must be a positive whole number of minutes", name)
	}
	return nil
}
func nonnegativeMinutes(name string, d time.Duration) error {
	if d < 0 || d%time.Minute != 0 {
		return fmt.Errorf("%s must be a non-negative whole number of minutes", name)
	}
	return nil
}

var supportedLocations = map[string]bool{
	"new_york": true, "san_francisco": true, "dallas": true,
	"amsterdam": true, "london": true, "frankfurt": true,
	"singapore": true, "sydney": true, "sao_paulo": true,
	"tokyo": true, "mumbai": true, "warsaw": true,
}

func monitorKey(kind MonitorKind, name string) string {
	return string(kind) + "\x00" + strings.TrimSpace(name)
}
func monitorName(m desiredMonitor) string {
	if m.kind == WebsiteMonitor {
		return m.website.Name
	}
	return m.cron.Name
}
func cloneInts(in []int) []int { return append([]int(nil), in...) }
func joinErrors(prefix string, errs []error) error {
	parts := make([]string, len(errs))
	for i, err := range errs {
		parts[i] = err.Error()
	}
	sort.Strings(parts)
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}
