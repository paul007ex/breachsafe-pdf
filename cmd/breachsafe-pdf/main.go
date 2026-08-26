// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Command breachsafe-pdf renders a BreachSAFE evidence PDF from a scan artifact set.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidenceapp"
	"github.com/paul007ex/breachsafe-pdf/internal/input"
	"github.com/paul007ex/breachsafe-pdf/internal/input/adapters/qureddy"
	"github.com/paul007ex/breachsafe-pdf/internal/pdf"
	"github.com/paul007ex/breachsafe-pdf/internal/report"
	"github.com/paul007ex/breachsafe-pdf/internal/report/profiles/community"
)

var version = "0.1.0"

type registries struct {
	inputs  input.Registry
	reports report.Registry
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		rootUsage()
		return 0
	}
	if args[0] == "help" {
		args = args[1:]
		if len(args) == 0 {
			rootUsage()
			return 0
		}
		if len(args) == 1 && args[0] == "profile" {
			profileUsage()
			return 0
		}
		if len(args) == 2 && args[0] == "profile" && args[1] == "render" {
			renderUsage()
			return 0
		}
		if len(args) == 1 && args[0] == "render" {
			renderUsage()
			return 0
		}
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version" || args[0] == "-version") {
		fmt.Printf("breachsafe-pdf %s\n", version)
		return 0
	}
	if args[0] == "render" {
		if len(args) == 2 && wantsHelp(args[1]) {
			renderUsage()
			return 0
		}
		regs, err := newRegistries()
		if err != nil {
			fmt.Fprintf(os.Stderr, "breachsafe-pdf: %v\n", err)
			return 1
		}
		return render(args[1:], regs)
	}
	if len(args) == 1 && args[0] == "profile" {
		profileUsage()
		return 0
	}
	if len(args) < 2 || args[0] != "profile" {
		rootUsage()
		return 2
	}
	if wantsHelp(args[1]) {
		profileUsage()
		return 0
	}

	regs, err := newRegistries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "breachsafe-pdf: %v\n", err)
		return 1
	}
	switch args[1] {
	case "list":
		if len(args) > 2 && wantsHelp(args[2]) {
			profileUsage()
			return 0
		}
		for _, id := range regs.reports.IDs() {
			fmt.Println(id)
		}
		return 0
	case "inspect":
		if len(args) > 2 && wantsHelp(args[2]) {
			profileUsage()
			return 0
		}
		if len(args) != 3 {
			profileUsage()
			return 2
		}
		profile, err := regs.reports.Resolve(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Printf("id=%s\nversion=%s\ninput_adapter=%s\nview=%s\n", profile.ID(), profile.Version(), profile.InputAdapterID(), profile.View())
		return 0
	case "render": // transition alias; canonical form is `breachsafe-pdf render --profile`.
		if len(args) == 3 && wantsHelp(args[2]) {
			renderUsage()
			return 0
		}
		return render(args[2:], regs)
	default:
		profileUsage()
		return 2
	}
}

func newRegistries() (registries, error) {
	inputs, err := input.NewRegistry(qureddy.Adapter{})
	if err != nil {
		return registries{}, fmt.Errorf("build input registry: %w", err)
	}
	reports, err := report.NewRegistry(community.Profile{})
	if err != nil {
		return registries{}, fmt.Errorf("build report registry: %w", err)
	}
	return registries{inputs: inputs, reports: reports}, nil
}

func render(args []string, regs registries) int {
	flags := flag.NewFlagSet("breachsafe-pdf render", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = renderUsage
	profileID := flags.String("profile", "", "report profile identifier")
	requestPath := flags.String("request", "", "report request JSON")
	cbomPath := flags.String("cbom", "", "CycloneDX CBOM JSON")
	scanPath := flags.String("scan-json", "", "producer scan JSON")
	pdfPath := flags.String("pdf", "", "new PDF output path")
	resultPath := flags.String("result", "", "new RenderResult output path")
	verbose := flags.Bool("verbose", false, "enable informational diagnostics on stderr")
	logFormat := flags.String("log-format", "text", "diagnostic format: text or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *profileID == "" {
		renderUsage()
		return 2
	}
	logger, err := newLogger(*logFormat, *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	profile, err := regs.reports.Resolve(*profileID)
	if err != nil {
		logger.Error("profile resolution failed", "profile", *profileID, "error", err)
		return 2
	}
	adapter, err := regs.inputs.Resolve(profile.InputAdapterID())
	if err != nil {
		logger.Error("input adapter resolution failed", "profile", profile.ID(), "error", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := evidenceapp.RenderFilesProfile(ctx, evidenceapp.FileRequest{
		RequestPath: *requestPath, CBOMPath: *cbomPath, ScanJSONPath: *scanPath,
		PDFPath: *pdfPath, ResultPath: *resultPath,
	}, pdf.New(version, ""), admission.DefaultLimits(), evidenceapp.Build{GeneratorVersion: version}, adapter, profile)
	if err != nil {
		logger.Error("render failed", "profile", profile.ID(), "error", err)
		return 1
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		logger.Error("result output failed", "error", err)
		return 1
	}
	logger.Info("render completed", "profile", profile.ID(), "pages", result.PDF.Pages, "pdf_bytes", result.PDF.Bytes)
	return 0
}

func wantsHelp(value string) bool {
	return value == "-h" || value == "--help" || value == "help"
}

func rootUsage() {
	fmt.Println("usage: breachsafe-pdf <version|render|profile|help>")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  version                         print the CLI version")
	fmt.Println("  render --profile PROFILE ...     render a report")
	fmt.Println("  profile list                    list report profiles")
	fmt.Println("  profile inspect PROFILE         inspect a report profile")
	fmt.Println("  help [profile|render]           show help")
}

func profileUsage() {
	fmt.Println("usage: breachsafe-pdf profile <list|inspect> [PROFILE]")
	fmt.Println()
	fmt.Println("use --help after a command for details")
}

func renderUsage() {
	fmt.Println("usage: breachsafe-pdf render --profile PROFILE OPTIONS")
	fmt.Println()
	fmt.Println("options:")
	fmt.Println("  --profile ID       report profile identifier")
	fmt.Println("  --request PATH     report request JSON")
	fmt.Println("  --cbom PATH        CycloneDX CBOM JSON")
	fmt.Println("  --scan-json PATH   producer scan JSON")
	fmt.Println("  --pdf PATH         new PDF output path")
	fmt.Println("  --result PATH      new RenderResult output path")
	fmt.Println("  --verbose          enable informational diagnostics on stderr")
	fmt.Println("  --log-format FMT   text or json diagnostics (default: text)")
}

func newLogger(format string, verbose bool) (*slog.Logger, error) {
	level := new(slog.LevelVar)
	if verbose {
		level.Set(slog.LevelInfo)
	} else {
		level.Set(slog.LevelError)
	}
	options := &slog.HandlerOptions{Level: level}
	switch format {
	case "text":
		return slog.New(slog.NewTextHandler(os.Stderr, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stderr, options)), nil
	default:
		return nil, fmt.Errorf("unsupported log format %q (want text or json)", format)
	}
}
