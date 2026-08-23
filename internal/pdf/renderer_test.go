// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

package pdf_test

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/paul007ex/breachsafe-pdf/internal/admission"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"github.com/paul007ex/breachsafe-pdf/internal/pdf"
)

func TestFullReportIsDeterministicAndExtractable(t *testing.T) {
	model := admittedModel(t)
	renderer := pdf.New("test", "deadbeef")
	first, err := renderer.Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	second, err := renderer.Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(first.Bytes), "%PDF-") {
		t.Fatal("output lacks PDF header")
	}
	if first.Pages != 9 {
		t.Fatalf("pages = %d, want 9", first.Pages)
	}
	if evidence.DigestBytes(first.Bytes) != evidence.DigestBytes(second.Bytes) {
		t.Fatal("same immutable input produced different PDF bytes")
	}
	if first.RendererName != "codeberg.org/go-pdf/fpdf" || first.RendererVersion != "v0.12.0" {
		t.Fatalf("renderer = %s %s", first.RendererName, first.RendererVersion)
	}
	if len(first.FontBundleSHA256) != 64 || len(first.AssetBundleSHA256) != 64 {
		t.Fatal("font or asset bundle identity is incomplete")
	}

	pdfPath := writePDF(t, first.Bytes)
	text := popplerText(t, pdfPath)
	normalizedText := strings.Join(strings.Fields(text), " ")
	for _, required := range []string{
		"Quantum Readiness Evidence Report", "Executive summary", "Scope, method and tools",
		"Evidence posture axes", "Observed results and findings", "Cryptographic inventory",
		"Gaps, errors and limitations", "Provenance and report integrity", "External integrity receipt",
		"1cc846a80255cb0fa1b9be41cc347c08591106d9a35bdcda5aae7f244f0dcf81",
		"Page 9 of 9",
	} {
		if !strings.Contains(normalizedText, required) {
			t.Fatalf("extracted PDF lacks %q", required)
		}
	}
	for _, excluded := range []string{"/private/producer/path", "s_client", "private output intentionally excluded"} {
		if strings.Contains(text, excluded) {
			t.Fatalf("PDF leaked excluded producer detail %q", excluded)
		}
	}
	info := pdfInfo(t, pdfPath)
	for _, required := range []string{"Pages:           9", "Tagged:          no", "JavaScript:      no", "Encrypted:       no", "Form:            none"} {
		if !strings.Contains(info, required) {
			t.Fatalf("pdfinfo lacks %q:\n%s", required, info)
		}
	}
}

func TestLetterGrayscaleProfileRendersAsLetterAndGray(t *testing.T) {
	pdftocairo, err := exec.LookPath("pdftocairo")
	if err != nil {
		t.Skip("pdftocairo is required for raster color verification")
	}
	model := admittedModel(t)
	model.RenderOptions.PageSize = evidence.PageLetter
	model.RenderOptions.ColorMode = evidence.ColorGrayscale
	document, err := pdf.New("test", "").Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	pdfPath := writePDF(t, document.Bytes)
	info := pdfInfo(t, pdfPath)
	if !strings.Contains(strings.ToLower(info), "(letter)") || !strings.Contains(info, "612 x 792 pts") {
		t.Fatalf("pdfinfo does not report Letter output:\n%s", info)
	}
	prefix := filepath.Join(t.TempDir(), "page")
	output, err := exec.Command(pdftocairo, "-f", "1", "-l", "1", "-singlefile", "-png", "-r", "72", pdfPath, prefix).CombinedOutput()
	if err != nil {
		t.Fatalf("pdftocairo: %v: %s", err, output)
	}
	file, err := os.Open(prefix + ".png")
	if err != nil {
		t.Fatal(err)
	}
	raster, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil || closeErr != nil {
		t.Fatalf("decode=%v close=%v", decodeErr, closeErr)
	}
	red, green, blue, _ := raster.At(raster.Bounds().Dx()/2, 5).RGBA()
	if red>>8 != green>>8 || green>>8 != blue>>8 {
		t.Fatalf("grayscale header pixel = (%d,%d,%d)", red>>8, green>>8, blue>>8)
	}
}

