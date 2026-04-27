package service

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"resume-service/internal/domain"
)

// NLPAnalyzer handles natural language processing of resume text
type NLPAnalyzer struct {
	emailRegex    *regexp.Regexp
	phoneRegex    *regexp.Regexp
	skillKeywords map[string]bool
	modelMutex    sync.RWMutex
	// In production, this would hold loaded ML models
	modelLoaded bool
}

// ExtractedEntities holds all entities extracted from resume text
type ExtractedEntities struct {
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	Summary        string
	Skills         []string
	Experience     []domain.ExperienceEntry
	Education      []domain.EducationEntry
	Languages      []string
	Certifications []string
}

// NewNLPAnalyzer creates a new NLP analyzer
func NewNLPAnalyzer() *NLPAnalyzer {
	analyzer := &NLPAnalyzer{
		emailRegex: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phoneRegex: regexp.MustCompile(`(?:\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`),
		skillKeywords: map[string]bool{
			// Programming languages
			"go": true, "python": true, "java": true, "javascript": true, "typescript": true,
			"c++": true, "c#": true, "ruby": true, "php": true, "swift": true,
			// Frameworks
			"react": true, "angular": true, "vue": true, "django": true, "flask": true,
			"spring": true, "express": true, "gin": true, "rails": true,
			// Databases
			"postgresql": true, "mysql": true, "mongodb": true, "redis": true, "elasticsearch": true,
			// DevOps
			"docker": true, "kubernetes": true, "terraform": true, "ansible": true, "jenkins": true,
			"git": true, "ci/cd": true, "aws": true, "gcp": true, "azure": true,
			// General
			"sql": true, "rest": true, "graphql": true, "microservices": true, "agile": true,
			"scrum": true, "linux": true, "machine learning": true, "data science": true,
		},
		modelLoaded: false,
	}

	// In production, load ML models here
	// analyzer.loadModels()

	return analyzer
}

// Analyze performs NLP analysis on resume text
func (a *NLPAnalyzer) Analyze(ctx context.Context, text string) (*ExtractedEntities, error) {
	entities := &ExtractedEntities{}

	// Extract contact information
	entities.Email = a.extractEmail(text)
	entities.Phone = a.extractPhone(text)

	// Extract name (simplified - in production, use NER model)
	entities.FirstName, entities.LastName = a.extractName(text)

	// Extract skills
	entities.Skills = a.extractSkills(text)

	// Extract experience sections
	entities.Experience = a.extractExperience(text)

	// Extract education sections
	entities.Education = a.extractEducation(text)

	// Extract languages
	entities.Languages = a.extractLanguages(text)

	// Extract certifications
	entities.Certifications = a.extractCertifications(text)

	// Extract summary (first paragraph or section)
	entities.Summary = a.extractSummary(text)

	return entities, nil
}

// extractEmail finds email addresses in text
func (a *NLPAnalyzer) extractEmail(text string) string {
	matches := a.emailRegex.FindString(text)
	return matches
}

// extractPhone finds phone numbers in text
func (a *NLPAnalyzer) extractPhone(text string) string {
	matches := a.phoneRegex.FindString(text)
	return matches
}

// extractName attempts to extract a person's name (simplified)
func (a *NLPAnalyzer) extractName(text string) (string, string) {
	// This is a simplified implementation
	// In production, use a Named Entity Recognition (NER) model
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		// Assume the first non-empty line contains the name
		firstLine := strings.TrimSpace(lines[0])
		parts := strings.Fields(firstLine)
		if len(parts) >= 2 {
			return parts[0], strings.Join(parts[1:], " ")
		} else if len(parts) == 1 {
			return parts[0], ""
		}
	}
	return "", ""
}

// extractSkills identifies technical skills mentioned in the text
func (a *NLPAnalyzer) extractSkills(text string) []string {
	lowerText := strings.ToLower(text)
	var skills []string
	skillSet := make(map[string]bool)

	for skill := range a.skillKeywords {
		if strings.Contains(lowerText, skill) && !skillSet[skill] {
			skills = append(skills, skill)
			skillSet[skill] = true
		}
	}

	return skills
}

// extractExperience parses work experience entries
func (a *NLPAnalyzer) extractExperience(text string) []domain.ExperienceEntry {
	// Simplified implementation
	// In production, use structured parsing or ML-based extraction
	var entries []domain.ExperienceEntry

	// Look for experience section markers
	sections := strings.Split(text, "\n")
	var currentEntry *domain.ExperienceEntry

	for _, line := range sections {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect company/position lines (simplified heuristic)
		if strings.Contains(strings.ToLower(line), "engineer") ||
			strings.Contains(strings.ToLower(line), "developer") ||
			strings.Contains(strings.ToLower(line), "manager") ||
			strings.Contains(strings.ToLower(line), "architect") {

			if currentEntry != nil {
				entries = append(entries, *currentEntry)
			}

			currentEntry = &domain.ExperienceEntry{
				Position: line,
			}
		} else if currentEntry != nil && currentEntry.Company == "" {
			currentEntry.Company = line
		}
	}

	if currentEntry != nil {
		entries = append(entries, *currentEntry)
	}

	return entries
}

// extractEducation parses education entries
func (a *NLPAnalyzer) extractEducation(text string) []domain.EducationEntry {
	// Simplified implementation
	var entries []domain.EducationEntry

	eduKeywords := []string{"bachelor", "master", "phd", "doctor", "degree", "university", "college", "bsc", "msc"}
	lines := strings.Split(strings.ToLower(text), "\n")

	for _, line := range lines {
		for _, keyword := range eduKeywords {
			if strings.Contains(line, keyword) {
				entries = append(entries, domain.EducationEntry{
					Degree: strings.TrimSpace(line),
				})
				break
			}
		}
	}

	return entries
}

// extractLanguages identifies languages mentioned in the text
func (a *NLPAnalyzer) extractLanguages(text string) []string {
	languages := []string{}
	lowerText := strings.ToLower(text)

	knownLanguages := []string{
		"english", "spanish", "french", "german", "chinese", "japanese",
		"korean", "russian", "arabic", "portuguese", "italian", "dutch",
		"turkish", "hindi", "polish", "ukrainian",
	}

	for _, lang := range knownLanguages {
		if strings.Contains(lowerText, lang) {
			languages = append(languages, lang)
		}
	}

	return languages
}

// extractCertifications finds certification mentions
func (a *NLPAnalyzer) extractCertifications(text string) []string {
	certifications := []string{}

	certPatterns := []string{
		"AWS Certified", "Google Cloud Professional", "Azure Certified",
		"CKA", "CKAD", "PMP", "CISSP", "CCNA", "CCNP",
	}

	for _, cert := range certPatterns {
		if strings.Contains(text, cert) {
			certifications = append(certifications, cert)
		}
	}

	return certifications
}

// extractSummary extracts a summary or objective statement
func (a *NLPAnalyzer) extractSummary(text string) string {
	lowerText := strings.ToLower(text)

	summaryMarkers := []string{"summary", "objective", "about me", "profile"}

	for _, marker := range summaryMarkers {
		idx := strings.Index(lowerText, marker)
		if idx != -1 {
			// Extract the paragraph following the marker
			remaining := text[idx+len(marker):]
			endIdx := strings.Index(remaining, "\n\n")
			if endIdx == -1 {
				endIdx = len(remaining)
			}
			return strings.TrimSpace(remaining[:endIdx])
		}
	}

	// Fallback: return first paragraph
	lines := strings.Split(text, "\n\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return ""
}
