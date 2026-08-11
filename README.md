# hetrixtools-sync

Declarative HetrixTools monitor synchronization in ordinary Go.

A program defines website and cron/heartbeat monitors, establishes an explicit
ownership boundary for existing monitors, and calls `hetrixtools.Main`. The
resulting command has two operations inspired by DNSControl. From the consumer
`package main` directory containing your definitions:

```console
go run . preview
go run . push
```

`preview` is a diff. `push` runs the same planner and executes only the
operations in its output. If the plan is empty, push performs no mutation.

This project is an Apache-2.0 fork of the Go client from
`xaf/terraform-provider-hetrixtools`. It is not affiliated with HetrixTools.
It requires Go 1.25 or newer.

## Install

```console
go get github.com/filippo-claude/hetrixtools-sync@v0.2.5
```

`v0.2.0` is the first release of the declarative library and the first release
using this module path; inherited `v0.1.x` tags belong to the upstream provider.

## Definitions

```go
package main

import (
    "slices"
    "time"

    ht "github.com/filippo-claude/hetrixtools-sync"
)

func main() { ht.Main(definitions) }

func definitions(h *ht.Hetrix) {
    h.WebsiteDefaults(ht.Website{
        Locations:                  []string{"new_york", "amsterdam"},
        Method:                     "GET",
        AcceptedHTTPStatuses:       []int{200},
        Timeout:                    10 * time.Second,
        Frequency:                  time.Minute,
        Tries:                      3,
        TriggeringLocations:        2,
        RepeatTimes:                3,
        RepeatEvery:                20 * time.Minute,
        MaxRedirects:               5,
    })
    h.CronDefaults(ht.Cron{Interval: 15 * time.Minute})

    h.IgnoreExisting(func(m ht.ExistingMonitor) bool {
        return !slices.Contains(m.ContactLists, "Production")
    })

    page := h.StatusPage(ht.StatusPage{Name: "Public status"})

    page.Add(h.Website(ht.Website{
        Name:        "API",
        Target:      "https://example.com/health",
        ContactList: "Production",
        Public:      true,
    }))

    page.Add(h.Cron(ht.Cron{
        Name:        "nightly import",
        ContactList: "Production",
        Public:      true,
    }))
}
```

Fields omitted from `Website` and `Cron` inherit type-specific defaults.
Definitions can use normal Go functions, loops, string concatenation, and data
structures; the library contains no deployment- or architecture-specific DSL.

## Commands

Set a HetrixTools API token and run the definitions program:

```console
export HETRIXTOOLS_API_TOKEN=...
go run . preview
go run . push
```

`HETRIXTOOLS_BASE_URL` overrides the API root, primarily for testing and
credential-injecting proxies.

## Safety model

- `IgnoreExisting` is mandatory. Ignored monitors are invisible to planning and
  are never modified or deleted.
- **Every nonignored existing monitor absent from the definitions is planned for
  deletion.** Review preview before push when changing the ownership predicate.
- Monitor deletion is performed only for a `- website ...` or `- cron ...`
  preview line. A `~` line is an in-place update using the existing monitor ID;
  it never deletes and recreates the monitor, so monitor identity and history
  are retained. (`- status-page ...` only removes page membership.)
- Push applies status-page removals and monitor deletions first, then updates,
  creates, and status-page additions. This frees account capacity before new
  monitors are created.
- Desired monitors excluded by `IgnoreExisting` are rejected.
- Names are stable identities within each monitor type.
- Unknown managed monitor types, ambiguous contact lists, multiple contact
  lists, non-active lifecycle states, and incomplete API responses fail closed.
- Identical duplicate monitors are reported and preserved. Conflicting
  duplicates are an error.
- Website updates preserve mutable remote settings not exposed by the public
  definition type, including expiration and nameserver warnings.
- Cron monitors are plain heartbeat/dead-man switches and support create,
  update, and delete. Updates explicitly keep the unused server-agent detail
  sections private; updates are refused if the API reports any of those sections
  as currently public.
- Status pages manage membership of live, managed monitors. Ignored monitors
  and dangling IDs are preserved. Page presentation settings are untouched.
- Push always computes and prints a fresh preview before making API calls. An
  empty preview causes zero mutation calls.
- Push is not transactional. A later API failure can leave earlier displayed
  operations applied; run preview again before retrying.

Defaults are intentionally one-way: a zero field inherits its type default.
Choose defaults whose zero/false overrides you do not need, or set such fields
per monitor instead of in defaults.

## Geomys example

[`example/geomys/main.go`](example/geomys/main.go) contains definitions derived
from the current Geomys CT monitor account. It expands the existing Sunlight,
Skylight, and add-pre-chain series with ordinary Go loops.

On August 11, 2026, its preview against the account contains only five missing
website monitors: four remaining loreto add-pre-chain checks and the missing
navigli 2028 H2 Sunlight check. Existing identical duplicates and dangling
status-page IDs are warnings, not mutations.

Run it with the same credentials as any other definitions program:

```console
HETRIXTOOLS_API_TOKEN=... go run ./example/geomys preview
```

## License

Apache License 2.0. See `LICENSE` and `NOTICE`.

## Development

```console
go test ./...
go vet ./...
```
