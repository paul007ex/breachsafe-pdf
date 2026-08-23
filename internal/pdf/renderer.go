// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package pdf is the thin, direct codeberg FPDF adapter for the fixed
// Community Single-Scan evidence report contract.
package pdf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"codeberg.org/go-pdf/fpdf"
	"github.com/paul007ex/breachsafe-pdf/internal/assets"
	"github.com/paul007ex/breachsafe-pdf/internal/evidence"
	"github.com/paul007ex/breachsafe-pdf/internal/fault"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

const (
	rendererName          = "codeberg.org/go-pdf/fpdf"
	pinnedRendererVersion = "v0.12.0"
	fontFamily            = "BreachSans"
	fontBundleName        = "Go fonts: Go Regular + Go Bold"
	assetBundleName       = "BreachSAFE approved report visuals/v1"
	helmetImageName       = "breachsafe-helmet"
	mediaType             = "application/pdf"
	hardMaxPages          = 500
)

// Renderer creates independent, in-memory FPDF documents. It contains only
// immutable build identity and is safe for concurrent Render calls.
type Renderer struct {
	generatorVersion string
	generatorCommit  string
}

func New(generatorVersion, generatorCommit string) *Renderer {
	return &Renderer{generatorVersion: generatorVersion, generatorCommit: generatorCommit}
}

// Ready reports whether the receiver can render. It is deliberately safe on a
// nil receiver so a typed-nil Renderer cannot cross the interface boundary.
func (renderer *Renderer) Ready() bool { return renderer != nil }

func (renderer *Renderer) Render(ctx context.Context, model evidence.CommunitySingleScan) (evidence.Document, error) {
	if renderer == nil {
		return evidence.Document{}, fault.New(fault.CodeInvalidInput, "pdf.render", "renderer", "renderer is required")
	}
	if err := ctx.Err(); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeCanceled, "pdf.render", err)
	}
	model = evidence.Canonicalize(model)
	if err := evidence.Validate(ctx, model, evidence.DefaultLimits()); err != nil {
		return evidence.Document{}, err
	}
	if err := ensureGlyphCoverage(ctx, model); err != nil {
		return evidence.Document{}, err
	}
	modelDigest, err := evidence.ModelDigest(model)
	if err != nil {
		return evidence.Document{}, err
	}
	icons, err := parseIcons()
	if err != nil {
		return evidence.Document{}, err
	}

	pageSize := "A4"
	if model.RenderOptions.PageSize == evidence.PageLetter {
		pageSize = "Letter"
	}
	pdf := fpdf.New("P", "mm", pageSize, "")
	pdf.SetCatalogSort(true)
	pdf.SetCompression(true)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(14, 23, 14)
	pdf.AddUTF8FontFromBytes(fontFamily, "", goregular.TTF)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", gobold.TTF)
	pdf.RegisterImageOptionsReader(helmetImageName, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(assets.HelmetPNG()))
	if err := pdf.Error(); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeRenderFailed, "pdf.fonts", err)
	}

	metadataTime := model.Identity.GeneratedAt.UTC()
	pdf.SetCreationDate(metadataTime)
	pdf.SetModificationDate(metadataTime)
	pdf.SetTitle(model.Identity.Name, true)
	pdf.SetAuthor("BreachSAFE", true)
	pdf.SetSubject("Evidence-backed Community Single-Scan quantum readiness report", true)
	pdf.SetCreator("breachsafe-report-go "+renderer.generatorVersion+"; "+rendererName+" "+libraryVersion(), true)
	pdf.SetKeywords("BreachSAFE, QuReddy, evidence, CBOM, CycloneDX, quantum readiness", true)
	pdf.AliasNbPages("{nb}")

	layout := newLayout(ctx, pdf, model, modelDigest, renderer, icons)
	layout.registerFurniture()
	if err := layout.compose(); err != nil {
		return evidence.Document{}, err
	}
	if err := pdf.Error(); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeRenderFailed, "pdf.compose", err)
	}
	if err := ctx.Err(); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeCanceled, "pdf.render", err)
	}

	pages := pdf.PageNo()
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeRenderFailed, "pdf.output", err)
	}
	if output.Len() > 100<<20 {
		return evidence.Document{}, fault.Format(fault.CodeLimitExceeded, "pdf.output", "pdf_bytes", "got %d; maximum is %d", output.Len(), 100<<20)
	}
	if err := ctx.Err(); err != nil {
		return evidence.Document{}, fault.Wrap(fault.CodeCanceled, "pdf.render", err)
	}
	return evidence.Document{
		Bytes: output.Bytes(), MediaType: mediaType, Pages: pages,
		RendererName: rendererName, RendererVersion: libraryVersion(), GeneratorVersion: renderer.generatorVersion,
		FontBundleName: fontBundleName, FontBundleSHA256: fontDigest(),
		AssetBundleSHA256: assets.BundleDigest(), Capabilities: capabilityResults(),
	}, nil
}

