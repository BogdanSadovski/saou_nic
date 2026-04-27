import React, { useState, FormEvent } from 'react';

interface InterviewSetupProps {
  onStart?: (config: InterviewConfig) => void;
}

interface InterviewConfig {
  mode: 'mock' | 'real';
  difficulty: 'junior' | 'mid' | 'senior';
  category: string;
  duration: number;
  useVideo: boolean;
}

const categories = [
  'Frontend',
  'Backend',
  'Full Stack',
  'DevOps',
  'Data Science',
  'System Design',
];

const InterviewSetup: React.FC<InterviewSetupProps> = ({ onStart }) => {
  const [mode, setMode] = useState<InterviewConfig['mode']>('mock');
  const [difficulty, setDifficulty] = useState<InterviewConfig['difficulty']>('mid');
  const [category, setCategory] = useState(categories[0]);
  const [duration, setDuration] = useState(30);
  const [useVideo, setUseVideo] = useState(true);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onStart?.({ mode, difficulty, category, duration, useVideo });
  };

  return (
    <form className="interview-setup" onSubmit={handleSubmit}>
      <h2 className="interview-setup__title">Interview Setup</h2>

      <div className="interview-setup__field">
        <label className="interview-setup__label">Interview Mode</label>
        <select
          className="interview-setup__select"
          value={mode}
          onChange={(e) => setMode(e.target.value as InterviewConfig['mode'])}
        >
          <option value="mock">Mock Interview</option>
          <option value="real">Real Interview</option>
        </select>
      </div>

      <div className="interview-setup__field">
        <label className="interview-setup__label">Difficulty</label>
        <select
          className="interview-setup__select"
          value={difficulty}
          onChange={(e) => setDifficulty(e.target.value as InterviewConfig['difficulty'])}
        >
          <option value="junior">Junior</option>
          <option value="mid">Mid-Level</option>
          <option value="senior">Senior</option>
        </select>
      </div>

      <div className="interview-setup__field">
        <label className="interview-setup__label">Category</label>
        <select
          className="interview-setup__select"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
        >
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat}
            </option>
          ))}
        </select>
      </div>

      <div className="interview-setup__field">
        <label className="interview-setup__label">Duration (minutes): {duration}</label>
        <input
          type="range"
          className="interview-setup__range"
          min={2}
          max={60}
          step={1}
          value={duration}
          onChange={(e) => setDuration(Number(e.target.value))}
        />
      </div>

      <div className="interview-setup__field">
        <label className="interview-setup__checkbox">
          <input
            type="checkbox"
            checked={useVideo}
            onChange={(e) => setUseVideo(e.target.checked)}
          />
          Enable Video Recording
        </label>
      </div>

      <button type="submit" className="interview-setup__submit">
        Start Interview
      </button>
    </form>
  );
};

export default InterviewSetup;
