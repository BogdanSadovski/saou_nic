import { useEffect, useState } from "react";

/**
 * InterviewerPreparing — заметная заглушка для самых первых секунд
 * сессии, пока бэкенд просыпается, генерирует первый вопрос и
 * стримит его обратно по WS.
 *
 * Что починили:
 *   • до этого, если пользователь успевал зайти на сессию раньше,
 *     чем летит `ai.typing.started`, экран был пустым 10–30 сек;
 *   • особенно больно в practice — там без задания нечего рисовать
 *     в редакторе.
 *
 * Решение — анимированный AI-баббл, который сам сменяет 3 фазы:
 *   0–2 c     «Подключаюсь к интервьюеру…»
 *   2–6 c     «Изучаю вашу вакансию и резюме…»
 *   6+ c      «Готовлю первый вопрос…»
 * + статичный typing-dot, чтобы было видно, что мы живы.
 *
 * Компонент не делает сетевых вызовов — это чисто UX-плейсхолдер.
 * Как только в чате появится первое AI-сообщение, страница перестаёт
 * его рендерить.
 */
type Props = {
  variant?: "chat" | "card";
  mode?: "practice" | "theory" | "softskills";
};

const PHASES = [
  { atSec: 0, text: "Подключаюсь к интервьюеру…" },
  { atSec: 2, text: "Изучаю вашу вакансию и профиль…" },
  { atSec: 6, text: "Готовлю первый вопрос…" },
  { atSec: 14, text: "Free-tier LLM просыпается, ещё пару секунд…" },
];

export function InterviewerPreparing({ variant = "chat", mode = "practice" }: Props) {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const startedAt = Date.now();
    const id = window.setInterval(() => {
      setElapsed(Math.floor((Date.now() - startedAt) / 1000));
    }, 500);
    return () => window.clearInterval(id);
  }, []);

  const phase = PHASES.reduce(
    (acc, p) => (elapsed >= p.atSec ? p : acc),
    PHASES[0],
  );

  const hint =
    mode === "practice"
      ? "Сейчас появится кодинг-задача с тест-кейсами."
      : mode === "softskills"
        ? "Сейчас придёт первый поведенческий вопрос по STAR."
        : "Сейчас придёт первый теоретический вопрос.";

  if (variant === "card") {
    return (
      <div className="prep-card" role="status" aria-live="polite">
        <div className="prep-card-head">
          <span className="prep-card-avatar">AI</span>
          <div>
            <strong>{phase.text}</strong>
            <div className="prep-card-hint">{hint}</div>
          </div>
        </div>
        <span className="typing-dot" aria-hidden="true">
          <i />
          <i />
          <i />
        </span>
      </div>
    );
  }

  return (
    <div className="msg ai prep-msg" role="status" aria-live="polite">
      <div className="msg-avatar">AI</div>
      <div>
        <div className="msg-bubble">
          <strong style={{ display: "block", marginBottom: 4 }}>{phase.text}</strong>
          <span className="muted" style={{ fontSize: 12 }}>{hint}</span>
          <div style={{ marginTop: 8 }}>
            <span className="typing-dot" aria-hidden="true">
              <i />
              <i />
              <i />
            </span>
          </div>
        </div>
        <div className="msg-meta">подготовка · {elapsed}s</div>
      </div>
    </div>
  );
}
