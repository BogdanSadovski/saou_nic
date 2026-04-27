import React from 'react';
import QuestionDisplay from './QuestionDisplay';
import AnswerRecorder from './AnswerRecorder';
import Timer from './Timer';

interface InterviewRoomProps {
  question?: string;
  questionIndex?: number;
  totalQuestions?: number;
  durationSeconds?: number;
  onAnswer?: (answer: string) => void;
  onNext?: () => void;
  onEnd?: () => void;
  useVideo?: boolean;
}

const InterviewRoom: React.FC<InterviewRoomProps> = ({
  question = 'В чем разница между let, const и var?',
  questionIndex = 1,
  totalQuestions = 10,
  durationSeconds = 1800,
  onAnswer,
  onNext,
  onEnd,
  useVideo = true,
}) => {
  return (
    <div className="interview-room">
      <header className="interview-room__header">
        <span className="interview-room__progress">
          Вопрос {questionIndex} из {totalQuestions}
        </span>
        <Timer durationSeconds={durationSeconds} onEnd={onEnd} />
      </header>

      <div className="interview-room__content">
        <QuestionDisplay question={question} />
        {useVideo && <AnswerRecorder />}
      </div>

      <div className="interview-room__actions">
        <button className="interview-room__btn" onClick={() => onAnswer?.('')}>
          Пропустить
        </button>
        <button
          className="interview-room__btn interview-room__btn--primary"
          onClick={onNext}
        >
          Следующий вопрос
        </button>
      </div>
    </div>
  );
};

export default InterviewRoom;
