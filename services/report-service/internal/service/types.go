package service

// GenerateReportRequest is payload for manual PDF/DOCX generation endpoints.
type GenerateReportRequest struct {
	TemplateData map[string]interface{} `json:"template_data"`
}
