import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { interviewModuleApi } from "@/features/interview-module/api";
import { env } from "@/shared/config/env";
import {
  ChatWindow,
  ConnectionStatus,
  InterviewTopBar,
  MessageComposer,
  PracticeCodeWorkspace,
} from "@/features/interview-module/components";
import {
  useChatStore,
  useNetworkStore,
  useSessionStore,
  useTimerStore,
} from "@/features/interview-module/stores";
import type { InterviewLevel, InterviewMessage, InterviewMode, InterviewRole } from "@/features/interview-module/types";

const formatMMSS = (seconds: number) => {
  const safe = Math.max(seconds, 0);
  const mm = Math.floor(safe / 60)
    .toString()
    .padStart(2, "0");
  const ss = Math.floor(safe % 60)
    .toString()
    .padStart(2, "0");
  return `${mm}:${ss}`;
};

const MAX_WS_RECONNECT_ATTEMPTS = 8;

const toWSUrl = (path: string) => {
  if (path.startsWith("ws://") || path.startsWith("wss://")) {
    return path;
  }
  const configured = env.apiWsUrl;
  const configuredUrl = configured.startsWith("ws://") || configured.startsWith("wss://") ? new URL(configured) : null;
  const protocol = configuredUrl?.protocol.replace(":", "") || (window.location.protocol === "https:" ? "wss" : "ws");
  const host = configuredUrl?.host || window.location.host;

  const token = localStorage.getItem("realsync_token");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;

  if (!token) {
    return `${protocol}://${host}${normalizedPath}`;
  }
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
  const pendingUserMessage = useChatStore((state) => state.pendingUserMessage);
  const aiTyping = useChatStore((state) => state.aiTyping);
  const streamBuffer = useChatStore((state) => state.streamBuffer);
  const setMessages = useChatStore((state) => state.setMessages);
  const addMessage = useChatStore((state) => state.addMessage);
  const setPendingUserMessage = useChatStore((state) => state.setPendingUserMessage);
  const setAiTyping = useChatStore((state) => state.setAiTyping);
  const pushStreamChunk = useChatStore((state) => state.pushStreamChunk);
  const clearStreamBuffer = useChatStore((state) => state.clearStreamBuffer);

  const countdownSec = useTimerStore((state) => state.countdownSec);
  const autoFinishTriggered = useTimerStore((state) => state.autoFinishTriggered);
  const tick = useTimerStore((state) => state.tick);
  const configureTimer = useTimerStore((state) => state.configure);

  const wsConnected = useNetworkStore((state) => state.wsConnected);
  const reconnectAttempts = useNetworkStore((state) => state.reconnectAttempts);
  const lastError = useNetworkStore((state) => state.lastError);
  const setConnected = useNetworkStore((state) => state.setConnected);
  const setLastError = useNetworkStore((state) => state.setLastError);
  const registerReconnectAttempt = useNetworkStore((state) => state.registerReconnectAttempt);

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectAttemptRef = useRef(0);
  const [bootLoading, setBootLoading] = useState(true);
  const [bootError, setBootError] = useState<string | null>(null);
  const [timerReady, setTimerReady] = useState(false);

  const timerLabel = useMemo(() => formatMMSS(countdownSec), [countdownSec]);

  const toRole = (value: string): InterviewRole => {
    const normalized = value.toLowerCase();
    if (normalized === "backend") return "Backend";
    if (normalized === "frontend") return "Frontend";
    if (normalized === "web") return "Web";
    if (normalized === "devops") return "DevOps";
    if (normalized === "ml") return "ML";
    if (normalized === "mobile") return "Mobile";
    if (normalized === "data") return "Data";
    if (normalized === "game") return "Game";
    if (normalized === "security") return "Security";
    if (normalized === "systems") return "Systems";
    if (normalized === "enterprise") return "Enterprise";
    if (normalized === "fintech") return "Fintech";
    if (normalized === "iot") return "IoT";
    if (normalized === "management") return "Management";
    return "Backend";
  };

  const toLevel = (value: string): InterviewLevel => {
    const normalized = value.toLowerCase();
    if (normalized === "junior") return "Junior";
    if (normalized === "senior") return "Senior";
    return "Middle";
  };

  const toInterviewMode = (value?: string): InterviewMode => {
    const normalized = (value || "").toLowerCase();
    if (normalized === "theory") return "theory";
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
  }, [
    sessionId,
    currentSession,
    navigate,
    setMessages,
    setSession,
    setStatus,
    configureTimer,
  ]);

  useEffect(() => {
    if (!timerReady || bootLoading) {
      return;
    }

    const timer = window.setInterval(() => tick(), 1000);
    return () => window.clearInterval(timer);
  }, [tick, timerReady, bootLoading]);

  useEffect(() => {
    if (bootLoading || !timerReady || !autoFinishTriggered || !sessionId) {
      return;
    }
    void interviewModuleApi.finishSession(sessionId).finally(() => {
      setStatus("finished");
      navigate(`/interview/result/${sessionId}`);
    });
  }, [autoFinishTriggered, sessionId, navigate, setStatus, bootLoading, timerReady]);

  useEffect(() => {
    if (!sessionId) {
      return;
    }

    let cancelled = false;
    let reconnectTimer: number | null = null;

    const connect = async () => {
      try {
        const session = await interviewModuleApi.getSession(sessionId);
        const wsPath = session.ws_url || `/api/interviews/sessions/${sessionId}/ws`;
        const socket = new WebSocket(toWSUrl(wsPath));
        socketRef.current = socket;

        socket.onopen = () => {
          if (cancelled) {
            return;
          }
          reconnectAttemptRef.current = 0;
          setConnected(true);
          setLastError(null);
        };

        socket.onmessage = (event) => {
          if (cancelled) {
            return;
          }

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
            const mapped: InterviewMessage = {
              messageId: (msgPayload as { message_id?: string }).message_id || crypto.randomUUID(),
              sender: (msgPayload as { sender?: "ai" | "user" }).sender || "ai",
              content: (msgPayload as { content?: string }).content || "",
              topic: (msgPayload as { topic?: string }).topic,
              difficulty: (msgPayload as { difficulty?: number }).difficulty,
              createdAt:
                (msgPayload as { created_at?: string }).created_at || new Date().toISOString(),
            };
            clearStreamBuffer();
            addMessage(mapped);
            return;
          }

          if (payload.type === "ai.message.chunk") {
            const chunk = (payload.payload as { chunk?: string }).chunk || "";
            if (chunk) {
              pushStreamChunk(chunk);
            }
            return;
          }

          if (payload.type === "session.finished") {
            setStatus("finished");
            navigate(`/interview/result/${sessionId}`);
            return;
          }

          void session;
        };

        socket.onclose = () => {
          setConnected(false);
          if (cancelled) {
            return;
          }

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
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }
      socketRef.current?.close();
    };
  }, [
    addMessage,
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

  const sendMessage = async (value: string) => {
    if (!sessionId) {
      return;
    }

    const local: InterviewMessage = {
      messageId: crypto.randomUUID(),
      sender: "user",
      content: value,
      createdAt: new Date().toISOString(),
    };
    addMessage(local);
    await interviewModuleApi.sendMessage(sessionId, value);

    // If realtime channel is reconnecting, poll once for the latest AI reply.
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
    if (sessionId) {
      await interviewModuleApi.finishSession(sessionId);
    }
    setStatus("finished");
    navigate(`/interview/result/${sessionId}`);
  };

  if (bootLoading) {
    return (
      <section className="interview-session-page">
        <div className="interview-setup-card">
          <h1>Загрузка интервью</h1>
          <p>Проверяем доступ и восстанавливаем историю сессии.</p>
        </div>
      </section>
    );
  }

  if (bootError) {
    return (
      <section className="interview-session-page">
        <div className="interview-setup-card">
          <h1>Не удалось открыть сессию</h1>
          <p>{bootError}</p>
          <button className="interview-start-btn" onClick={() => navigate("/interview")} type="button">
            Вернуться к настройке интервью
          </button>
        </div>
      </section>
    );
  }

  return (
    <section className="interview-session-page">
      <InterviewTopBar
        interviewMode={interviewMode}
        level={level}
        onExit={() => void exitInterview()}
        role={role}
        timerLabel={timerLabel}
        vacancyTitle={vacancyTitle}
      />
      <ConnectionStatus
        connected={wsConnected}
        lastError={lastError}
        reconnectAttempts={reconnectAttempts}
      />
      {interviewMode === "practice" ? (
        <PracticeCodeWorkspace
          aiTyping={aiTyping}
          disabled={!sessionId}
          messages={messages}
          onSubmitCode={sendMessage}
        />
      ) : (
        <>
          <ChatWindow aiTyping={aiTyping} messages={messages} streamBuffer={streamBuffer} />
          <MessageComposer
            disabled={!sessionId}
            onPendingChange={setPendingUserMessage}
            onSend={sendMessage}
            pending={pendingUserMessage}
          />
        </>
      )}
    </section>
  );
}
