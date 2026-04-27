package grpcserver

import (
	"context"
	"fmt"

	"scoring-service/internal/domain"
	"scoring-service/internal/service"
)

// ScoringHandler implements the gRPC scoring service server.
type ScoringHandler struct {
	UnimplementedScoringServiceServer
	scoringService *service.ScoringService
}

// NewScoringHandler creates a new gRPC scoring handler.
func NewScoringHandler(scoringService *service.ScoringService) *ScoringHandler {
	return &ScoringHandler{
		scoringService: scoringService,
	}
}

// EvaluateScore processes a gRPC scoring request.
func (h *ScoringHandler) EvaluateScore(ctx context.Context, req *EvaluateScoreRequest) (*EvaluateScoreResponse, error) {
	if req.GetSubmissionId() == "" {
		return nil, fmt.Errorf("submission_id is required")
	}

	scoreType := domain.ScoreType(req.GetScoreType())
	scoringReq := domain.ScoringRequest{
		SubmissionID:  req.GetSubmissionId(),
		RepositoryURL: req.GetRepositoryUrl(),
		CommitHash:    req.GetCommitHash(),
		ScoreType:     scoreType,
	}

	if req.RubricId != nil {
		scoringReq.RubricID = req.RubricId
	}

	score, err := h.scoringService.Evaluate(ctx, scoringReq)
	if err != nil {
		return nil, fmt.Errorf("evaluate score: %w", err)
	}

	return &EvaluateScoreResponse{
		ScoreId:     score.ID,
		TotalScore:  score.TotalScore,
		MaxScore:    score.MaxScore,
		Percentage:  score.Percentage,
		Grade:       score.Grade,
		Status:      string(score.Status),
		Breakdown:   convertBreakdownToProto(score.Breakdown),
	}, nil
}

// GetScore retrieves a score by ID via gRPC.
func (h *ScoringHandler) GetScore(ctx context.Context, req *GetScoreRequest) (*GetScoreResponse, error) {
	score, err := h.scoringService.GetScore(ctx, req.GetScoreId())
	if err != nil {
		return nil, fmt.Errorf("get score: %w", err)
	}

	return &GetScoreResponse{
		Score: &Score{
			Id:           score.ID,
			SubmissionId: score.SubmissionID,
			ScoreType:    string(score.ScoreType),
			TotalScore:   score.TotalScore,
			MaxScore:     score.MaxScore,
			Percentage:   score.Percentage,
			Grade:        score.Grade,
			Status:       string(score.Status),
			Breakdown:    convertBreakdownToProto(score.Breakdown),
		},
	}, nil
}

// GetEvaluationResult retrieves the aggregated evaluation result for a submission.
func (h *ScoringHandler) GetEvaluationResult(ctx context.Context, req *GetEvaluationResultRequest) (*GetEvaluationResultResponse, error) {
	result, err := h.scoringService.GetEvaluationResult(ctx, req.GetSubmissionId())
	if err != nil {
		return nil, fmt.Errorf("get evaluation result: %w", err)
	}

	return &GetEvaluationResultResponse{
		OverallScore:   result.OverallScore,
		OverallGrade:   result.OverallGrade,
		PassThreshold:  result.PassThreshold,
		ScoreCount:     int64(len(result.Scores)),
	}, nil
}

// convertBreakdownToProto converts domain criterion scores to protobuf format.
func convertBreakdownToProto(breakdown []domain.CriterionScore) []*CriterionScore {
	protoScores := make([]*CriterionScore, 0, len(breakdown))
	for _, cs := range breakdown {
		protoScores = append(protoScores, &CriterionScore{
			CriterionName: cs.CriterionName,
			Score:         cs.Score,
			MaxScore:      cs.MaxScore,
			Weight:        cs.Weight,
			WeightedScore: cs.WeightedScore,
			Comments:      cs.Comments,
		})
	}
	return protoScores
}
