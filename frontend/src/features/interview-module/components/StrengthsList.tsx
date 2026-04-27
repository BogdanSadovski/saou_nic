type Props = { items: string[] };

export function StrengthsList({ items }: Props) {
  return (
    <section className="result-panel">
      <h3>Сильные стороны</h3>
      <ul>
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </section>
  );
}
