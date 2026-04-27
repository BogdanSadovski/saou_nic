import type { InterviewMessage } from "../types";
import { MessageList } from "./MessageList";
import { TypingIndicator } from "./TypingIndicator";

type Props = {
  messages: InterviewMessage[];
  aiTyping: boolean;
  streamBuffer: string;
};

export function ChatWindow({ messages, aiTyping, streamBuffer }: Props) {
  return (
    <section className="chat-window">
      <MessageList messages={messages} />
      {streamBuffer ? <div className="stream-buffer">{streamBuffer}</div> : null}
      {aiTyping ? <TypingIndicator /> : null}
    </section>
  );
}
