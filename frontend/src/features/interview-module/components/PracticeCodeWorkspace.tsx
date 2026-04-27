import { useEffect, useMemo, useState } from "react";

import type { InterviewMessage } from "../types";

type Props = {
  disabled?: boolean;
  aiTyping: boolean;
  messages: InterviewMessage[];
  onSubmitCode: (payload: string) => Promise<void>;
};

const taskDraftKey = (task: string) => `practice:draft:${task.slice(0, 120).toLowerCase()}`;

type PracticeTurn = {
  feedback: string | null;
  errors: string | null;
  hint: string | null;
  task: string;
};

const parsePracticeTurn = (content: string): PracticeTurn => {
  const text = (content || "").trim();
  if (!text) {
    return {
      feedback: null,
      errors: null,
      hint: null,
      task: "Ожидаем первое задание от интервьюера...",
    };
  }

  const feedbackMarker = "[FEEDBACK]";
  const errorsMarker = "[ERRORS]";
  const hintMarker = "[HINT]";
  const taskMarker = "[NEXT_TASK]";
  const feedbackIdx = text.indexOf(feedbackMarker);
  const errorsIdx = text.indexOf(errorsMarker);
  const hintIdx = text.indexOf(hintMarker);
  const taskIdx = text.indexOf(taskMarker);

  if (feedbackIdx === -1 || taskIdx === -1 || taskIdx < feedbackIdx) {
    return {
      feedback: null,
      errors: null,
      hint: null,
      task: text,
    };
  }

  const feedbackEnd = errorsIdx > feedbackIdx ? errorsIdx : hintIdx > feedbackIdx ? hintIdx : taskIdx;
  const errorsEnd = hintIdx > errorsIdx ? hintIdx : taskIdx;
  const hintEnd = taskIdx;

  const feedback = text.slice(feedbackIdx + feedbackMarker.length, feedbackEnd).trim();
  const errors = errorsIdx >= 0 ? text.slice(errorsIdx + errorsMarker.length, errorsEnd).trim() : "";
  const hint = hintIdx >= 0 ? text.slice(hintIdx + hintMarker.length, hintEnd).trim() : "";
  const task = text
    .slice(taskIdx + taskMarker.length)
    .trim();

  return {
    feedback: feedback || null,
    errors: errors || null,
    hint: hint || null,
    task: task || "Напишите решение и нажмите «Проверить решение».",
  };
};

