import { cn } from "@/shared/lib/cn";

type ChatBubbleProps = {
  role: "ai" | "user";
  text: string;
};

export function ChatBubble({ role, text }: ChatBubbleProps) {
  return (
    <article
      className={cn(
        "chat-bubble",
        role === "ai" ? "chat-bubble-ai" : "chat-bubble-user",
      )}
    >
      {text}
    </article>
  );
}
