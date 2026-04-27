import type { InterviewMode } from "../types";

type Props = {
  value: InterviewMode;
  onChange: (value: InterviewMode) => void;
};

const options: Array<{ value: InterviewMode; label: string; hint: string }> = [
  {
    value: "practice",
    label: "Практика",
    hint: "Лайв-код: задание, решение в редакторе, проверка по кнопке",
  },
  {
    value: "theory",
    label: "Теория",
    hint: "Только вопросы по техтеории в выбранном направлении",
  },
];

export function InterviewModeSelector({ value, onChange }: Props) {
  return (
    <div className="interview-field interview-mode-field">
      <label>Формат интервью</label>
      <div className="interview-mode-grid">
        {options.map((option) => {
          const selected = option.value === value;
          return (
            <button
              key={option.value}
              className={selected ? "mode-option selected" : "mode-option"}
              onClick={() => onChange(option.value)}
              type="button"
            >
              <span className="mode-title">{option.label}</span>
              <span className="mode-hint">{option.hint}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
