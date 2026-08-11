package hetrixtools

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	api "github.com/filippo-claude/hetrixtools-sync/internal/hetrixapi"
)

// Main runs the preview/push CLI for a set of ordinary Go definitions.
//
// A typical program is:
//
//	func main() { hetrixtools.Main(definitions) }
func Main(definitions func(*Hetrix)) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := Run(ctx, os.Args, os.Stdout, os.Stderr, definitions); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Run executes the CLI with explicit process inputs. Main is the convenient
// wrapper for normal command programs.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, definitions func(*Hetrix)) error {
	return runCLI(ctx, args, stdout, stderr, definitions, defaultAPIClient())
}

func defaultAPIClient() apiClient {
	token := os.Getenv("HETRIXTOOLS_API_TOKEN")
	baseURL := os.Getenv("HETRIXTOOLS_BASE_URL")
	if baseURL != "" {
		return api.NewClientWithBaseURL(baseURL, token)
	}
	return api.NewClient(token)
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer, definitions func(*Hetrix), client apiClient) error {
	program := "hetrixtools"
	if len(args) > 0 && args[0] != "" {
		program = filepath.Base(args[0])
	}
	if len(args) != 2 || (args[1] != "preview" && args[1] != "push") {
		fmt.Fprintf(stderr, "usage: %s preview|push\n", program)
		return fmt.Errorf("expected preview or push")
	}
	if definitions == nil {
		return fmt.Errorf("definitions function is nil")
	}
	h := newHetrix()
	definitions(h)
	p, err := buildPlan(ctx, h, client)
	if err != nil {
		return err
	}
	printPlan(stdout, p)
	if args[1] == "preview" || len(p.operations) == 0 {
		return nil
	}
	return executePlan(ctx, stdout, client, p)
}

func printPlan(w io.Writer, p *plan) {
	for _, warning := range p.warnings {
		fmt.Fprintf(w, "! %s\n", warning)
	}
	if len(p.operations) == 0 {
		fmt.Fprintln(w, "No changes.")
		return
	}
	for _, op := range p.operations {
		switch op.kind {
		case opCreate:
			fmt.Fprintf(w, "+ %s %s\n", op.desired.kind, op.name)
		case opUpdate:
			fmt.Fprintf(w, "~ %s %s\n", op.desired.kind, op.name)
			for _, diff := range op.diffs {
				fmt.Fprintf(w, "    %s: %s -> %s\n", diff.name, diff.old, diff.new)
			}
		case opPageRemove:
			fmt.Fprintf(w, "- status-page %s: %s\n", op.pageName, op.name)
		case opPageAdd:
			fmt.Fprintf(w, "+ status-page %s: %s\n", op.pageName, op.name)
		case opDelete:
			kind := MonitorKind("monitor")
			if op.actual != nil {
				kind = op.actual.kind
			}
			fmt.Fprintf(w, "- %s %s\n", kind, op.name)
		default:
			// Push executes exactly the operations printed here, so an
			// operation this switch cannot describe must never reach
			// executePlan.
			panic(fmt.Sprintf("plan contains unprintable operation kind %d", op.kind))
		}
	}
	fmt.Fprintf(w, "\n%d change(s).\n", len(p.operations))
}

func executePlan(ctx context.Context, w io.Writer, client apiClient, p *plan) error {
	created := make(map[string]string)
	for _, op := range p.operations {
		switch op.kind {
		case opCreate:
			result, err := client.CreateUptimeMonitor(ctx, requestForCreate(op.desired, op.contactID))
			if err != nil {
				return fmt.Errorf("create %s %q: %w", op.desired.kind, op.name, err)
			}
			if strings.TrimSpace(result.MonitorID) == "" {
				return fmt.Errorf("create %s %q returned no monitor ID", op.desired.kind, op.name)
			}
			created[op.key] = result.MonitorID
			if op.desired.kind == CronMonitor {
				fmt.Fprintf(w, "created cron %s: monitor_id=%s", op.name, result.MonitorID)
				if result.ServerID != "" {
					fmt.Fprintf(w, " server_id=%s", result.ServerID)
				}
				fmt.Fprintln(w)
			}
		case opUpdate:
			actual := &op.actual.raw
			if _, err := client.UpdateUptimeMonitor(ctx, requestForUpdate(op.desired, op.id, op.contactID, actual)); err != nil {
				return fmt.Errorf("update %s %q: %w", op.desired.kind, op.name, err)
			}
		case opPageRemove:
			if err := client.RemoveStatusPageMonitors(ctx, op.pageID, []string{op.memberID}); err != nil {
				return fmt.Errorf("remove %q from status page %q: %w", op.name, op.pageName, err)
			}
		case opPageAdd:
			id := op.memberID
			if id == "" {
				id = created[op.key]
			}
			if id == "" {
				return fmt.Errorf("add %q to status page %q: monitor ID is unknown", op.name, op.pageName)
			}
			if err := client.AddStatusPageMonitors(ctx, op.pageID, []string{id}); err != nil {
				return fmt.Errorf("add %q to status page %q: %w", op.name, op.pageName, err)
			}
		case opDelete:
			if err := client.DeleteUptimeMonitor(ctx, op.id); err != nil {
				return fmt.Errorf("delete %s %q: %w", op.actual.kind, op.name, err)
			}
		}
	}
	fmt.Fprintln(w, "Push complete.")
	return nil
}