export function PracticeCodeWorkspace({ disabled = false, aiTyping, messages, onSubmitCode }: Props) {
  const [code, setCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<"idle" | "saved">("idle");
  const [copiedTask, setCopiedTask] = useState(false);

  const latestTurn = useMemo(() => {
    const aiMessages = messages.filter((message) => message.sender === "ai");
    const latest = aiMessages[aiMessages.length - 1];
    return parsePracticeTurn(latest?.content || "");
  }, [messages]);

  const currentTask = latestTurn.task;
  const lastErrors = latestTurn.errors;
  const lastHint = latestTurn.hint;

  const lastFeedback = useMemo(() => {
    if (latestTurn.feedback) {
      return latestTurn.feedback;
    }
    const aiMessages = messages.filter((message) => message.sender === "ai");
    if (aiMessages.length < 2) {
      return null;
    }
    const previous = parsePracticeTurn(aiMessages[aiMessages.length - 2]?.content || "");
    return previous.feedback;
  }, [latestTurn.feedback, messages]);

  const canSubmit = !disabled && !submitting && code.trim().length > 0;
  const canRequestControl = !disabled && !submitting && !aiTyping;

  useEffect(() => {
    const saved = localStorage.getItem(taskDraftKey(currentTask));
    if (saved) {
      setCode(saved);
      setSaveState("saved");
      return;
    }
    setCode("");
    setSaveState("idle");
  }, [currentTask]);

  useEffect(() => {
    if (!currentTask || !code.trim()) {
      return;
    }
    localStorage.setItem(taskDraftKey(currentTask), code);
    setSaveState("saved");
  }, [code, currentTask]);

  const submit = async () => {
    if (!canSubmit) {
      return;
    }

    const payload = [
      "[LIVE_CODE_SUBMISSION]",
      "Текущее задание:",
      currentTask,
      "",
      "Решение кандидата:",
      code.trim(),
    ].join("\n");

    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode(payload);
      localStorage.removeItem(taskDraftKey(currentTask));
      setCode("");
      setSaveState("idle");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось отправить решение. Повторите попытку.";
      setSubmitError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const copyTask = async () => {
    try {
      await navigator.clipboard.writeText(currentTask);
      setCopiedTask(true);
      window.setTimeout(() => setCopiedTask(false), 1200);
    } catch {
      setSubmitError("Не удалось скопировать условие задачи.");
    }
  };

  const clearDraft = () => {
    localStorage.removeItem(taskDraftKey(currentTask));
    setCode("");
    setSaveState("idle");
  };

  const requestHint = async () => {
    if (!canRequestControl) {
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode("[HINT_REQUEST]");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось получить подсказку.";
      if (/busy|409/i.test(message)) {
        await new Promise((resolve) => window.setTimeout(resolve, 700));
        try {
          await onSubmitCode("[HINT_REQUEST]");
          return;
        } catch (retryError) {
          const retryMessage = retryError instanceof Error ? retryError.message : message;
          setSubmitError(retryMessage);
        }
      } else {
        setSubmitError(message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const requestTests = async () => {
    if (!canRequestControl) {
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      await onSubmitCode("[TEST_CASE_REQUEST]");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Не удалось получить тест-кейсы.";
      if (/busy|409/i.test(message)) {
        await new Promise((resolve) => window.setTimeout(resolve, 700));
        try {
          await onSubmitCode("[TEST_CASE_REQUEST]");
          return;
        } catch (retryError) {
          const retryMessage = retryError instanceof Error ? retryError.message : message;
          setSubmitError(retryMessage);
        }
      } else {
        setSubmitError(message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="practice-workspace">
      <article className="practice-task-card">
        <header>
          <h2>Задание</h2>
          {aiTyping ? <span className="practice-status">Проверяем и готовим следующий шаг...</span> : null}
        </header>
        <pre className="practice-task-text">{currentTask}</pre>
        <div className="practice-task-actions">
          <button className="practice-secondary-btn" onClick={() => void copyTask()} type="button">
            {copiedTask ? "Скопировано" : "Копировать задание"}
          </button>
        </div>
      </article>

      <article className="practice-editor-card">
        <header>
          <h3>Редактор кода</h3>
          <small>Вставьте решение и нажмите «Проверить»</small>
        </header>
        <div className="practice-editor-toolbar">
          <span>{code.length} символов</span>
          <span>{saveState === "saved" ? "Черновик сохранен" : "Черновик не сохранен"}</span>
        </div>
        <textarea
          className="practice-code-editor"
          disabled={disabled || submitting}
          onChange={(event) => setCode(event.target.value)}
          onKeyDown={(event) => {
            if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
              event.preventDefault();
              void submit();
            }
          }}
          placeholder="// Напишите решение здесь"
          rows={16}
          spellCheck={false}
          value={code}
        />
        <button className="practice-submit-btn" disabled={!canSubmit} onClick={() => void submit()} type="button">
          {submitting ? "Проверка..." : "Проверить решение"}
        </button>
        <div className="practice-quick-actions">
          <button className="practice-secondary-btn" disabled={!canRequestControl} onClick={() => void requestHint()} type="button">
            Нужна подсказка
          </button>
          <button className="practice-secondary-btn" disabled={!canRequestControl} onClick={() => void requestTests()} type="button">
            Показать тест-кейсы
          </button>
          <button className="practice-secondary-btn" disabled={disabled || submitting || aiTyping || !code.trim()} onClick={clearDraft} type="button">
            Очистить редактор
          </button>
        </div>
        <small className="practice-hotkey-hint">Горячая клавиша отправки: Ctrl/Cmd + Enter</small>
      </article>

      {lastFeedback ? (
        <article className="practice-feedback-card">
          <h3>Ответ интервьюера</h3>
          <p>{lastFeedback}</p>
        </article>
      ) : null}

      {lastErrors ? (
        <article className="practice-feedback-card practice-feedback-error">
          <h3>Найденные ошибки</h3>
          <p>{lastErrors}</p>
        </article>
      ) : null}

      {lastHint ? (
        <article className="practice-feedback-card">
          <h3>Подсказка</h3>
          <p>{lastHint}</p>
        </article>
      ) : null}

      {submitError ? (
        <article className="practice-feedback-card practice-feedback-error">
          <h3>Ошибка проверки</h3>
          <p>{submitError}</p>
        </article>
      ) : null}
    </section>
  );
}
