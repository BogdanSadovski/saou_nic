import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import { PracticeCodeWorkspace } from "@/features/interview-module/components";
import { env } from "@/shared/config/env";
import {
  useChatStore,
  useNetworkStore,
  useSessionStore,
  useTimerStore,
} from "@/features/interview-module/stores";
import type { InterviewLevel, InterviewMessage, InterviewMode, InterviewRole } from "@/features/interview-module/types";
import { RsIcon as Icon } from "@/shared/ui/realsync";

const formatMMSS = (seconds: number) => {
  const safe = Math.max(seconds, 0);
  const mm = Math.floor(safe / 60).toString().padStart(2, "0");
  const ss = Math.floor(safe % 60).toString().padStart(2, "0");
  return `${mm}:${ss}`;
};

const MAX_WS_RECONNECT_ATTEMPTS = 8;

const toWSUrl = (path: string) => {
  if (path.startsWith("ws://") || path.startsWith("wss://")) {
    return path;
  }
  const configured = env.apiWsUrl;
  const configuredUrl =
    configured.startsWith("ws://") || configured.startsWith("wss://") ? new URL(configured) : null;
  const protocol = configuredUrl?.protocol.replace(":", "") || (window.location.protocol === "https:" ? "wss" : "ws");
  const host = configuredUrl?.host || window.location.host;
  const token = localStorage.getItem("realsync_token");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  if (!token) return `${protocol}://${host}${normalizedPath}`;
  const sep = normalizedPath.includes("?") ? "&" : "?";
  return `${protocol}://${host}${normalizedPath}${sep}access_token=${encodeURIComponent(token)}`;
};