func TestRendererSupportsConcurrentIndependentDocuments(t *testing.T) {
	model := admittedModel(t)
	renderer := pdf.New("concurrency", "")
	want, err := renderer.Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := evidence.DigestBytes(want.Bytes)
	const workers = 16
	start := make(chan struct{})
	errorsChannel := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			document, renderErr := renderer.Render(context.Background(), model)
			if renderErr == nil && (document.Pages != want.Pages || evidence.DigestBytes(document.Bytes) != wantDigest) {
				renderErr = errors.New("concurrent output differs from baseline")
			}
			errorsChannel <- renderErr
		}()
	}
	close(start)
	group.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLongFindingAndManyRowsPaginateWithoutOmission(t *testing.T) {
	model := admittedModel(t)
	template := model.Findings.Items[0]
	model.Findings.Items = make([]evidence.Finding, 0, 60)
	for index := range 60 {
		finding := template
		finding.ID = fmt.Sprintf("pressure-finding-%03d", index)
		finding.Title = fmt.Sprintf("Pressure finding %03d", index)
		finding.Description = strings.Repeat(fmt.Sprintf("bounded-evidence-%03d ", index), 18)
		model.Findings.Items = append(model.Findings.Items, finding)
	}
	model.Findings.Items[0].Description = "LONG-FINDING-START " + strings.Repeat("page-spanning-evidence ", 550) + " LONG-FINDING-END"
	model.Findings.TotalCount = len(model.Findings.Items)
	model.Findings.OmittedCount = 0
	model.Findings.SelectionRule = ""
	model.Findings.AuthoritativeArtifactRef = ""
	document, err := pdf.New("pagination", "").Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	if document.Pages < 20 {
		t.Fatalf("pressure input produced only %d pages", document.Pages)
	}
	text := popplerText(t, writePDF(t, document.Bytes))
	for _, sentinel := range []string{"LONG-FINDING-START", "LONG-FINDING-END"} {
		if !strings.Contains(text, sentinel) {
			t.Fatalf("overheight finding lacks %s", sentinel)
		}
	}
	for index := range 60 {
		if !strings.Contains(text, fmt.Sprintf("pressure-finding-%03d", index)) {
			t.Fatalf("finding %03d omitted", index)
		}
	}
	if strings.Count(text, "Page ") != document.Pages {
		t.Fatalf("page labels = %d, want %d", strings.Count(text, "Page "), document.Pages)
	}
}

func TestOverheightInventoryAndErrorCardsPaginateWithoutOmission(t *testing.T) {
	model := admittedModel(t)
	model.Inventory.Items[0].Primitive = "INVENTORY-CELL-START " + strings.Repeat("primitive ", 20)
	model.Inventory.Items[0].Protocol = strings.Repeat("protocol ", 24)
	model.Inventory.Items[0].ProtocolVersion = strings.Repeat("version ", 24)
	model.Inventory.Items[0].ParameterSet = strings.Repeat("parameter ", 22) + "INVENTORY-CELL-END"
	model.Errors = []evidence.ErrorRecord{{
		ID: "pressure-error", Stage: "observation", SourceRef: "scan-json", Code: "target_scan_failed",
		Description: "ERROR-CARD-START " + strings.Repeat("bounded-error-evidence ", 520) + " ERROR-CARD-END",
		SubjectRef:  model.Subject.ID,
	}}
	document, err := pdf.New("cell-pressure", "").Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	text := popplerText(t, writePDF(t, document.Bytes))
	for _, sentinel := range []string{"INVENTORY-CELL-START", "INVENTORY-CELL-END", "ERROR-CARD-START", "ERROR-CARD-END"} {
		if !strings.Contains(text, sentinel) {
			t.Fatalf("fragmented content lacks %s", sentinel)
		}
	}
	if strings.Count(text, "Page ") != document.Pages {
		t.Fatalf("page labels = %d, want %d", strings.Count(text, "Page "), document.Pages)
	}
}

