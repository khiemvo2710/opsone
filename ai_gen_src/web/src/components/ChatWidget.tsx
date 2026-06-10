import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api } from '../api/client';
import { useVoiceInput } from '../hooks/useVoiceInput';

interface ChatMessage {
  role: 'user' | 'assistant';
  text: string;
}

export function ChatWidget() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sessionId] = useState(() => crypto.randomUUID());
  const feedRef = useRef<HTMLDivElement>(null);

  const scrollFeedToBottom = useCallback(() => {
    const feed = feedRef.current;
    if (!feed) return;
    feed.scrollTop = feed.scrollHeight;
  }, []);

  const voice = useVoiceInput((text) => {
    setInput(text);
  });

  const send = useMutation({
    mutationFn: (message: string) =>
      api<{ reply: string }>('/chat', {
        method: 'POST',
        body: JSON.stringify({ message, session_id: sessionId }),
      }),
    onSuccess: (data) => {
      setMessages((prev) => [...prev, { role: 'assistant', text: data.reply }]);
    },
  });

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => scrollFeedToBottom());
  }, [messages, send.isPending, open, scrollFeedToBottom]);

  const submitMessage = useCallback(() => {
    const msg = (voice.state === 'listening' ? voice.transcript : input).trim();
    if (!msg || send.isPending) return;
    setMessages((prev) => [...prev, { role: 'user', text: msg }]);
    setInput('');
    voice.setTranscript('');
    send.mutate(msg);
  }, [input, send, voice]);

  const onInputKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submitMessage();
    }
  };

  return (
    <div className={`chat-widget${open ? ' chat-widget--open' : ''}`}>
      {open ? (
        <div className="chat-widget__panel">
          <header className="chat-widget__header">
            <strong>Chat OpsOne</strong>
            <button
              type="button"
              className="btn btn--ghost btn--xs chat-widget__close"
              aria-label="Thu gọn chat"
              onClick={() => setOpen(false)}
            >
              −
            </button>
          </header>
          <p className="chat-widget__hint muted">Enter gửi · Shift+Enter xuống dòng</p>

          <div className="chat-widget__feed" ref={feedRef}>
            {messages.length === 0 && (
              <p className="muted chat-widget__empty">
                Ví dụ: &quot;Trạng thái hệ thống&quot;, &quot;Sự cố gần nhất&quot;
              </p>
            )}
            {messages.map((m, i) => (
              <div key={i} className={`chat-bubble chat-bubble--${m.role}`}>
                {m.text}
              </div>
            ))}
            {send.isPending && (
              <div className="chat-bubble chat-bubble--assistant">Đang xử lý...</div>
            )}
          </div>

          <div className="chat-widget__input">
            <textarea
              value={voice.state === 'listening' ? voice.transcript : input}
              onChange={(e) => {
                voice.setTranscript(e.target.value);
                setInput(e.target.value);
              }}
              onKeyDown={onInputKeyDown}
              placeholder="Nhập câu hỏi..."
              rows={2}
            />
            <div className="chat-input__actions">
              {voice.supported && (
                <button
                  type="button"
                  className={`btn btn--mic${voice.state === 'listening' ? ' btn--mic-active' : ''}`}
                  aria-label="Micro"
                  onClick={voice.start}
                >
                  Mic
                </button>
              )}
              <button
                type="button"
                className="btn btn--primary btn--xs"
                disabled={send.isPending}
                onClick={submitMessage}
              >
                Gửi
              </button>
            </div>
          </div>
          {voice.state === 'listening' && (
            <p className="voice-hint chat-widget__voice">Đang nghe... nói tiếng Việt</p>
          )}
        </div>
      ) : (
        <button
          type="button"
          className="chat-widget__toggle btn btn--primary"
          onClick={() => setOpen(true)}
        >
          Chat
        </button>
      )}
    </div>
  );
}