export default function InterviewSessionPage() {
  const { sessionId = "" } = useParams();
  const navigate = useNavigate();

  const role = useSessionStore((state) => state.role);
  const level = useSessionStore((state) => state.level);
  const vacancyTitle = useSessionStore((state) => state.vacancyTitle);
  const interviewMode = useSessionStore((state) => state.interviewMode);
  const currentSession = useSessionStore((state) => state.sessionId);
  const setStatus = useSessionStore((state) => state.setStatus);
  const setSession = useSessionStore((state) => state.setSession);

  const messages = useChatStore((state) => state.messages);
  const aiTyping = useChatStore((state) => state.aiTyping);
  const setMessages = useChatStore((state) => state.setMessages);
  const addMessage = useChatStore((state) => state.addMessage);
  const setAiTyping = useChatStore((state) => state.setAiTyping);
  const pushStreamChunk = useChatStore((state) => state.pushStreamChunk);
  const clearStreamBuffer = useChatStore((state) => state.clearStreamBuffer);
  const applyVerdict = useChatStore((state) => state.applyVerdict);

  const countdownSec = useTimerStore((state) => state.countdownSec);
  const autoFinishTriggered = useTimerStore((state) => state.autoFinishTriggered);
  const tick = useTimerStore((state) => state.tick);
  const configureTimer = useTimerStore((state) => state.configure);

  const wsConnected = useNetworkStore((state) => state.wsConnected);
  const setConnected = useNetworkStore((state) => state.setConnected);
  const setLastError = useNetworkStore((state) => state.setLastError);
  const registerReconnectAttempt = useNetworkStore((state) => state.registerReconnectAttempt);

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectAttemptRef = useRef(0);
  const chatRef = useRef<HTMLDivElement | null>(null);
  const [bootLoading, setBootLoading] = useState(true);
  const [bootError, setBootError] = useState<string | null>(null);
  const [timerReady, setTimerReady] = useState(false);
  const [input, setInput] = useState("");

  const timerLabel = useMemo(() => formatMMSS(countdownSec), [countdownSec]);
  const low = countdownSec < 5 * 60;

  const toRole = (value: string): InterviewRole => {
    const n = value.toLowerCase();
    if (n === "backend") return "Backend";
    if (n === "frontend") return "Frontend";
    if (n === "web") return "Web";
    if (n === "devops") return "DevOps";
    if (n === "ml") return "ML";
    if (n === "mobile") return "Mobile";
    if (n === "data") return "Data";
    if (n === "game") return "Game";
    if (n === "security") return "Security";
    if (n === "systems") return "Systems";
    if (n === "enterprise") return "Enterprise";
    if (n === "fintech") return "Fintech";
    if (n === "iot") return "IoT";
    if (n === "management") return "Management";
    return "Backend";
  };
  const toLevel = (value: string): InterviewLevel => {
    const n = value.toLowerCase();
    if (n === "junior") return "Junior";
    if (n === "senior") return "Senior";
    return "Middle";
  };
  const toInterviewMode = (value?: string): InterviewMode => {
    const v = (value || "").toLowerCase();
    if (v === "theory") return "theory";
    if (v === "softskills") return "softskills";
    return "practice";
  };

  useEffect(() => {
    if (!sessionId) {
      navigate("/interview", { replace: true });
      return;
    }
    setBootLoading(true);
    setBootError(null);
    setTimerReady(false);

    if (!currentSession || currentSession !== sessionId) {
      void interviewModuleApi
        .getSession(sessionId)
        .then((session) => {
          const endsAtMs = Date.parse(session.expires_at);
          const remainingSec = Number.isFinite(endsAtMs)
            ? Math.max(0, Math.floor((endsAtMs - Date.now()) / 1000))
            : 0;
          setSession({
            sessionId,
            role: toRole(session.role),
            level: toLevel(session.level),
            vacancyTitle: session.vacancy_title || session.role,
            vacancyCategory: toRole(session.vacancy_category || session.role),
            interviewMode: toInterviewMode(session.interview_mode),
            focusAreas: session.focus_areas || [],
            primarySkills: session.primary_skills || [],
            theoryFocus: session.theory_focus || [],
            practiceFocus: session.practice_focus || [],
            startedAt: session.started_at,
            endsAt: session.expires_at,
          });
          configureTimer(remainingSec);
          setTimerReady(true);
          setStatus(session.status === "finished" ? "finished" : "active");
          return interviewModuleApi.getMessages(sessionId);
        })
        .then((loadedMessages) => {
          setMessages(loadedMessages);
          setBootLoading(false);
        })
        .catch((error: unknown) => {
          const message = error instanceof Error ? error.message : "Не удалось загрузить интервью-сессию";
          setBootError(message);
          setBootLoading(false);
        });
    } else {
      setTimerReady(true);
      void interviewModuleApi
        .getMessages(sessionId)
        .then((loadedMessages) => {
          setMessages(loadedMessages);
          setBootLoading(false);
        })
        .catch((error: unknown) => {
          const message = error instanceof Error ? error.message : "Не удалось загрузить сообщения сессии";
          setBootError(message);
          setBootLoading(false);
        });
    }
  }, [sessionId, currentSession, navigate, setMessages, setSession, setStatus, configureTimer]);

  useEffect(() => {
    if (!timerReady || bootLoading) return;
    const lastMsg = messages[messages.length - 1];
    const awaitingAI = aiTyping || (lastMsg !== undefined && lastMsg.sender === "user");
    if (awaitingAI) return;
    const timer = window.setInterval(() => tick(), 1000);
    return () => window.clearInterval(timer);
  }, [tick, timerReady, bootLoading, aiTyping, messages]);

  useEffect(() => {
    if (bootLoading || !timerReady || !autoFinishTriggered || !sessionId) return;
    void interviewModuleApi.finishSession(sessionId).finally(() => {
      setStatus("finished");
      navigate(`/interview/result/${sessionId}`);
    });
  }, [autoFinishTriggered, sessionId, navigate, setStatus, bootLoading, timerReady]);

  useEffect(() => {
    if (!sessionId) return;
    let cancelled = false;
    let reconnectTimer: number | null = null;

    const connect = async () => {
      try {
        const session = await interviewModuleApi.getSession(sessionId);
        const wsPath = session.ws_url || `/api/interviews/sessions/${sessionId}/ws`;
        const socket = new WebSocket(toWSUrl(wsPath));
        socketRef.current = socket;

        socket.onopen = () => {
          if (cancelled) return;
          reconnectAttemptRef.current = 0;
          setConnected(true);
          setLastError(null);
        };

        socket.onmessage = (event) => {
          if (cancelled) return;
          let payload: {
            type: string;
            payload:
              | InterviewMessage
              | {
                  message_id?: string;
                  sender?: "ai" | "user";
                  content?: string;
                  topic?: string;
                  difficulty?: number;
                  created_at?: string;
                  chunk?: string;
                  typing?: boolean;
                };
          };
          try {
            payload = JSON.parse(event.data) as typeof payload;
          } catch {
            setLastError("Некорректный realtime payload");
            return;
          }

          if (payload.type === "ai.typing.started") {
            setAiTyping(true);
            return;
          }
          if (payload.type === "ai.typing.stopped") {
            setAiTyping(false);
            return;
          }
          if (payload.type === "message.ai") {
            const msgPayload = payload.payload;
            const content = (msgPayload as { content?: string }).content || "";
            const topic = (msgPayload as { topic?: string }).topic;
            const difficulty = (msgPayload as { difficulty?: number }).difficulty;
            const mapped: InterviewMessage = {
              messageId: (msgPayload as { message_id?: string }).message_id || crypto.randomUUID(),
              sender: (msgPayload as { sender?: "ai" | "user" }).sender || "ai",
              content,
              topic,
              difficulty,
              createdAt: (msgPayload as { created_at?: string }).created_at || new Date().toISOString(),
            };
            clearStreamBuffer();
            addMessage(mapped);
            return;
          }
          if (payload.type === "session.timer.adjusted") {
            const adj = payload.payload as { added_seconds?: number };
            const added = Math.max(0, Math.round(adj.added_seconds ?? 0));
            if (added > 0) {
              useTimerStore.setState((s) => ({ countdownSec: s.countdownSec + added }));
            }
            return;
          }
          if (payload.type === "ai.message.chunk") {
            const chunk = (payload.payload as { chunk?: string }).chunk || "";
            if (chunk) pushStreamChunk(chunk);
            return;
          }
          if (payload.type === "message.user.evaluated") {
            const ev = payload.payload as { message_id?: string; verdict?: string; verdict_reason?: string };
            const allowed = ["correct", "partial", "wrong", "skipped", "off_topic"] as const;
            if (
              ev.message_id &&
              ev.verdict &&
              (allowed as readonly string[]).includes(ev.verdict)
            ) {
              applyVerdict(ev.message_id, ev.verdict as (typeof allowed)[number], ev.verdict_reason);
            }
            return;
          }
          if (payload.type === "session.finished") {
            setStatus("finished");
            navigate(`/interview/result/${sessionId}`);
            return;
          }
        };

        socket.onclose = () => {
          setConnected(false);
          if (cancelled) return;
          reconnectAttemptRef.current += 1;
          const attempt = reconnectAttemptRef.current;
          if (attempt > MAX_WS_RECONNECT_ATTEMPTS) {
            setLastError("Realtime канал недоступен. Продолжаем в HTTP режиме, обновите страницу позже.");
            return;
          }
          registerReconnectAttempt();
          const delay = Math.min(5000, 600 * attempt);
          reconnectTimer = window.setTimeout(() => {
            void connect();
          }, delay);
        };

        socket.onerror = () => {
          setLastError("Ошибка realtime-соединения");
        };
      } catch {
        setLastError("Не удалось подключить realtime канал");
      }
    };

    void connect();
    return () => {
      cancelled = true;
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
      socketRef.current?.close();
    };
  }, [
    addMessage,
    applyVerdict,
    clearStreamBuffer,
    navigate,
    pushStreamChunk,
    registerReconnectAttempt,
    sessionId,
    setAiTyping,
    setConnected,
    setLastError,
    setStatus,
  ]);

  useEffect(() => {
    chatRef.current?.scrollTo({ top: chatRef.current.scrollHeight, behavior: "smooth" });
  }, [messages, aiTyping]);

  const sendMessage = async (value: string) => {
    if (!sessionId || !value.trim()) return;
    const local: InterviewMessage = {
      messageId: crypto.randomUUID(),
      sender: "user",
      content: value,
      createdAt: new Date().toISOString(),
    };
    addMessage(local);
    setInput("");
    await interviewModuleApi.sendMessage(sessionId, value);
    if (!wsConnected) {
      for (let i = 0; i < 5; i++) {
        await new Promise((resolve) => window.setTimeout(resolve, 450));
        const latest = await interviewModuleApi.getMessages(sessionId);
        const hasAIReply = latest.some((m) => m.sender === "ai" && m.createdAt >= local.createdAt);
        if (hasAIReply) {
          setMessages(latest);
          break;
        }
      }
    }
  };

  const exitInterview = async () => {
    if (sessionId) await interviewModuleApi.finishSession(sessionId);
    setStatus("finished");
    navigate(`/interview/result/${sessionId}`);
  };

  if (bootLoading) {
    return (
      <main className="page">
        <h1 className="expr-headline"><span className="ital">Загрузка</span> интервью</h1>
        <p className="muted">Проверяем доступ и восстанавливаем историю сессии.</p>
      </main>
    );
  }

  if (bootError) {
    return (
      <main className="page">
        <h1 className="expr-headline"><span className="ital">Не удалось</span> открыть сессию</h1>
        <p className="muted">{bootError}</p>
        <button className="btn btn--ghost" onClick={() => navigate("/interview")} type="button">
          Вернуться к настройке интервью
        </button>
      </main>
    );
  }

  const step = messages.filter((m) => m.sender === "ai").length;

  return (
    <section className="session" data-screen-label="05 Interview Session">
      <div className="session-top">
        <div className="session-status"></div>
        <div className="session-meta">
          <strong>
            {interviewMode === "softskills" ? "Софт-скиллы" : `${role} · ${vacancyTitle}`}
          </strong>
          <span className="tag tag--lime">{(interviewMode || "practice").toUpperCase()}</span>
          {interviewMode !== "softskills" && (
            <span className="tag">{level?.toUpperCase()}</span>
          )}
          <span className="mono" style={{ fontSize: 11, color: "var(--muted)" }}>
            SESSION · #{sessionId.slice(0, 6)} · WS {wsConnected ? "connected" : "reconnecting"}
          </span>
        </div>
        <div className={`session-timer ${low ? "is-low" : ""}`}>{timerLabel}</div>
        <button className="btn btn--ghost btn--sm" onClick={() => void exitInterview()} type="button">
          Завершить интервью
        </button>
      </div>

      <div className="sysbar">
        <span><span className="dot"></span><span className="k">mode</span><span className="v">{interviewMode}</span></span>
        {interviewMode !== "softskills" && (
          <>
            <span><span className="k">role</span><span className="v">{role}</span></span>
            <span><span className="k">level</span><span className="v">{level}</span></span>
          </>
        )}
        {interviewMode === "softskills" && (
          <span><span className="k">scorer</span><span className="v">rubert-tiny2 + ml</span></span>
        )}
        <span><span className="k">messages</span><span className="v">{messages.length}</span></span>
      </div>

      {/* В practice-режиме чат — это история диалога с интервьюером. Пока
          сообщений нет, прячем его целиком: задание само рендерится в
          PracticeCodeWorkspace ниже. В theory-режиме чат остаётся всегда. */}
      {(interviewMode !== "practice" || messages.length > 0 || aiTyping) && (
        <div
          className="chat"
          ref={chatRef}
          style={
            interviewMode === "practice"
              ? { maxHeight: 280, minHeight: 0, overflowY: "auto" }
              : { maxHeight: 480, overflowY: "auto" }
          }
        >
          {messages.map((m, i) => (
            <div className={`msg ${m.sender}`} key={m.messageId || i}>
              <div className="msg-avatar">{m.sender === "ai" ? "AI" : "СБ"}</div>
              <div>
                <div className="msg-bubble">{m.content}</div>
                <div className="msg-meta">{m.topic || "reply"} · {m.verdict || ""}</div>
              </div>
            </div>
          ))}
          {aiTyping && (
            <div className="msg ai">
              <div className="msg-avatar">AI</div>
              <div>
                <div className="msg-bubble">
                  <span className="typing-dot"><i></i><i></i><i></i></span>
                </div>
                <div className="msg-meta">генерирует ответ…</div>
              </div>
            </div>
          )}
        </div>
      )}

      {interviewMode === "practice" ? (
        <PracticeCodeWorkspace
          aiTyping={aiTyping}
          messages={messages}
          onSubmitCode={(payload) => sendMessage(payload)}
          disabled={!wsConnected}
        />
      ) : (
        <div className="composer">
          <textarea
            placeholder="Напишите ответ… (Cmd+Enter — отправить)"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if ((e.metaKey || e.ctrlKey) && e.key === "Enter") void sendMessage(input);
            }}
          />
          <button className="btn btn--accent composer-send" onClick={() => void sendMessage(input)} type="button">
            Отправить <Icon name="arrow" size={14} />
          </button>
        </div>
      )}

      <div className="row-between mono" style={{ fontSize: 11, color: "var(--muted)" }}>
        <span>Вопрос {step} · role: {role}</span>
        <span>Mode: {interviewMode} · auto-pace: on</span>
      </div>
    </section>
  );
}
