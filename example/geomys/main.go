package main

import (
	"slices"
	"strconv"
	"strings"
	"time"

	ht "github.com/filippo-claude/hetrixtools-sync"
)

func main() {
	ht.Main(definitions)
}

func definitions(h *ht.Hetrix) {
	h.WebsiteDefaults(ht.Website{
		Locations:            []string{"new_york", "san_francisco", "amsterdam", "london", "tokyo"},
		Method:               "GET",
		AcceptedHTTPStatuses: []int{200},
		Timeout:              10 * time.Second,
		Frequency:            time.Minute,
		Tries:                3,
		TriggeringLocations:  3,
		RepeatTimes:          3,
		RepeatEvery:          20 * time.Minute,
		MaxRedirects:         5,
	})
	h.CronDefaults(ht.Cron{Interval: 15 * time.Minute})

	h.IgnoreExisting(func(m ht.ExistingMonitor) bool {
		return !slices.Contains(m.ContactLists, "CT log") &&
			!slices.Contains(m.ContactLists, "CT staging")
	})

	status := h.StatusPage(ht.StatusPage{Name: "Geomys CT Logs"})
	type environment struct {
		contactList string
		suffix      string
		public      bool
		status      *ht.StatusPageRef
	}
	production := environment{contactList: "CT log", public: true, status: status}
	staging := environment{contactList: "CT staging", suffix: " staging"}

	website := func(env environment, m ht.Website) {
		m.ContactList = env.contactList
		m.Category += env.suffix
		m.Public = env.public
		ref := h.Website(m)
		if env.status != nil {
			env.status.Add(ref)
		}
	}
	cron := func(env environment, m ht.Cron) {
		m.ContactList = env.contactList
		m.Public = env.public
		ref := h.Cron(m)
		if env.status != nil {
			env.status.Add(ref)
		}
	}

	for _, check := range []struct {
		env        environment
		site       string
		service    string
		alertAfter time.Duration
	}{
		{production, "tuscolo", "sunlight", 0},
		{production, "tuscolo", "skylight", 0},
		{production, "trastevere", "sunlight", 3 * time.Minute},
		{production, "trastevere", "skylight", 3 * time.Minute},
		{staging, "navigli", "sunlight", 0},
		{staging, "loreto", "sunlight", 0},
	} {
		host := check.site + "." + check.service + ".geomys.org"
		website(check.env, ht.Website{
			Name:       host,
			Target:     "https://" + host + "/health",
			Category:   title(check.service),
			AlertAfter: check.alertAfter,
		})
	}

	for _, deployment := range []struct {
		env  environment
		site string
	}{
		{production, "tuscolo"},
		{staging, "navigli"},
	} {
		for _, service := range []string{"sunlight", "skylight"} {
			for year := 2026; year <= 2028; year++ {
				for half := 1; half <= 2; half++ {
					// The Sunlight series starts in 2026 H2; Skylight starts in H1.
					if service == "sunlight" && year == 2026 && half == 1 {
						continue
					}
					period := strconv.Itoa(year) + "h" + strconv.Itoa(half)
					host := deployment.site + period + "." + service + ".geomys.org"
					path, method, keyword := "/checkpoint", "GET", "— "+deployment.site+period+".sunlight.geomys.org"
					if service == "sunlight" {
						path, method, keyword = "/ct/v1/get-roots", "HEAD", ""
					}
					website(deployment.env, ht.Website{
						Name: host, Target: "https://" + host + path,
						Category: title(service), Method: method, Keyword: keyword,
					})
				}
			}
		}
	}

	for _, item := range []struct {
		env      environment
		site     string
		category string
	}{
		{production, "tuscolo", "Sunlight"},
		{production, "trastevere", "Sunlight"},
		{staging, "navigli", ""},
		{staging, "loreto", ""},
	} {
		cron(item.env, ht.Cron{Name: item.site + " partial-aftersun", Category: item.category})
	}

	website(production, ht.Website{
		Name: "endpoint_uptime_24h.csv", Target: "https://uptime.geomys.org/ct/24h/geomys.org",
		Category: "Skylight", AlertAfter: 15 * time.Minute, TriggeringLocations: 5,
	})
	website(production, ht.Website{
		Name: "tuscolo.skylight.geomys.org logs.json", Target: "https://tuscolo.skylight.geomys.org/logs.json",
		Category: "Skylight", Keyword: "Geomys",
	})
	website(staging, ht.Website{
		Name:     "witness.navigli.sunlight.geomys.org",
		Target:   "https://uptime.geomys.org/witness/add-checkpoint/witness.navigli.sunlight.geomys.org+a3e00fe2+BNy/co4C1Hn1p+INwJrfUlgz7W55dSZReusH/GhUhJ/G",
		Category: "Sunlight", Keyword: "witness signature valid", AlertAfter: 15 * time.Minute, TriggeringLocations: 5,
	})

	// The add-pre-chain checks are a Sunlight series that was only started
	// manually with loreto2026h2.
	for year := 2026; year <= 2028; year++ {
		for half := 1; half <= 2; half++ {
			if year == 2026 && half == 1 {
				continue
			}
			period := strconv.Itoa(year) + "h" + strconv.Itoa(half)
			host := "loreto" + period + ".sunlight.geomys.org"
			website(staging, ht.Website{
				Name:     host + " add-pre-chain",
				Target:   "https://uptime.geomys.org/ct/add-pre-chain/" + host,
				Category: "Sunlight", Timeout: 15 * time.Second, Tries: 2,
			})
		}
	}
}

func title(s string) string { return strings.ToUpper(s[:1]) + s[1:] }
