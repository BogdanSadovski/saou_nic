package generator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/config"
)

// DOCXGenerator handles DOCX report generation from templates
type DOCXGenerator struct {
	templateDir string
	assetsDir   string
}

// NewDOCXGenerator creates a new DOCX generator
func NewDOCXGenerator(cfg config.GeneratorConfig) (*DOCXGenerator, error) {
	// Ensure template directory exists
	if _, err := os.Stat(cfg.TemplateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", cfg.TemplateDir)
	}

	return &DOCXGenerator{
		templateDir: cfg.TemplateDir,
		assetsDir:   cfg.AssetsDir,
	}, nil
}

// Generate creates a DOCX from a template and data
func (g *DOCXGenerator) Generate(templateName string, data map[string]interface{}) ([]byte, error) {
	// Load template
	tmplPath := filepath.Join(g.templateDir, templateName)
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("reading template file: %w", err)
	}

	// Parse template
	tmpl, err := template.New("docx").Funcs(templateFuncs()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Execute template
	var docxContent bytes.Buffer
	if err := tmpl.Execute(&docxContent, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Create DOCX package
	docxData, err := g.createDOCXPackage(docxContent.String(), data)
	if err != nil {
		return nil, fmt.Errorf("creating DOCX package: %w", err)
	}

	return docxData, nil
}

// createDOCXPackage creates a valid DOCX file (ZIP with XML content)
func (g *DOCXGenerator) createDOCXPackage(content string, data map[string]interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`
	if err := writeFile(zipWriter, "[Content_Types].xml", contentTypes); err != nil {
		return nil, err
	}

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	if err := writeFile(zipWriter, "_rels/.rels", rels); err != nil {
		return nil, err
	}

	// word/document.xml
	documentXML := g.buildDocumentXML(content, data)
	if err := writeFile(zipWriter, "word/document.xml", documentXML); err != nil {
		return nil, err
	}

	// word/styles.xml
	stylesXML := g.buildStylesXML()
	if err := writeFile(zipWriter, "word/styles.xml", stylesXML); err != nil {
		return nil, err
	}

	// word/_rels/document.xml.rels
	wordRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
	if err := writeFile(zipWriter, "word/_rels/document.xml.rels", wordRels); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// buildDocumentXML creates the main document XML
func (g *DOCXGenerator) buildDocumentXML(content string, data map[string]interface{}) string {
	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>`

	// Add title
	if title, ok := data["title"].(string); ok && title != "" {
		xml += fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Title"/>
        <w:jc w:val="center"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="36"/>
          <w:color w:val="003366"/>
        </w:rPr>
        <w:t xml:space="preserve">%s</w:t>
      </w:r>
    </w:p>`, escapeXML(title))
	}

	// Add subtitle
	if subtitle, ok := data["subtitle"].(string); ok && subtitle != "" {
		xml += fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:jc w:val="center"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:i/>
          <w:sz w:val="24"/>
          <w:color w:val="666666"/>
        </w:rPr>
        <w:t xml:space="preserve">%s</w:t>
      </w:r>
    </w:p>`, escapeXML(subtitle))
	}

	// Add date
	xml += fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:jc w:val="right"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:sz w:val="20"/>
          <w:color w:val="808080"/>
        </w:rPr>
        <w:t xml:space="preserve">Generated: %s</w:t>
      </w:r>
    </w:p>`, time.Now().Format("2006-01-02 15:04:05"))

	// Add separator
	xml += `
    <w:p>
      <w:pPr>
        <w:pBdr>
          <w:bottom w:val="single" w:sz="12" w:space="1" w:color="006699"/>
        </w:pBdr>
      </w:pPr>
    </w:p>`

	// Add content sections
	sections := g.buildContentSections(data)
	xml += sections

	// Add footer
	xml += `
    <w:p>
      <w:pPr>
        <w:jc w:val="center"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:i/>
          <w:sz w:val="16"/>
          <w:color w:val="808080"/>
        </w:rPr>
        <w:t xml:space="preserve">Confidential - For Internal Use Only</w:t>
      </w:r>
    </w:p>`

	xml += `
  </w:body>
</w:document>`

	return xml
}

// buildContentSections creates Word XML paragraphs from data
func (g *DOCXGenerator) buildContentSections(data map[string]interface{}) string {
	var xml string

	// Candidate information
	if candidate, ok := data["candidate"].(map[string]interface{}); ok {
		xml += `
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="26"/>
          <w:color w:val="006699"/>
        </w:rPr>
        <w:t xml:space="preserve">Candidate Information</w:t>
      </w:r>
    </w:p>`

		if name, ok := candidate["name"].(string); ok {
			xml += createParagraph("Name: "+name, false, false)
		}
		if email, ok := candidate["email"].(string); ok {
			xml += createParagraph("Email: "+email, false, false)
		}
		if position, ok := candidate["position"].(string); ok {
			xml += createParagraph("Position: "+position, false, false)
		}
	}

	// Interview details
	if interview, ok := data["interview"].(map[string]interface{}); ok {
		xml += `
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="26"/>
          <w:color w:val="006699"/>
        </w:rPr>
        <w:t xml:space="preserve">Interview Details</w:t>
      </w:r>
    </w:p>`

		if date, ok := interview["date"].(string); ok {
			xml += createParagraph("Date: "+date, false, false)
		}
		if interviewer, ok := interview["interviewer"].(string); ok {
			xml += createParagraph("Interviewer: "+interviewer, false, false)
		}
		if duration, ok := interview["duration"].(string); ok {
			xml += createParagraph("Duration: "+duration, false, false)
		}
	}

	// Assessment scores
	if scores, ok := data["scores"].([]map[string]interface{}); ok {
		xml += `
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="26"/>
          <w:color w:val="006699"/>
        </w:rPr>
        <w:t xml:space="preserve">Assessment Scores</w:t>
      </w:r>
    </w:p>`

		for _, score := range scores {
			if category, ok := score["category"].(string); ok {
				value := ""
				if v, ok := score["value"].(string); ok {
					value = v
				} else if v, ok := score["value"].(float64); ok {
					value = fmt.Sprintf("%.1f", v)
				}
				xml += createParagraph(fmt.Sprintf("%s: %s", category, value), true, false)
			}
		}
	}

	// Feedback
	if feedback, ok := data["feedback"].(string); ok && feedback != "" {
		xml += `
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="26"/>
          <w:color w:val="006699"/>
        </w:rPr>
        <w:t xml:space="preserve">Summary &amp; Feedback</w:t>
      </w:r>
    </w:p>`
		xml += createParagraph(feedback, false, false)
	}

	// Recommendations
	if recommendations, ok := data["recommendations"].([]string); ok {
		xml += `
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:b/>
          <w:sz w:val="26"/>
          <w:color w:val="006699"/>
        </w:rPr>
        <w:t xml:space="preserve">Recommendations</w:t>
      </w:r>
    </w:p>`

		for _, rec := range recommendations {
			xml += createParagraph("- "+rec, false, false)
		}
	}

	return xml
}

// createParagraph creates a Word XML paragraph
func createParagraph(text string, bold bool, isHeading bool) string {
	boldPr := ""
	if bold {
		boldPr = `<w:rPr><w:b/></w:rPr>`
	}

	size := "22"
	if isHeading {
		size = "26"
	}

	return fmt.Sprintf(`
    <w:p>
      <w:r>
        %s
        <w:rPr>
          <w:sz w:val="%s"/>
        </w:rPr>
        <w:t xml:space="preserve">%s</w:t>
      </w:r>
    </w:p>`, boldPr, size, escapeXML(text))
}

// buildStylesXML creates the styles XML for the document
func (g *DOCXGenerator) buildStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:rPr>
      <w:sz w:val="22"/>
      <w:color w:val="333333"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
    <w:rPr>
      <w:b/>
      <w:sz w:val="36"/>
      <w:color w:val="003366"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="Heading 1"/>
    <w:rPr>
      <w:b/>
      <w:sz w:val="26"/>
      <w:color w:val="006699"/>
    </w:rPr>
  </w:style>
</w:styles>`
}

// writeFile writes a file to the ZIP archive
func writeFile(zw *zip.Writer, name string, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("creating zip entry %s: %w", name, err)
	}
	_, err = w.Write([]byte(content))
	return err
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
