import React from 'react';

interface AnalysisScore {
  category: string;
  score: number;
  maxScore: number;
  feedback?: string;
}

interface ResumeAnalysisProps {
  scores?: AnalysisScore[];
  overallScore?: number;
  suggestions?: string[];
  isLoading?: boolean;
}

const defaultScores: AnalysisScore[] = [
  { category: 'Formatting', score: 8, maxScore: 10, feedback: 'Good structure' },
  { category: 'Keywords', score: 6, maxScore: 10, feedback: 'Add more industry keywords' },
  { category: 'Experience', score: 9, maxScore: 10, feedback: 'Strong experience section' },
  { category: 'Education', score: 7, maxScore: 10 },
];

const ResumeAnalysis: React.FC<ResumeAnalysisProps> = ({
  scores = defaultScores,
  overallScore = 75,
  suggestions = ['Tailor your resume to the job description', 'Quantify achievements with metrics'],
  isLoading = false,
}) => {
  if (isLoading) {
    return <div className="resume-analysis resume-analysis--loading">Analyzing resume...</div>;
  }

  return (
    <div className="resume-analysis">
      <div className="resume-analysis__overall">
        <div className="resume-analysis__score-circle">{overallScore}%</div>
        <h3>Overall Score</h3>
      </div>

      <div className="resume-analysis__breakdown">
        <h4>Category Breakdown</h4>
        {scores.map((item) => (
          <div key={item.category} className="resume-analysis__category">
            <div className="resume-analysis__category-header">
              <span>{item.category}</span>
              <span>
                {item.score}/{item.maxScore}
              </span>
            </div>
            <div className="resume-analysis__bar">
              <div
                className="resume-analysis__bar-fill"
                style={{ width: `${(item.score / item.maxScore) * 100}%` }}
              />
            </div>
            {item.feedback && <p className="resume-analysis__feedback">{item.feedback}</p>}
          </div>
        ))}
      </div>

      {suggestions.length > 0 && (
        <div className="resume-analysis__suggestions">
          <h4>Suggestions</h4>
          <ul>
            {suggestions.map((s, idx) => (
              <li key={idx}>{s}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default ResumeAnalysis;
