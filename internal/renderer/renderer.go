package renderer

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	barcodeGen "github.com/ostretsov/labelsrv/internal/barcode"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
	"github.com/ostretsov/labelsrv/internal/visibility"
)

//go:embed fonts
var fontsFS embed.FS

// fontVariant holds the embedded bytes for one TTF file.
type fontVariant struct {
	style string
	file  string
}

// freeSansVariants maps gofpdf style strings to embedded font filenames.
var freeSansVariants = []fontVariant{
	{"", "fonts/FreeSans.ttf"},
	{"B", "fonts/FreeSansBold.ttf"},
	{"I", "fonts/FreeSansOblique.ttf"},
	{"BI", "fonts/FreeSansBoldOblique.ttf"},
}

// registerFreeSans registers all FreeSans variants as a UTF-8 font family in pdf.
func registerFreeSans(pdf *gofpdf.Fpdf) error {
	for _, v := range freeSansVariants {
		data, err := fontsFS.ReadFile(v.file)
		if err != nil {
			return fmt.Errorf("reading embedded font %q: %w", v.file, err)
		}

		pdf.AddUTF8FontFromBytes("FreeSans", v.style, data)

		if err := pdf.Error(); err != nil {
			return fmt.Errorf("registering font style %q: %w", v.style, err)
		}
	}

	return nil
}

// extraFont holds a TTF font loaded from disk, ready to register per PDF.
type extraFont struct {
	family string
	style  string
	data   []byte
}

// Renderer renders label templates to PDF.
type Renderer struct {
	extraFonts []extraFont
}

// New creates a new Renderer. If fontsDir is non-empty, all TTF files found
// there are loaded and made available via font_family in templates.
// Naming convention: "Roboto-Bold.ttf" → family "Roboto", style "B".
func New(fontsDir string) (*Renderer, error) {
	r := &Renderer{}

	if fontsDir != "" {
		if err := r.loadFonts(fontsDir); err != nil {
			return nil, fmt.Errorf("loading fonts from %q: %w", fontsDir, err)
		}
	}

	return r, nil
}

// loadFonts reads all TTF files from dir into r.extraFonts.
func (r *Renderer) loadFonts(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("reading fonts directory: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || strings.ToLower(filepath.Ext(e.Name())) != ".ttf" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // fonts dir is trusted
		if err != nil {
			return fmt.Errorf("reading font %q: %w", e.Name(), err)
		}

		family, style := parseFontFilename(e.Name())
		r.extraFonts = append(r.extraFonts, extraFont{family: family, style: style, data: data})
	}

	return nil
}

// parseFontFilename derives a gofpdf family name and style from a TTF filename.
// Examples: "Roboto-Bold.ttf" → ("Roboto","B"), "OpenSans.ttf" → ("OpenSans","").
func parseFontFilename(name string) (family, style string) {
	base := strings.TrimSuffix(name, filepath.Ext(name))

	if idx := strings.LastIndex(base, "-"); idx != -1 {
		switch strings.ToLower(base[idx+1:]) {
		case "regular":
			return base[:idx], ""
		case "bold":
			return base[:idx], "B"
		case "italic", "oblique":
			return base[:idx], "I"
		case "bolditalic", "boldoblique", "semibolditalic":
			return base[:idx], "BI"
		}
	}

	return base, ""
}

// registerExtraFonts registers all fonts loaded from the fonts directory into pdf.
func (r *Renderer) registerExtraFonts(pdf *gofpdf.Fpdf) error {
	for _, f := range r.extraFonts {
		pdf.AddUTF8FontFromBytes(f.family, f.style, f.data)

		if err := pdf.Error(); err != nil {
			return fmt.Errorf("registering font %q %q: %w", f.family, f.style, err)
		}
	}

	return nil
}

// Render renders a template with input data to PDF bytes.
func (r *Renderer) Render(t *tmpl.Template, data map[string]any) ([]byte, error) {
	widthMM, err := tmpl.ParseSize(t.Size.Width)
	if err != nil {
		return nil, fmt.Errorf("parsing template width: %w", err)
	}

	heightMM, err := tmpl.ParseSize(t.Size.Height)
	if err != nil {
		return nil, fmt.Errorf("parsing template height: %w", err)
	}

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size: gofpdf.SizeType{
			Wd: widthMM,
			Ht: heightMM,
		},
	})
	pdf.AddPage()
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	if err := registerFreeSans(pdf); err != nil {
		return nil, fmt.Errorf("registering fonts: %w", err)
	}

	if err := r.registerExtraFonts(pdf); err != nil {
		return nil, err
	}

	for _, el := range t.Layout {
		if el.VisibleIf != "" {
			visible, err := visibility.Evaluate(el.VisibleIf, data)
			if err != nil {
				return nil, fmt.Errorf("evaluating visible_if for element %q: %w", el.ID, err)
			}

			if !visible {
				continue
			}
		}

		value, err := resolveValue(el, t, data)
		if err != nil {
			return nil, fmt.Errorf("resolving value for element %q: %w", el.ID, err)
		}

		switch el.Type {
		case "text":
			if err := renderText(pdf, el, value); err != nil {
				return nil, fmt.Errorf("rendering text element %q: %w", el.ID, err)
			}
		case "barcode":
			if err := renderBarcode(pdf, el, value); err != nil {
				return nil, fmt.Errorf("rendering barcode element %q: %w", el.ID, err)
			}
		case "image":
			if err := renderImage(pdf, el, value); err != nil {
				return nil, fmt.Errorf("rendering image element %q: %w", el.ID, err)
			}
		case "textbox":
			if err := renderTextBox(pdf, el, value); err != nil {
				return nil, fmt.Errorf("rendering textbox element %q: %w", el.ID, err)
			}
		case "line":
			renderLine(pdf, el)
		case "rect":
			renderRect(pdf, el)
		default:
			return nil, fmt.Errorf("unknown element type %q for element %q", el.Type, el.ID)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF output: %w", err)
	}

	return buf.Bytes(), nil
}