func TestHardPageLimitFailsClosed(t *testing.T) {
	model := admittedModel(t)
	template := model.Findings.Items[0]
	model.Findings.Items = make([]evidence.Finding, 0, 500)
	for index := range 500 {
		finding := template
		finding.ID = fmt.Sprintf("page-limit-%03d", index)
		finding.Title = fmt.Sprintf("Page limit finding %03d", index)
		finding.Description = strings.Repeat(fmt.Sprintf("bounded page pressure %03d ", index), 450)
		model.Findings.Items = append(model.Findings.Items, finding)
	}
	model.Findings.TotalCount = len(model.Findings.Items)
	model.Findings.OmittedCount = 0
	model.Findings.SelectionRule = ""
	model.Findings.AuthoritativeArtifactRef = ""
	_, err := pdf.New("page-limit", "").Render(context.Background(), model)
	if err == nil || fault.CodeOf(err) != fault.CodeLimitExceeded {
		t.Fatalf("expected LIMIT_EXCEEDED, got %v", err)
	}
	var typed *fault.Error
	if !errors.As(err, &typed) || typed.Op != "pdf.paginate" || typed.Field != "pages" {
		t.Fatalf("unexpected page-limit error: %v", err)
	}
}

func TestMarkupAndPDFControlWordsRemainLiteral(t *testing.T) {
	model := admittedModel(t)
	model.Findings.Items[0].Description = `MARKUP-START <script>alert("x")</script> /JavaScript /OpenAction endobj %%EOF MARKUP-END`
	document, err := pdf.New("inert", "").Render(context.Background(), model)
	if err != nil {
		t.Fatal(err)
	}
	pdfPath := writePDF(t, document.Bytes)
	text := popplerText(t, pdfPath)
	for _, literal := range []string{"MARKUP-START", `<script>alert("x")</script>`, "/JavaScript", "/OpenAction", "endobj", "%%EOF", "MARKUP-END"} {
		if !strings.Contains(text, literal) {
			t.Fatalf("literal %q not extracted", literal)
		}
	}
	if !strings.Contains(pdfInfo(t, pdfPath), "JavaScript:      no") {
		t.Fatal("literal control word activated a PDF JavaScript feature")
	}
}

func TestUnsupportedGlyphAndCancellationFailExplicitly(t *testing.T) {
	model := admittedModel(t)
	model.Identity.Name += " 😀"
	_, err := pdf.New("glyph", "").Render(context.Background(), model)
	if err == nil || fault.CodeOf(err) != fault.CodeInvalidInput {
		t.Fatalf("unsupported glyph error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = pdf.New("cancel", "").Render(ctx, admittedModel(t))
	if err == nil || fault.CodeOf(err) != fault.CodeCanceled {
		t.Fatalf("cancellation error = %v", err)
	}
}

func admittedModel(t testing.TB) evidence.CommunitySingleScan {
	t.Helper()
	request := readFile(t, "../../examples/community-single-scan.request.json")
	cbom := readFile(t, "../admission/testdata/success.cbom.json")
	scan := readFile(t, "../admission/testdata/success.scan.json")
	result, err := admission.Admit(context.Background(), request, cbom, scan, admission.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return result.Model
}

func readFile(t testing.TB, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writePDF(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func popplerText(t *testing.T, path string) string {
	t.Helper()
	tool, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext is required for extraction acceptance tests")
	}
	output, err := exec.Command(tool, "-layout", path, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("pdftotext: %v: %s", err, output)
	}
	return string(output)
}

func pdfInfo(t *testing.T, path string) string {
	t.Helper()
	tool, err := exec.LookPath("pdfinfo")
	if err != nil {
		t.Skip("pdfinfo is required for structure acceptance tests")
	}
	output, err := exec.Command(tool, path).CombinedOutput()
	if err != nil {
		t.Fatalf("pdfinfo: %v: %s", err, output)
	}
	return string(output)
}

func BenchmarkRenderFullCommunityReport(b *testing.B) {
	model := admittedModel(b)
	renderer := pdf.New("benchmark", "")
	b.ReportAllocs()
	for range b.N {
		document, err := renderer.Render(context.Background(), model)
		if err != nil {
			b.Fatal(err)
		}
		if document.Pages < 1 {
			b.Fatal("empty document")
		}
	}
}
