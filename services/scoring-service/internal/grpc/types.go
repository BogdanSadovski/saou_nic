package grpcserver

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type UnimplementedScoringServiceServer struct{}

type EvaluateScoreRequest struct {
	SubmissionId  string
	ScoreType     string
	RepositoryUrl string
	CommitHash    string
	RubricId      *string
}

func (r *EvaluateScoreRequest) GetSubmissionId() string {
	if r == nil {
		return ""
	}
	return r.SubmissionId
}
func (r *EvaluateScoreRequest) GetScoreType() string {
	if r == nil {
		return ""
	}
	return r.ScoreType
}
func (r *EvaluateScoreRequest) GetRepositoryUrl() string {
	if r == nil {
		return ""
	}
	return r.RepositoryUrl
}
func (r *EvaluateScoreRequest) GetCommitHash() string {
	if r == nil {
		return ""
	}
	return r.CommitHash
}

type EvaluateScoreResponse struct {
	ScoreId    string
	TotalScore float64
	MaxScore   float64
	Percentage float64
	Grade      string
	Status     string
	Breakdown  []*CriterionScore
}

type GetScoreRequest struct{ ScoreId string }

func (r *GetScoreRequest) GetScoreId() string {
	if r == nil {
		return ""
	}
	return r.ScoreId
}

type GetScoreResponse struct{ Score *Score }

type GetEvaluationResultRequest struct{ SubmissionId string }

func (r *GetEvaluationResultRequest) GetSubmissionId() string {
	if r == nil {
		return ""
	}
	return r.SubmissionId
}

type GetEvaluationResultResponse struct {
	OverallScore  float64
	OverallGrade  string
	PassThreshold bool
	ScoreCount    int64
}

type CriterionScore struct {
	CriterionName string
	Score         float64
	MaxScore      float64
	Weight        float64
	WeightedScore float64
	Comments      []string
}

type Score struct {
	Id           string
	SubmissionId string
	ScoreType    string
	TotalScore   float64
	MaxScore     float64
	Percentage   float64
	Grade        string
	Status       string
	Breakdown    []*CriterionScore
}

func RegisterScoringServiceServer(_ *grpc.Server, _ interface{}) {}

var _ = context.Background
var _ = wrapperspb.Bool(true)