func parseIcons() (map[assets.IconName]fpdf.SVGBasicType, error) {
	result := make(map[assets.IconName]fpdf.SVGBasicType, 4)
	for _, name := range []assets.IconName{assets.IconShieldCheck, assets.IconScan, assets.IconAlert, assets.IconDownload} {
		data, err := assets.IconSVG(name)
		if err != nil {
			return nil, fault.Wrap(fault.CodeRenderFailed, "pdf.assets", err)
		}
		icon, err := fpdf.SVGBasicParse(data)
		if err != nil {
			return nil, fault.Wrap(fault.CodeRenderFailed, "pdf.assets", err)
		}
		result[name] = icon
	}
	return result, nil
}

func ensureGlyphCoverage(ctx context.Context, model evidence.CommunitySingleScan) error {
	data, err := json.Marshal(model)
	if err != nil {
		return fault.Wrap(fault.CodeInvalidInput, "pdf.font_preflight", err)
	}
	fonts := []struct {
		name string
		data []byte
	}{
		{name: "regular", data: goregular.TTF},
		{name: "bold", data: gobold.TTF},
	}
	for _, fontData := range fonts {
		font, parseErr := sfnt.Parse(fontData.data)
		if parseErr != nil {
			return fault.WrapField(fault.CodeRenderFailed, "pdf.font_preflight", fontData.name, parseErr)
		}
		var buffer sfnt.Buffer
		for index, valueRune := range string(data) {
			if index%2048 == 0 {
				if err := ctx.Err(); err != nil {
					return fault.Wrap(fault.CodeCanceled, "pdf.font_preflight", err)
				}
			}
			glyph, glyphErr := font.GlyphIndex(&buffer, valueRune)
			if glyphErr != nil {
				return fault.WrapField(fault.CodeRenderFailed, "pdf.font_preflight", fontData.name, glyphErr)
			}
			if glyph == 0 {
				return fault.New(fault.CodeInvalidInput, "pdf.font_preflight", "text", fmt.Sprintf("bundled %s font does not support U+%04X", fontData.name, valueRune))
			}
		}
	}
	return nil
}

func libraryVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dependency := range info.Deps {
			if dependency.Path == rendererName && dependency.Version != "" {
				return dependency.Version
			}
		}
	}
	return pinnedRendererVersion
}

func fontDigest() string {
	data := make([]byte, 0, len(goregular.TTF)+len(gobold.TTF)+32)
	data = append(data, []byte("goregular.ttf\x00")...)
	data = append(data, goregular.TTF...)
	data = append(data, []byte("\x00gobold.ttf\x00")...)
	data = append(data, gobold.TTF...)
	return evidence.DigestBytes(data)
}

func capabilityResults() []evidence.CapabilityResult {
	return []evidence.CapabilityResult{
		{Name: "embedded-fonts", Status: "pass", Detail: fontBundleName + " embedded; source text preflighted against both font faces"},
		{Name: "embedded-approved-visual-assets", Status: "pass", Detail: assetBundleName + "; embedded helmet PNG and normalized SVG icons; no caller paths or remote resources"},
		{Name: "exact-byte-determinism", Status: "pass", Detail: "fixed metadata timestamps, sorted PDF catalog, and immutable inputs"},
		{Name: "javascript-forms-attachments", Status: "pass", Detail: "renderer emits none"},
		{Name: "tagged-pdf-accessibility", Status: "not_supported", Detail: "accessibility_profile=none only"},
		{Name: "pdf-a", Status: "not_supported", Detail: "no archival conformance claim"},
		{Name: "complex-script-shaping", Status: "not_supported", Detail: "bounded Go-font repertoire without shaping"},
		{Name: "digital-signature-encryption", Status: "not_supported", Detail: "owned by a future trust and delivery layer"},
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "Not asserted"
	}
	return value.UTC().Format("2006-01-02 15:04:05 UTC")
}
