type Props = {
  loading?: boolean;
  onClick: () => void;
};

export function StartInterviewButton({ loading = false, onClick }: Props) {
  return (
    <button className="interview-start-btn" disabled={loading} onClick={onClick} type="button">
      {loading ? "Запуск..." : "Начать интервью"}
    </button>
  );
}
