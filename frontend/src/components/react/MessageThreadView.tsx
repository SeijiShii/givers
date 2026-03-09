import { useState, useEffect, useRef } from "react";
import {
  getThread,
  getThreadMessages,
  sendMessage,
  getMe,
  type MessageThread,
  type Message,
} from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  threadId: string;
  locale: Locale;
  basePath: string;
}

export default function MessageThreadView({
  threadId,
  locale,
  basePath,
}: Props) {
  const [thread, setThread] = useState<MessageThread | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [body, setBody] = useState("");
  const [currentUserId, setCurrentUserId] = useState<string>("");
  const [error, setError] = useState<string>("");
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getMe()
      .then((user) => {
        if (user) setCurrentUserId(user.id);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    setLoading(true);
    Promise.all([getThread(threadId), getThreadMessages(threadId, 50)])
      .then(([t, res]) => {
        setThread(t);
        // API returns newest first, reverse for chat display
        setMessages(res.messages.reverse());
        setNextCursor(res.next_cursor);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [threadId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const loadOlder = () => {
    if (!nextCursor) return;
    getThreadMessages(threadId, 50, nextCursor)
      .then((res) => {
        setMessages((prev) => [...res.messages.reverse(), ...prev]);
        setNextCursor(res.next_cursor);
      })
      .catch(() => {});
  };

  const handleSend = () => {
    if (!body.trim() || sending) return;
    setSending(true);
    sendMessage(threadId, body.trim())
      .then((msg) => {
        setMessages((prev) => [...prev, msg]);
        setBody("");
      })
      .catch((err) => setError(err.message))
      .finally(() => setSending(false));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (loading) return <p>...</p>;
  if (error) return <p className="error-message">{error}</p>;
  if (!thread) return <p>{t(locale, "messages.empty")}</p>;

  const formatTime = (iso: string) => {
    const d = new Date(iso);
    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, "0")}`;
  };

  return (
    <div className="message-thread-view">
      <div className="thread-view-header">
        <a href={`${basePath}/messages`} className="back-link">
          &larr; {t(locale, "messages.title")}
        </a>
        <h1>{thread.subject}</h1>
        <div className="thread-view-meta">
          <span>
            {thread.host_name} &rarr; {thread.participant_name}
          </span>
          {!thread.active && (
            <span className="thread-closed-badge">
              {t(locale, "messages.threadClosed")}
            </span>
          )}
        </div>
      </div>

      <div className="message-list">
        {nextCursor && (
          <button
            type="button"
            className="btn btn-secondary load-older"
            onClick={loadOlder}
          >
            {t(locale, "messages.loadOlder")}
          </button>
        )}

        {messages.map((msg) => {
          const isMe = msg.sender_id === currentUserId;
          return (
            <div
              key={msg.id}
              className={`message-bubble ${isMe ? "sent" : "received"}`}
            >
              <div className="message-sender">{msg.sender_name}</div>
              <div className="message-body">{msg.body}</div>
              <div className="message-time">{formatTime(msg.created_at)}</div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      {thread.active ? (
        <div className="message-compose">
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t(locale, "messages.body")}
            rows={2}
            disabled={sending}
          />
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleSend}
            disabled={sending || !body.trim()}
          >
            {t(locale, "messages.send")}
          </button>
        </div>
      ) : (
        <div className="thread-closed-banner">
          {t(locale, "messages.threadClosed")}
        </div>
      )}
    </div>
  );
}
