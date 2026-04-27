import { useMemo, useState } from "react";

import { useInterviewStore } from "@/app/store";
import { interviewApi } from "@/shared/api";
import { useTranslation } from "@/shared/i18n";
import { ChatBubble, FloatingInput, GlassButton, GlassCard } from "@/shared/ui";

export function InterviewPanel() {
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);

  const messages = useInterviewStore((state) => state.messages);
  const addMessage = useInterviewStore((state) => state.addMessage);
  const t = useTranslation();

  const typingPreview = useMemo(() => {
    if (!typing) return "";
    return t.aiIsTyping;
  }, [typing, t]);

  const send = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalized = draft.trim();
    if (!normalized) return;

    addMessage({
      id: `m-user-${Date.now()}`,
      role: "user",
      content: normalized,
      timestamp: new Date().toISOString(),
    });

    setDraft("");
    setTyping(true);

    try {
      const feedback = await interviewApi.analyzeAnswer(normalized);
      addMessage({
        id: `m-ai-${Date.now()}`,
        role: "ai",
        content: feedback,
        timestamp: new Date().toISOString(),
      });
    } catch {
      addMessage({
        id: `m-ai-${Date.now()}`,
        role: "ai",
        content: "Не удалось получить ответ модели. Проверьте доступность API ИИ.",
        timestamp: new Date().toISOString(),
      });
    } finally {
      setTyping(false);
    }
  };

  return (
    <GlassCard className="interview-panel">
      <div className="chat-feed">
        {messages.map((msg) => (
          <ChatBubble key={msg.id} role={msg.role} text={msg.content} />
        ))}
        {typing && (
          <p className="typing-dots">
            {typingPreview}
            <span>.</span>
            <span>.</span>
            <span>.</span>
          </p>
        )}
      </div>

      <form className="chat-form" onSubmit={send}>
        <FloatingInput
          label={t.yourAnswer}
          onChange={(event) => setDraft(event.target.value)}
          value={draft}
        />
        <GlassButton type="submit">{t.send}</GlassButton>
      </form>
    </GlassCard>
  );
}
