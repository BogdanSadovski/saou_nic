package nlp

import (
	"fmt"
	"strings"
)

// ParsedResume holds the result of parsing a resume document
type ParsedResume struct {
	Text        string
	RawMetadata map[string]string
	WordCount   int
	PageCount   int
}

// ParsePDF extracts text content from a PDF file
func ParsePDF(data []byte) (*ParsedResume, error) {
	// In production, use a PDF parsing library like ledongthuc/pdf or pdfcpu
	// This is a placeholder implementation

	text := extractTextFromPDF(data)

	return &ParsedResume{
		Text: text,
		RawMetadata: map[string]string{
			"format": "pdf",
		},
		WordCount: countWords(text),
		PageCount: estimatePageCount(data),
	}, nil
}

// ParseDOCX extracts text content from a DOCX file
func ParseDOCX(data []byte) (*ParsedResume, error) {
	// In production, use unioffice or similar library
	// This is a placeholder implementation

	text := extractTextFromDOCX(data)

	return &ParsedResume{
		Text: text,
		RawMetadata: map[string]string{
			"format": "docx",
		},
		WordCount: countWords(text),
		PageCount: 1,
	}, nil
}

// ParsePlainText handles plain text resume content
func ParsePlainText(data []byte) (*ParsedResume, error) {
	text := string(data)

	return &ParsedResume{
		Text: strings.TrimSpace(text),
		RawMetadata: map[string]string{
			"format": "text",
		},
		WordCount: countWords(text),
		PageCount: 1,
	}, nil
}

// extractTextFromPDF is a placeholder for actual PDF text extraction
func extractTextFromPDF(data []byte) string {
	// In production, use a proper PDF parsing library
	// For now, return placeholder text
	return "Sample resume text extracted from PDF"
}

// extractTextFromDOCX is a placeholder for actual DOCX text extraction
func extractTextFromDOCX(data []byte) string {
	// In production, use unioffice or similar library
	// For now, return placeholder text
	return "Sample resume text extracted from DOCX"
}

// countWords counts the number of words in text
func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// estimatePageCount estimates page count from file size
func estimatePageCount(data []byte) int {
	// Rough estimate: ~3KB per page for PDFs
	sizeKB := len(data) / 1024
	if sizeKB <= 0 {
		return 1
	}
	pages := sizeKB / 3
	if pages < 1 {
		return 1
	}
	return pages
}

// ValidateContent checks if the parsed content is valid
func (p *ParsedResume) ValidateContent() error {
	if p.Text == "" {
		return fmt.Errorf("resume text is empty")
	}

	if p.WordCount < 10 {
		return fmt.Errorf("resume text too short: %d words", p.WordCount)
	}

	return nil
}

// Sanitize cleans up the extracted text
func (p *ParsedResume) Sanitize() {
	// Remove excessive whitespace
	lines := strings.Split(p.Text, "\n")
	var cleaned []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" || (len(cleaned) > 0 && cleaned[len(cleaned)-1] != "") {
			cleaned = append(cleaned, line)
		}
	}

	p.Text = strings.Join(cleaned, "\n")
	p.WordCount = countWords(p.Text)
}

// GetSections splits the resume text into logical sections
func (p *ParsedResume) GetSections() map[string]string {
	sections := make(map[string]string)

	sectionMarkers := map[string]string{
		"experience":    "experience",
		"education":     "education",
		"skills":        "skills",
		"summary":       "summary",
		"certification": "certifications",
		"languages":     "languages",
		"projects":      "projects",
	}

	lines := strings.Split(p.Text, "\n")
	currentSection := "header"
	var sectionContent strings.Builder

	for _, line := range lines {
		lowerLine := strings.ToLower(strings.TrimSpace(line))
		foundSection := false

		for key, marker := range sectionMarkers {
			if strings.Contains(lowerLine, marker) {
				// Save previous section
				if sectionContent.Len() > 0 {
					sections[currentSection] = strings.TrimSpace(sectionContent.String())
				}
				currentSection = key
				sectionContent.Reset()
				foundSection = true
				break
			}
		}

		if !foundSection {
			if sectionContent.Len() > 0 {
				sectionContent.WriteString("\n")
			}
			sectionContent.WriteString(line)
		}
	}

	// Save last section
	if sectionContent.Len() > 0 {
		sections[currentSection] = strings.TrimSpace(sectionContent.String())
	}

	return sections
}