// resolveValue resolves the display value for a layout element.
func resolveValue(el tmpl.LayoutElement, t *tmpl.Template, data map[string]any) (string, error) {
	switch el.Source {
	case "input":
		val, ok := data[el.Key]
		if !ok {
			return "", nil
		}

		return fmt.Sprintf("%v", val), nil
	case "constant":
		c, ok := t.Constants[el.Key]
		if !ok {
			return "", fmt.Errorf("constant %q not found", el.Key)
		}

		return c.Value, nil
	default:
		return el.Value, nil
	}
}

// parseHexColor parses a "#RRGGBB" or "RRGGBB" hex color string into RGB components.
// Returns (0, 0, 0) for an empty or invalid string.
func parseHexColor(s string) (r, g, b int) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) != 6 {
		return 0, 0, 0
	}

	rv, err1 := strconv.ParseInt(s[0:2], 16, 32)
	gv, err2 := strconv.ParseInt(s[2:4], 16, 32)
	bv, err3 := strconv.ParseInt(s[4:6], 16, 32)

	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0
	}

	return int(rv), int(gv), int(bv)
}

// gofpdfStyle maps a FontStyle string to a gofpdf style string.
func gofpdfStyle(fontStyle string) string {
	switch strings.ToLower(strings.ReplaceAll(fontStyle, " ", "")) {
	case "bold", "b":
		return "B"
	case "italic", "cursive", "i":
		return "I"
	case "bold-italic", "bolditalic", "italic-bold", "bi", "ib":
		return "BI"
	default:
		return ""
	}
}

// renderText renders a text element.
func renderText(pdf *gofpdf.Fpdf, el tmpl.LayoutElement, value string) error {
	fontSize := el.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}

	// FreeSans is the default because it supports Unicode (including Cyrillic).
	// Fall back to Arial only when an explicit non-FreeSans family is requested.
	fontFamily := el.FontFamily
	if fontFamily == "" {
		fontFamily = "FreeSans"
	}

	pdf.SetFont(fontFamily, gofpdfStyle(el.FontStyle), fontSize)

	if el.Color != "" {
		r, g, b := parseHexColor(el.Color)
		pdf.SetTextColor(r, g, b)
	}

	lineHeight := fontSize * 0.352778 * 1.2

	pdf.SetXY(el.X, el.Y)

	if el.MaxWidth > 0 {
		pdf.MultiCell(el.MaxWidth, lineHeight, value, "", "L", false)
	} else {
		pdf.Cell(0, lineHeight, value)
	}

	pdf.SetTextColor(0, 0, 0)

	return nil
}

// renderLine renders a line element from (X, Y) to (X2, Y2).
func renderLine(pdf *gofpdf.Fpdf, el tmpl.LayoutElement) {
	lw := el.LineWidth
	if lw <= 0 {
		lw = 0.3
	}

	pdf.SetLineWidth(lw)

	if el.Color != "" {
		r, g, b := parseHexColor(el.Color)
		pdf.SetDrawColor(r, g, b)
	}

	pdf.Line(el.X, el.Y, el.X2, el.Y2)

	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
}

// renderRect renders a rectangle element.
func renderRect(pdf *gofpdf.Fpdf, el tmpl.LayoutElement) {
	lw := el.LineWidth
	if lw <= 0 {
		lw = 0.3
	}

	pdf.SetLineWidth(lw)

	hasFill := el.FillColor != ""
	hasStroke := el.Color != "" || !hasFill

	if el.Color != "" {
		r, g, b := parseHexColor(el.Color)
		pdf.SetDrawColor(r, g, b)
	}

	if hasFill {
		r, g, b := parseHexColor(el.FillColor)
		pdf.SetFillColor(r, g, b)
	}

	var style string

	switch {
	case hasFill && hasStroke:
		style = "FD"
	case hasFill:
		style = "F"
	default:
		style = "D"
	}

	pdf.Rect(el.X, el.Y, el.Width, el.Height, style)

	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetFillColor(255, 255, 255)
}

