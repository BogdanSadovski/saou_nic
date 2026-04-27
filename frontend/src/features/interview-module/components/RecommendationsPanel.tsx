type Props = { items: string[] };

export function RecommendationsPanel({ items }: Props) {
  return (
    <section className="result-panel">
      <h3>Рекомендации</h3>
      <ul>
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </section>
  );
}
