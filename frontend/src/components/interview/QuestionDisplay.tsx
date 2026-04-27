import React from 'react';

interface QuestionDisplayProps {
  question: string;
  hint?: string;
  category?: string;
}

const QuestionDisplay: React.FC<QuestionDisplayProps> = ({
  question,
  hint,
  category,
}) => {
  return (
    <div className="question-display">
      {category && (
        <span className="question-display__category">{category}</span>
      )}
      <h3 className="question-display__question">{question}</h3>
      {hint && <p className="question-display__hint">Подсказка: {hint}</p>}
    </div>
  );
};

export default QuestionDisplay;
