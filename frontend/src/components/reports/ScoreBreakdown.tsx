import React from 'react';

interface ScoreItem {
  category: string;
  score: number;
  weight: number;
}

interface ScoreBreakdownProps {
  scores?: ScoreItem[];
  totalScore?: number;
  maxScore?: number;
}

const defaultScores: ScoreItem[] = [
  { category: 'Technical Skills', score: 85, weight: 40 },
  { category: 'Communication', score: 78, weight: 25 },
  { category: 'Problem Solving', score: 90, weight: 25 },
  { category: 'Culture Fit', score: 72, weight: 10 },
];

const ScoreBreakdown: React.FC<ScoreBreakdownProps> = ({
  scores = defaultScores,
  totalScore = 82,
  maxScore = 100,
}) => {
  const weightedTotal = scores.reduce(
    (acc, item) => acc + (item.score * item.weight) / 100,
    0
  );

  return (
    <div className="score-breakdown">
      <h3 className="score-breakdown__title">Score Breakdown</h3>

      <div className="score-breakdown__total">
        <span className="score-breakdown__total-label">Total Score</span>
        <span className="score-breakdown__total-value">
          {totalScore}/{maxScore}
        </span>
      </div>

      <div className="score-breakdown__list">
        {scores.map((item) => (
          <div key={item.category} className="score-breakdown__item">
            <div className="score-breakdown__item-header">
              <span className="score-breakdown__item-category">
                {item.category}
              </span>
              <span className="score-breakdown__item-score">
                {item.score} ({item.weight}%)
              </span>
            </div>
            <div className="score-breakdown__bar">
              <div
                className="score-breakdown__bar-fill"
                style={{ width: `${item.score}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      <div className="score-breakdown__weighted">
        Weighted Score: {weightedTotal.toFixed(1)}
      </div>
    </div>
  );
};

export default ScoreBreakdown;
