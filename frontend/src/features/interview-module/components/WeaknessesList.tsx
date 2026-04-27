type Props = { items: string[] };

export function WeaknessesList({ items }: Props) {
  return (
    <section className="result-panel">
      <h3>Зоны роста</h3>
      <ul>
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </section>
  );
}
