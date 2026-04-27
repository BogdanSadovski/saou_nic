type Props = {
  onClick: () => void;
};

export function RetryInterviewButton({ onClick }: Props) {
  return (
    <button className="retry-btn" onClick={onClick} type="button">
      Пройти интервью заново
    </button>
  );
}
