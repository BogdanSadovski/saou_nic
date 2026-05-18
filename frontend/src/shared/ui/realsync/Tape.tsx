type TapeProps = {
  items: string[];
};

export function Tape({ items }: TapeProps) {
  const row = items.flatMap((s, i) => [
    <span key={`${s}-${i}`}>{s}</span>,
    <span key={`d-${s}-${i}`} className="dot"></span>,
  ]);
  return (
    <div className="tape">
      <div className="tape-track">
        {row}
        {row}
      </div>
    </div>
  );
}
