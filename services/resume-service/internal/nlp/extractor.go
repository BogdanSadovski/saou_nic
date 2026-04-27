package nlp

import (
	"regexp"
	"strings"
	"unicode"
)

// ResumeExtractor provides advanced extraction capabilities
type ResumeExtractor struct {
	urlRegex     *regexp.Regexp
	dateRegex    *regexp.Regexp
	bulletRegex  *regexp.Regexp
}

// ExtractionResult holds all extracted data from a resume
type ExtractionResult struct {
	ContactInfo    ContactInfo
	Sections       map[string]string
	KeyPhrases     []string
	ActionVerbs    []string
	Metrics        []string
	Technologies   []string
}

// ContactInfo holds extracted contact details
type ContactInfo struct {
	Emails     []string
	Phones     []string
	Urls       []string
	Addresses  []string
	SocialLinks map[string]string
}

// NewResumeExtractor creates a new extractor instance
func NewResumeExtractor() *ResumeExtractor {
	return &ResumeExtractor{
		urlRegex:    regexp.MustCompile(`https?://[^\s]+`),
		dateRegex:   regexp.MustCompile(`\b(?:jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)[,\s]+\d{4}\b|\b\d{1,2}/\d{4}\b|\b\d{4}\b`),
		bulletRegex: regexp.MustCompile(`^[•\-\*\u2022\u2023\u2043]\s*`),
	}
}

// Extract performs comprehensive extraction from parsed resume text
func (e *ResumeExtractor) Extract(text string) *ExtractionResult {
	result := &ExtractionResult{
		Sections: make(map[string]string),
	}

	result.ContactInfo = e.extractContactInfo(text)
	result.Sections = e.extractSections(text)
	result.KeyPhrases = e.extractKeyPhrases(text)
	result.ActionVerbs = e.extractActionVerbs(text)
	result.Metrics = e.extractMetrics(text)
	result.Technologies = e.extractTechnologies(text)

	return result
}

// extractContactInfo extracts all contact information
func (e *ResumeExtractor) extractContactInfo(text string) ContactInfo {
	info := ContactInfo{
		SocialLinks: make(map[string]string),
	}

	// Extract emails
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	info.Emails = emailRegex.FindAllString(text, -1)

	// Extract phones
	phoneRegex := regexp.MustCompile(`(?:\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	info.Phones = phoneRegex.FindAllString(text, -1)

	// Extract URLs
	info.Urls = e.urlRegex.FindAllString(text, -1)

	// Categorize social links
	for _, url := range info.Urls {
		lowerURL := strings.ToLower(url)
		if strings.Contains(lowerURL, "linkedin") {
			info.SocialLinks["linkedin"] = url
		} else if strings.Contains(lowerURL, "github") {
			info.SocialLinks["github"] = url
		} else if strings.Contains(lowerURL, "twitter") || strings.Contains(lowerURL, "x.com") {
			info.SocialLinks["twitter"] = url
		}
	}

	return info
}

// extractSections splits resume into logical sections
func (e *ResumeExtractor) extractSections(text string) map[string]string {
	sections := make(map[string]string)

	// Common section headers
	sectionPattern := regexp.MustCompile(`(?m)^(EXPERIENCE|WORK EXPERIENCE|EMPLOYMENT|EDUCATION|SKILLS|SUMMARY|OBJECTIVE|CERTIFICATIONS|PROJECTS|AWARDS|PUBLICATIONS|LANGUAGES|VOLUNTEER|REFERENCES)\s*$`)

	matches := sectionPattern.FindAllStringIndex(text, -1)

	for i, match := range matches {
		sectionName := strings.ToLower(strings.TrimSpace(text[match[0]:match[1]]))

		var content string
		if i+1 < len(matches) {
			content = strings.TrimSpace(text[match[1]:matches[i+1][0]])
		} else {
			content = strings.TrimSpace(text[match[1]:])
		}

		sections[sectionName] = content
	}

	return sections
}

// extractKeyPhrases identifies important phrases using heuristics
func (e *ResumeExtractor) extractKeyPhrases(text string) []string {
	var phrases []string

	// Look for capitalized phrases (potential key terms)
	capitalPhraseRegex := regexp.MustCompile(`\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+){1,3})\b`)
	matches := capitalPhraseRegex.FindAllString(text, -1)

	seen := make(map[string]bool)
	for _, phrase := range matches {
		// Filter out common non-key phrases
		if !isCommonPhrase(phrase) && !seen[phrase] {
			phrases = append(phrases, phrase)
			seen[phrase] = true
		}
	}

	return phrases
}

// extractActionVerbs finds action verbs commonly used in resumes
func (e *ResumeExtractor) extractActionVerbs(text string) []string {
	actionVerbs := []string{
		"developed", "designed", "implemented", "managed", "led",
		"created", "built", "optimized", "improved", "delivered",
		"architected", "deployed", "maintained", "refactored", "migrated",
		"established", "coordinated", "facilitated", "achieved", "increased",
		"reduced", "automated", "streamlined", "launched", "spearheaded",
	}

	lowerText := strings.ToLower(text)
	var found []string

	for _, verb := range actionVerbs {
		if strings.Contains(lowerText, verb) {
			found = append(found, verb)
		}
	}

	return found
}

// extractMetrics finds quantitative achievements
func (e *ResumeExtractor) extractMetrics(text string) []string {
	var metrics []string

	// Pattern for numbers with context (e.g., "increased by 50%", "managed team of 10")
	metricPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\d+%`),
		regexp.MustCompile(`\$\d+(?:\.\d+)?[kmbKMB]?`),
		regexp.MustCompile(`\d+\+\s*\w+`),
		regexp.MustCompile(`team of \d+`),
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		for _, pattern := range metricPatterns {
			if pattern.MatchString(line) {
				metrics = append(metrics, strings.TrimSpace(line))
				break
			}
		}
	}

	return metrics
}

