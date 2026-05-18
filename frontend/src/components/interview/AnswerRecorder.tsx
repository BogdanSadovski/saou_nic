import React, { useState } from 'react';

interface AnswerRecorderProps {
  onSave?: (answer: string) => void;
  isRecording?: boolean;
}

const AnswerRecorder: React.FC<AnswerRecorderProps> = ({ onSave, isRecording = false }) => {
  const [answer, setAnswer] = useState('');
  const [recording, setRecording] = useState(isRecording);

  const handleToggleRecording = () => {
    setRecording((prev) => !prev);
  };

  const handleSave = () => {
    onSave?.(answer);
    setAnswer('');
  };

  return (
    <div className="answer-recorder">
      <div className="answer-recorder__video">
        {recording ? (
          <div className="answer-recorder__recording-indicator">
            <span className="answer-recorder__rec-dot" /> Запись...
          </div>
        ) : (
          <p className="answer-recorder__placeholder">Превью камеры</p>
        )}
      </div>

      <div className="answer-recorder__controls">
        <button
          className={`answer-recorder__btn ${recording ? 'answer-recorder__btn--stop' : 'answer-recorder__btn--start'}`}
          onClick={handleToggleRecording}
        >
          {recording ? 'Остановить запись' : 'Начать запись'}
        </button>
      </div>

      <textarea
        className="answer-recorder__text"
        placeholder="Введите ответ (опционально)..."
        value={answer}
        onChange={(e) => setAnswer(e.target.value)}
        rows={4}
      />

      <button
        className="answer-recorder__save"
        onClick={handleSave}
        disabled={!answer.trim()}
      >
        Сохранить ответ
      </button>
    </div>
  );
};

export default AnswerRecorder;
