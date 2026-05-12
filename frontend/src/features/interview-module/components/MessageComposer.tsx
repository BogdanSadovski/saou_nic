import { useState } from "react";

type Props = {
  pending: string;
  disabled?: boolean;
  onPendingChange: (value: string) => void;
  onSend: (value: string) => Promise<void>;
};

/**
 * Chat composer with two affordances:
 *   - "Отправить" submits the typed answer.
 *   - "Пропустить" submits the literal sentinel "__skip__", which the
 *     interview-service recognises in detectCandidateIntent and routes
 *     to the topic-switch path (verdict='skipped', no grading, next
 *     topic). Better than asking the user to type "пропустить" each
 *     time and avoids polluting the transcript.
 */
export function MessageComposer({ pending, disabled = false, onPendingChange, onSend }: Props) {
  const [sending, setSending] = useState(false);
  const [skipping, setSkipping] = useState(false);

  const submit = async () => {
    const value = pending.trim();
    if (!value || sending || skipping || disabled) {
      return;
    }
    setSending(true);
    try {
      await onSend(value);
      onPendingChange("");
    } finally {
      setSending(false);
    }
  };

  const skip = async () => {
    if (sending || skipping || disabled) {
      return;
    }
    setSkipping(true);
    try {
      await onSend("__skip__");
      onPendingChange("");
    } finally {
      setSkipping(false);
    }
  };

  const busy = sending || skipping;

  return (
    <div className="message-composer">
      <textarea
        disabled={disabled || busy}
        onChange={(event) => onPendingChange(event.target.value)}
        placeholder="Введите ваш ответ"
        rows={3}
        value={pending}
      />
      <div className="message-composer-actions">
        <button
          className="composer-skip"
          disabled={disabled || busy}
          onClick={() => void skip()}
          type="button"
          title="Пропустить этот вопрос — AI перейдёт к следующему без оценки"
        >
          {skipping ? "..." : "Пропустить"}
        </button>
        <button
          className="composer-send"
          disabled={disabled || busy || !pending.trim()}
          onClick={() => void submit()}
          type="button"
        >
          {sending ? "Отправка..." : "Отправить"}
        </button>
      </div>
    </div>
  );
}
