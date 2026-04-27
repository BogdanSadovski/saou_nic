import { useEffect, useRef } from "react";

import type { InterviewMessage } from "../types";

type Props = {
  messages: InterviewMessage[];
};

export function MessageList({ messages }: Props) {
  const endRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages]);

  return (
    <div className="interview-message-list" aria-live="polite">
      {messages.map((message) => (
        <article
          className={message.sender === "ai" ? "msg msg-ai" : "msg msg-user"}
          key={message.messageId}
        >
          <p>{message.content}</p>
          <small>{new Date(message.createdAt).toLocaleTimeString()}</small>
        </article>
      ))}
      <div ref={endRef} />
    </div>
  );
}
