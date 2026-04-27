import type { InterviewLevel } from "../types";

type Props = {
  value: InterviewLevel;
  onChange: (value: InterviewLevel) => void;
};

const levels: InterviewLevel[] = ["Junior", "Middle", "Senior"];

export function LevelSelector({ value, onChange }: Props) {
  return (
    <div className="interview-field">
      <label htmlFor="level">Уровень</label>
      <select id="level" value={value} onChange={(event) => onChange(event.target.value as InterviewLevel)}>
        {levels.map((level) => (
          <option key={level} value={level}>
            {level}
          </option>
        ))}
      </select>
    </div>
  );
}
