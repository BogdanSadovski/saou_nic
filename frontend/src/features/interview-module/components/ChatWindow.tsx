import { useEffect, useRef } from "react";

import type { InterviewMessage } from "../types";
import { MessageList } from "./MessageList";
import { TypingIndicator } from "./TypingIndicator";

type Props = {
  messages: InterviewMessage[];
  aiTyping: boolean;
  streamBuffer: string;
};

/**
 * Chat container that renders both the persisted message log and the
 * still-streaming AI response. The streaming view is styled like a
 * regular AI message bubble (matching the eventual `message.ai`
 * delivery) with a blinking caret, so users see the answer appear
 * in-place rather than as a "raw text appearing in the corner".
 *
 * Auto-scrolls to the end whenever new chunks arrive so the user
 * doesn't have to chase the typing cursor on long answers.
 */
export function ChatWindow({ messages, aiTyping, streamBuffer }: Props) {
  const tailRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (streamBuffer || aiTyping) {
      tailRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [streamBuffer, aiTyping]);

  return (
    <section className="chat-window">
      <MessageList messages={messages} />

      {streamBuffer ? (
        <article className="msg msg-ai msg-streaming" aria-live="polite">
          <p>
            {streamBuffer}
            <span className="msg-caret" aria-hidden="true" />
          </p>
        </article>
      ) : null}

      {aiTyping && !streamBuffer ? <TypingIndicator /> : null}

      <div ref={tailRef} />
    </section>
  );
}
