// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

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
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
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

type renderOptions struct {
	profileID string
	request   string
	cbom      string
	scanJSON  string
	pdf       string
	result    string
	verbose   bool
	logFormat string
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		rootUsage()
		return 0
	}
	if args[0] == "help" {
		return helpCommand(args[1:])
	}
	if isVersionCommand(args) {
		fmt.Printf("breachsafe-pdf %s\n", version)
		return 0
	}
	registries, err := newRegistries()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	switch args[0] {
	case "render":
		return render(args[1:], registries)
	case "profile":
		return profileCommand(args[1:], registries)
	default:
		rootUsage()
		return 2
	}
}

func helpCommand(args []string) int {
	if len(args) == 0 {
		rootUsage()
		return 0
	}
	if len(args) == 1 && args[0] == "profile" {
		profileUsage()
		return 0
	}
	if len(args) == 1 && args[0] == "render" {
		renderUsage()
		return 0
	}
	if len(args) == 2 && args[0] == "profile" && args[1] == "render" {
		renderUsage()
		return 0
	}
	rootUsage()
	return 2
}

func profileCommand(args []string, registries registries) int {
	if len(args) == 0 || wantsHelp(args[0]) {
		profileUsage()
		return 0
	}
	switch args[0] {
	case "list":
		return listProfiles(args[1:], registries.reports)
	case "inspect":
		return inspectProfile(args[1:], registries.reports)
	case "render":
		// Compatibility alias; canonical syntax is `breachsafe-pdf render`.
		return render(args[1:], registries)
	default:
		profileUsage()
		return 2
	}
}

func listProfiles(args []string, registry report.Registry) int {
	if len(args) > 0 {
		if len(args) == 1 && wantsHelp(args[0]) {
			profileUsage()
			return 0
		}
		return 2
	}
	for _, id := range registry.IDs() {
		fmt.Println(id)
	}
	return 0
}

func inspectProfile(args []string, registry report.Registry) int {
	if len(args) == 1 && wantsHelp(args[0]) {
		profileUsage()
		return 0
	}
	if len(args) != 1 {
		profileUsage()
		return 2
	}
	profile, err := registry.Resolve(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fmt.Printf("id=%s\nversion=%s\ninput_adapter=%s\nview=%s\n", profile.ID(), profile.Version(), profile.InputAdapterID(), profile.View())
	return 0
}

func newRegistries() (registries, error) {
	inputs, err := input.NewRegistry(qureddy.Adapter{})
	if err != nil {
		return registries{}, fmt.Errorf("register input adapters: %w", err)
	}
	reports, err := report.NewRegistry(community.Profile{})
	if err != nil {
		return registries{}, fmt.Errorf("register report profiles: %w", err)
	}
	return registries{inputs: inputs, reports: reports}, nil
}

func render(args []string, registries registries) int {
	options, err := parseRenderOptions(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		renderUsage()
		return 2
	}
	logger, err := newLogger(options.logFormat, options.verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	profile, err := registries.reports.Resolve(options.profileID)
	if err != nil {
		logger.Error("profile resolution failed", "profile", options.profileID, "error", err)
		return 2
	}
	adapter, err := registries.inputs.Resolve(profile.InputAdapterID())
	if err != nil {
		logger.Error("input adapter resolution failed", "profile", profile.ID(), "error", err)
		return 2
	}
	return executeRender(options, profile, adapter, logger)
}

func parseRenderOptions(args []string) (renderOptions, error) {
	flags := flag.NewFlagSet("breachsafe-pdf render", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = renderUsage
	options := renderOptions{}
	flags.StringVar(&options.profileID, "profile", "", "report profile identifier")
	flags.StringVar(&options.request, "request", "", "report request JSON")
	flags.StringVar(&options.cbom, "cbom", "", "CycloneDX CBOM JSON")
	flags.StringVar(&options.scanJSON, "scan-json", "", "producer scan JSON")
	flags.StringVar(&options.pdf, "pdf", "", "new PDF output path")
	flags.StringVar(&options.result, "result", "", "new RenderResult output path")
	flags.BoolVar(&options.verbose, "verbose", false, "enable informational diagnostics on stderr")
	flags.StringVar(&options.logFormat, "log-format", "text", "diagnostic format: text or json")
	if err := flags.Parse(args); err != nil {
		return renderOptions{}, err
	}
	if flags.NArg() != 0 || options.profileID == "" {
		return renderOptions{}, fmt.Errorf("profile and options are required")
	}
	return options, nil
}

func executeRender(options renderOptions, profile report.Profile, adapter input.Adapter, logger *slog.Logger) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := evidenceapp.RenderFilesProfile(ctx, evidenceapp.FileRequest{
		RequestPath: options.request, CBOMPath: options.cbom, ScanJSONPath: options.scanJSON,
		PDFPath: options.pdf, ResultPath: options.result,
	}, pdf.New(version, ""), admission.DefaultLimits(), evidenceapp.Build{GeneratorVersion: version}, adapter, profile)
	if err != nil {
		logger.Error("render failed", "profile", profile.ID(), "error", err)
		return fault.ExitCode(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		logger.Error("result output failed", "error", err)
		return 1
	}
	logger.Info("render completed", "profile", profile.ID(), "pages", result.PDF.Pages, "pdf_bytes", result.PDF.Bytes)
	return 0
}

func isVersionCommand(args []string) bool {
	return len(args) == 1 && (args[0] == "version" || args[0] == "--version" || args[0] == "-version")
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
