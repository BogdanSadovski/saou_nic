import { useState } from "react";

type Props = {
  pending: string;
  disabled?: boolean;
  onPendingChange: (value: string) => void;
  onSend: (value: string) => Promise<void>;
};

export function MessageComposer({ pending, disabled = false, onPendingChange, onSend }: Props) {
  const [sending, setSending] = useState(false);

  const submit = async () => {
    const value = pending.trim();
    if (!value || sending || disabled) {
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

  return (
    <div className="message-composer">
      <textarea
        disabled={disabled}
        onChange={(event) => onPendingChange(event.target.value)}
        placeholder="Введите ваш ответ"
        rows={3}
        value={pending}
      />
      <button disabled={disabled || sending || !pending.trim()} onClick={() => void submit()} type="button">
        {sending ? "Отправка..." : "Отправить"}
      </button>
    </div>
  );
}
