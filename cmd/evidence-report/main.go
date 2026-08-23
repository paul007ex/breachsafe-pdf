// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidenceapp"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/pdf"
)

var (
	generatorVersion = "0.1.0"
	generatorCommit  = ""
)

func main() { os.Exit(run(os.Args[1:])) }

func run(arguments []string) int {
	flags := flag.NewFlagSet("evidence-report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.Bool("version", false, "print the report compiler version and exit")
	requestPath := flags.String("request", "", "report identity and bounded render options JSON")
	cbomPath := flags.String("cbom", "", "exact CycloneDX 1.7 CBOM JSON")
	scanPath := flags.String("scan-json", "", "exact qureddy.scan.v1 JSON")
	pdfPath := flags.String("pdf", "", "new PDF output path")
	resultPath := flags.String("result", "", "new RenderResult JSON output path")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if *version {
		fmt.Printf("breachsafe-report-go %s\n", generatorVersion)
		return 0
	}
	if flags.NArg() != 0 || *requestPath == "" || *cbomPath == "" || *scanPath == "" || *pdfPath == "" || *resultPath == "" {
		fmt.Fprintln(os.Stderr, "usage: evidence-report -request REQUEST.json -cbom CBOM.json -scan-json SCAN.json -pdf REPORT.pdf -result REPORT.result.json")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	build := evidenceapp.Build{GeneratorVersion: generatorVersion, GeneratorCommit: generatorCommit}
	result, err := evidenceapp.RenderFiles(ctx, evidenceapp.FileRequest{
		RequestPath: *requestPath, CBOMPath: *cbomPath, ScanJSONPath: *scanPath,
		PDFPath: *pdfPath, ResultPath: *resultPath,
	}, pdf.New(build.GeneratorVersion, build.GeneratorCommit), admission.DefaultLimits(), build)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return fault.ExitCode(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, fault.Wrap(fault.CodeWriteFailed, "cli.stdout", err))
		return 1
	}
	return 0
}
