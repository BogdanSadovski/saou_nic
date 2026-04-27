type Props = {
  role: string;
  level: string;
  vacancyTitle?: string;
  interviewMode?: string;
  timerLabel: string;
  onExit: () => void;
};

export function InterviewTopBar({
  role,
  level,
  vacancyTitle,
  interviewMode,
  timerLabel,
  onExit,
}: Props) {
  const modeLabel = interviewMode === "theory" ? "Теория" : "Практика";

  return (
    <header className="interview-topbar">
      <div className="interview-pill">{timerLabel}</div>
      <div className="interview-meta">
        <strong>{role}</strong>
        <span>{level}</span>
        {vacancyTitle ? <small>{vacancyTitle}</small> : null}
        {interviewMode ? <small className="interview-mode-pill">{modeLabel}</small> : null}
      </div>
      <button className="interview-exit-btn" onClick={onExit} type="button">
        Выйти
      </button>
    </header>
  );
}