// renderTextBox renders a textbox element: an optional rectangle container with wrapped text inside.
// Set border_color for a visible border; omit it for an invisible (debug-layout) box.
func renderTextBox(pdf *gofpdf.Fpdf, el tmpl.LayoutElement, value string) error {
	hasFill := el.FillColor != ""
	hasStroke := el.BorderColor != ""

	if hasFill || hasStroke {
		lw := el.LineWidth
		if lw <= 0 {
			lw = 0.3
		}

		pdf.SetLineWidth(lw)

		if hasStroke {
			r, g, b := parseHexColor(el.BorderColor)
			pdf.SetDrawColor(r, g, b)
		}

		if hasFill {
			r, g, b := parseHexColor(el.FillColor)
			pdf.SetFillColor(r, g, b)
		}

		var style string

		switch {
		case hasFill && hasStroke:
			style = "FD"
		case hasFill:
			style = "F"
		default:
			style = "D"
		}

		pdf.Rect(el.X, el.Y, el.Width, el.Height, style)
		pdf.SetLineWidth(0.3)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetFillColor(255, 255, 255)
	}

	fontSize := el.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}

	fontFamily := el.FontFamily
	if fontFamily == "" {
		fontFamily = "FreeSans"
	}

	pdf.SetFont(fontFamily, gofpdfStyle(el.FontStyle), fontSize)

	if el.TextColor != "" {
		r, g, b := parseHexColor(el.TextColor)
		pdf.SetTextColor(r, g, b)
	}

	lineHeight := fontSize * 0.352778 * 1.2

	align := strings.ToUpper(el.Align)
	if align == "" {
		align = "L"
	}

	padding := el.Padding

	textWidth := el.Width - 2*padding
	if textWidth <= 0 {
		textWidth = el.Width
	}

	if el.Clip {
		pdf.ClipRect(el.X, el.Y, el.Width, el.Height, false)
	}

	pdf.SetXY(el.X+padding, el.Y+padding)
	pdf.MultiCell(textWidth, lineHeight, value, "", align, false)

	if el.Clip {
		pdf.ClipEnd()
	}

	pdf.SetTextColor(0, 0, 0)

	return nil
}

// renderBarcode renders a barcode element.
func renderBarcode(pdf *gofpdf.Fpdf, el tmpl.LayoutElement, value string) error {
	if value == "" {
		return nil
	}

	barcodeType := el.BarcodeType
	if barcodeType == "" {
		barcodeType = "code128"
	}

	const (
		dpi       = 96.0
		mmPerInch = 25.4
	)

	pxWidth := int(el.Width * dpi / mmPerInch)
	pxHeight := int(el.Height * dpi / mmPerInch)

	if pxWidth <= 0 {
		pxWidth = 200
	}

	if pxHeight <= 0 {
		pxHeight = 100
	}

	pngBytes, err := barcodeGen.Generate(barcodeType, value, pxWidth, pxHeight)
	if err != nil {
		return fmt.Errorf("generating barcode: %w", err)
	}

	imageID := fmt.Sprintf("%s_%d", el.ID, time.Now().UnixNano())
	reader := bytes.NewReader(pngBytes)
	pdf.RegisterImageOptionsReader(imageID, gofpdf.ImageOptions{ImageType: "PNG"}, reader)
	pdf.ImageOptions(imageID, el.X, el.Y, el.Width, el.Height, false, gofpdf.ImageOptions{}, 0, "")

	if el.BorderColor != "" {
		lw := el.LineWidth
		if lw <= 0 {
			lw = 0.3
		}

		r, g, b := parseHexColor(el.BorderColor)

		pdf.SetLineWidth(lw)
		pdf.SetDrawColor(r, g, b)
		pdf.Rect(el.X, el.Y, el.Width, el.Height, "D")
		pdf.SetLineWidth(0.3)
		pdf.SetDrawColor(0, 0, 0)
	}

	return nil
}

// renderImage renders an image element.
func renderImage(pdf *gofpdf.Fpdf, el tmpl.LayoutElement, value string) error {
	src := el.Src
	if src == "" {
		src = value
	}

	if src == "" {
		return nil
	}

	opts := gofpdf.ImageOptions{}
	pdf.ImageOptions(src, el.X, el.Y, el.Width, el.Height, false, opts, 0, "")

	if el.BorderColor != "" {
		lw := el.LineWidth
		if lw <= 0 {
			lw = 0.3
		}

		r, g, b := parseHexColor(el.BorderColor)

		pdf.SetLineWidth(lw)
		pdf.SetDrawColor(r, g, b)
		pdf.Rect(el.X, el.Y, el.Width, el.Height, "D")
		pdf.SetLineWidth(0.3)
		pdf.SetDrawColor(0, 0, 0)
	}

	return nil
}
