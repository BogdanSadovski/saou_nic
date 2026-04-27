type Props = {
  value: number;
  onChange: (value: number) => void;
};

const durations = [2, 10, 15, 20, 30, 45, 60];

export function DurationSelector({ value, onChange }: Props) {
  return (
    <div className="interview-field">
      <label htmlFor="duration">Длительность</label>
      <select
        id="duration"
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      >
        {durations.map((duration) => (
          <option key={duration} value={duration}>
            {duration} мин
          </option>
        ))}
      </select>
    </div>
  );
}
