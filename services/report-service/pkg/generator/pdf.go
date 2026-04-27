package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/config"
	"github.com/jung-kurt/gofpdf"
)

// PDFGenerator handles PDF report generation from templates
type PDFGenerator struct {
	templateDir string
	assetsDir   string
}

// NewPDFGenerator creates a new PDF generator
func NewPDFGenerator(cfg config.GeneratorConfig) (*PDFGenerator, error) {
	// Ensure template directory exists
	if _, err := os.Stat(cfg.TemplateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", cfg.TemplateDir)
	}

	return &PDFGenerator{
		templateDir: cfg.TemplateDir,
		assetsDir:   cfg.AssetsDir,
	}, nil
}

// Generate creates a PDF from a template and data
func (g *PDFGenerator) Generate(templateName string, data map[string]interface{}) ([]byte, error) {
	// Load template
	tmplPath := filepath.Join(g.templateDir, templateName)
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("reading template file: %w", err)
	}

	// Parse template as Go template
	tmpl, err := template.New("pdf").Funcs(templateFuncs()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Execute template to get HTML content
	var htmlContent bytes.Buffer
	if err := tmpl.Execute(&htmlContent, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Generate PDF from HTML content
	pdfData, err := g.renderPDF(htmlContent.String(), data)
	if err != nil {
		return nil, fmt.Errorf("rendering PDF: %w", err)
	}

	return pdfData, nil
}

// renderPDF creates a PDF from HTML content using gofpdf
func (g *PDFGenerator) renderPDF(htmlContent string, data map[string]interface{}) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// Add header with logo
	g.addHeader(pdf, data)

	// Set default font
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(51, 51, 51)

	// Add content (simplified HTML to PDF conversion)
	// In production, you'd use a proper HTML-to-PDF library like wkhtmltopdf or Chrome headless
	_ = extractTextFromHTML(htmlContent)

	// Add title if present
	if title, ok := data["title"].(string); ok && title != "" {
		pdf.SetFont("Helvetica", "B", 18)
		pdf.SetTextColor(0, 51, 102)
		pdf.CellFormat(0, 15, title, "", 1, "L", false, 0, "")
		pdf.Ln(5)
		pdf.SetTextColor(51, 51, 51)
	}

	// Add subtitle if present
	if subtitle, ok := data["subtitle"].(string); ok && subtitle != "" {
		pdf.SetFont("Helvetica", "I", 12)
		pdf.SetTextColor(102, 102, 102)
		pdf.CellFormat(0, 8, subtitle, "", 1, "L", false, 0, "")
		pdf.Ln(5)
		pdf.SetTextColor(51, 51, 51)
	}

	// Add content sections
	sections := g.extractSections(data)
	for _, section := range sections {
		g.addSection(pdf, section)
	}

	// Add footer
	g.addFooter(pdf)

	// Write PDF to buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("outputting PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// addHeader adds a header section to the PDF
func (g *PDFGenerator) addHeader(pdf *gofpdf.Fpdf, data map[string]interface{}) {
	// Try to load logo
	logoPath := filepath.Join(g.assetsDir, "logo.png")
	if _, err := os.Stat(logoPath); err == nil {
		pdf.RegisterImage("logo", logoPath)
		pdf.ImageOptions("logo", 160, 10, 35, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	}

	// Company name
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(0, 51, 102)
	pdf.CellFormat(150, 10, "Assessment Report", "", 0, "L", false, 0, "")
	pdf.Ln(15)

	// Separator line
	pdf.SetDrawColor(0, 102, 153)
	pdf.SetLineWidth(0.5)
	pdf.Line(10, 25, 200, 25)
	pdf.Ln(10)
}

// addFooter adds a footer to the PDF
func (g *PDFGenerator) addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.SetY(-15)
	pdf.CellFormat(0, 10, fmt.Sprintf("Generated on %s | Confidential", time.Now().Format("2006-01-02 15:04:05")), "", 0, "C", false, 0, "")
}

// addSection adds a content section to the PDF
func (g *PDFGenerator) addSection(pdf *gofpdf.Fpdf, section ContentSection) {
	// Section header
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(0, 102, 153)
	pdf.CellFormat(0, 10, section.Title, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Section content
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(51, 51, 51)

	for _, line := range section.Lines {
		// Check if we need a new page
		if pdf.GetY() > 260 {
			pdf.AddPage()
			g.addHeader(pdf, nil)
		}

		if line.IsBold {
			pdf.SetFont("Helvetica", "B", 10)
		} else {
			pdf.SetFont("Helvetica", "", 10)
		}

		// Multi-line cell for wrapping
		pdf.MultiCell(0, 6, line.Text, "", "L", false)
		pdf.Ln(1)
	}

	pdf.Ln(5)
}

// ContentSection represents a section of content in the PDF
type ContentSection struct {
	Title string
	Lines []ContentLine
}

// ContentLine represents a single line of content
type ContentLine struct {
	Text   string
	IsBold bool
}

// extractSections extracts content sections from template data
func (g *PDFGenerator) extractSections(data map[string]interface{}) []ContentSection {
	var sections []ContentSection

	// Extract candidate information
	if candidate, ok := data["candidate"].(map[string]interface{}); ok {
		section := ContentSection{Title: "Candidate Information"}
		if name, ok := candidate["name"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Name: %s", name), IsBold: false})
		}
		if email, ok := candidate["email"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Email: %s", email), IsBold: false})
		}
		if position, ok := candidate["position"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Position: %s", position), IsBold: false})
		}
		sections = append(sections, section)
	}

	// Extract interview details
	if interview, ok := data["interview"].(map[string]interface{}); ok {
		section := ContentSection{Title: "Interview Details"}
		if date, ok := interview["date"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Date: %s", date), IsBold: false})
		}
		if interviewer, ok := interview["interviewer"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Interviewer: %s", interviewer), IsBold: false})
		}
		if duration, ok := interview["duration"].(string); ok {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("Duration: %s", duration), IsBold: false})
		}
		sections = append(sections, section)
	}

	// Extract scores/assessment
	if scores, ok := data["scores"].([]map[string]interface{}); ok {
		section := ContentSection{Title: "Assessment Scores"}
		for _, score := range scores {
			if category, ok := score["category"].(string); ok {
				value := ""
				if v, ok := score["value"].(string); ok {
					value = v
				} else if v, ok := score["value"].(float64); ok {
					value = fmt.Sprintf("%.1f", v)
				}
				section.Lines = append(section.Lines, ContentLine{
					Text:   fmt.Sprintf("%s: %s", category, value),
					IsBold: false,
				})
			}
		}
		sections = append(sections, section)
	}

	// Extract feedback/summary
	if feedback, ok := data["feedback"].(string); ok && feedback != "" {
		section := ContentSection{Title: "Summary & Feedback"}
		section.Lines = append(section.Lines, ContentLine{Text: feedback, IsBold: false})
		sections = append(sections, section)
	}

	// Extract recommendations
	if recommendations, ok := data["recommendations"].([]string); ok {
		section := ContentSection{Title: "Recommendations"}
		for _, rec := range recommendations {
			section.Lines = append(section.Lines, ContentLine{Text: fmt.Sprintf("- %s", rec), IsBold: false})
		}
		sections = append(sections, section)
	}

	return sections
}

// extractTextFromHTML is a simplified HTML to text converter
// In production, use a proper HTML parser
func extractTextFromHTML(html string) string {
	// Remove HTML tags (very simplified)
	text := html
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "<p>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n")
	text = strings.ReplaceAll(text, "<h1>", "\n")
	text = strings.ReplaceAll(text, "</h1>", "\n")
	text = strings.ReplaceAll(text, "<h2>", "\n")
	text = strings.ReplaceAll(text, "</h2>", "\n")
	text = strings.ReplaceAll(text, "<li>", "- ")
	text = strings.ReplaceAll(text, "</li>", "\n")
	// Remove remaining tags
	for strings.Contains(text, "<") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if end > start {
			text = text[:start] + text[end+1:]
		} else {
			break
		}
	}
	return strings.TrimSpace(text)
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"title": strings.ToTitle,
		"date":  func(t interface{}) string { return fmt.Sprintf("%v", t) },
	}
}