// extractTechnologies identifies technology mentions
func (e *ResumeExtractor) extractTechnologies(text string) []string {
	technologies := []string{
		// Languages
		"Go", "Python", "Java", "JavaScript", "TypeScript", "C++", "C#", "Ruby", "PHP", "Swift", "Kotlin", "Rust",
		// Frameworks
		"React", "Angular", "Vue.js", "Django", "Flask", "Spring Boot", "Express", "Gin", "Ruby on Rails",
		// Databases
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "Elasticsearch", "Cassandra", "DynamoDB",
		// Cloud & DevOps
		"AWS", "GCP", "Azure", "Docker", "Kubernetes", "Terraform", "Ansible", "Jenkins", "GitLab CI",
		// Tools
		"Git", "Jira", "Confluence", "Grafana", "Prometheus", "Datadog",
		// Concepts
		"REST API", "GraphQL", "Microservices", "Event-driven", "CI/CD", "Agile", "Scrum",
	}

	var found []string
	seen := make(map[string]bool)

	for _, tech := range technologies {
		if strings.Contains(text, tech) && !seen[tech] {
			found = append(found, tech)
			seen[tech] = true
		}
	}

	return found
}

// isCommonPhrase filters out non-informative capitalized phrases
func isCommonPhrase(phrase string) bool {
	commonPhrases := map[string]bool{
		"I": true, "A": true, "The": true, "And": true, "Or": true,
		"Work Experience": true, "Education": true, "Skills": true,
		"References": true, "Available Upon Request": true,
	}

	return commonPhrases[phrase]
}

// NormalizeText cleans and normalizes extracted text
func NormalizeText(text string) string {
	// Normalize whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Normalize quotes
	text = strings.ReplaceAll(text, "\u2018", "'")
	text = strings.ReplaceAll(text, "\u2019", "'")
	text = strings.ReplaceAll(text, "\u201C", "\"")
	text = strings.ReplaceAll(text, "\u201D", "\"")

	// Normalize dashes
	text = strings.ReplaceAll(text, "\u2013", "-")
	text = strings.ReplaceAll(text, "\u2014", "-")

	return strings.TrimSpace(text)
}

// CountWords counts words in text
func CountWords(text string) int {
	count := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			count++
			inWord = true
		}
	}

	return count
}
