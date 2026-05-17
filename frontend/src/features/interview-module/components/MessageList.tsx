import { useEffect, useRef } from "react";

import type { InterviewAnswerVerdict, InterviewMessage } from "../types";

type Props = {
  messages: InterviewMessage[];
};

const VERDICT_META: Record<
  InterviewAnswerVerdict,
  { glyph: string; label: string; className: string }
> = {
  correct: { glyph: "✓", label: "Верно", className: "verdict-correct" },
  partial: { glyph: "~", label: "Частично", className: "verdict-partial" },
  wrong: { glyph: "✕", label: "Неверно", className: "verdict-wrong" },
  skipped: { glyph: "→", label: "Пропущен", className: "verdict-skipped" },
  off_topic: { glyph: "!", label: "Не по теме", className: "verdict-offtopic" },
};

/**
 * Renders the interview transcript. User messages display an AI
 * verdict badge once the next-turn payload comes back from the
 * backend (correct / partial / wrong / skipped / off_topic), with
 * the human-readable reason as a tooltip — same pattern as a real
 * interviewer giving immediate feedback.
 */
export function MessageList({ messages }: Props) {
  const endRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages]);

  return (
    <div className="interview-message-list" aria-live="polite">
      {messages.map((message) => {
        const verdict = message.sender === "user" ? message.verdict : undefined;
        const meta = verdict ? VERDICT_META[verdict] : undefined;
        return (
          <article
            className={message.sender === "ai" ? "msg msg-ai" : "msg msg-user"}
            key={message.messageId}
          >
            <p>{message.content}</p>
            <div className="msg-foot">
              <small>{new Date(message.createdAt).toLocaleTimeString()}</small>
              {meta ? (
                <span
                  className={`verdict-badge ${meta.className}`}
                  title={message.verdictReason || meta.label}
                >
                  <span className="verdict-glyph" aria-hidden="true">
                    {meta.glyph}
                  </span>
                  <span>{meta.label}</span>
                </span>
              ) : null}
            </div>
          </article>
        );
      })}
      <div ref={endRef} />
    </div>
  );
}
