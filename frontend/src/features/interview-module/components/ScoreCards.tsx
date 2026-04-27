type Props = {
  correctness: number;
  clarity: number;
  completeness: number;
  relevance: number;
  overallScore: number;
};

const metric = (label: string, value: number) => (
  <article className="score-card" key={label}>
    <h4>{label}</h4>
    <p>{Math.round(value)}</p>
  </article>
);

export function ScoreCards({ correctness, clarity, completeness, relevance, overallScore }: Props) {
  return (
    <section className="score-grid">
      {metric("Корректность", correctness)}
      {metric("Ясность", clarity)}
      {metric("Полнота", completeness)}
      {metric("Релевантность", relevance)}
      {metric("Итог", overallScore)}
    </section>
  );
}
