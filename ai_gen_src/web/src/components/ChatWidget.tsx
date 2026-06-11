import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { useVoiceInput } from '../hooks/useVoiceInput';
import { useChatDock } from '../hooks/useChatDock';
import { RESIZE_CORNERS, useChatResize } from '../hooks/useChatResize';

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
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const widgetRef = useRef<HTMLDivElement>(null);
  const { corner, floatPos, isDragging, onDragStart, onDragMove, onDragEnd, consumeDragClick } =
    useChatDock(widgetRef);
  const { size, isResizing, activeCorner, onResizeStart, onResizeMove, onResizeEnd } =
    useChatResize(widgetRef);

  const scrollFeedToBottom = useCallback(() => {
    const feed = feedRef.current;
    if (!feed) return;
    feed.scrollTop = feed.scrollHeight;
  }, []);

  const send = useMutation({
    mutationFn: (message: string) =>
      api<{ reply: string }>('/chat', {
        method: 'POST',
        body: JSON.stringify({ message, session_id: sessionId }),
      }),
    onSuccess: (data) => {
      setMessages((prev) => [...prev, { role: 'assistant', text: data.reply }]);
    },
    onError: (err: Error) => {
      const text = err instanceof ApiClientError ? err.message : 'Chat thất bại — thử lại.';
      setMessages((prev) => [...prev, { role: 'assistant', text }]);
    },
  });

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [open]);

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => scrollFeedToBottom());
  }, [messages, send.isPending, open, scrollFeedToBottom]);

  const voice = useVoiceInput({
    onTranscript: setInput,
    onSubmit: (raw) => {
      const msg = raw.trim();
      if (!msg || send.isPending) return;
      setInput('');
      setMessages((prev) => [...prev, { role: 'user', text: msg }]);
      send.mutate(msg);
    },
  });

  const submitMessage = useCallback(() => {
    const msg = input.trim();
    if (!msg || send.isPending) return;
    setMessages((prev) => [...prev, { role: 'user', text: msg }]);
    setInput('');
    voice.setTranscript('');
    send.mutate(msg);
  }, [input, send.isPending, send, voice]);

  const onInputKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submitMessage();
    }
  };

  const handleHeaderPointerDown = (e: React.PointerEvent<HTMLElement>) => {
    if ((e.target as HTMLElement).closest('.chat-widget__close')) return;
    if ((e.target as HTMLElement).closest('.chat-widget__resize-handle')) return;
    onDragStart(e);
  };

  const handleTogglePointerDown = (e: React.PointerEvent<HTMLButtonElement>) => {
    onDragStart(e);
  };

  const finishDrag = (e: React.PointerEvent) => {
    onDragEnd(e);
    if (!open && !consumeDragClick()) {
      setOpen(true);
    }
  };

  const handleWidgetPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    onResizeMove(e);
    onDragMove(e);
  };

  const handleWidgetPointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (onResizeEnd(e)) return;
    finishDrag(e);
  };

  const dockStyle = floatPos
    ? { left: floatPos.left, top: floatPos.top, right: 'auto', bottom: 'auto' }
    : undefined;

  return (
    <div
      ref={widgetRef}
      className={`chat-widget chat-widget--${corner}${open ? ' chat-widget--open' : ''}${isDragging ? ' chat-widget--dragging' : ''}${isResizing ? ' chat-widget--resizing' : ''}`}
      style={dockStyle}
      onPointerMove={handleWidgetPointerMove}
      onPointerUp={handleWidgetPointerUp}
      onPointerCancel={(e) => {
        onResizeEnd(e);
        finishDrag(e);
      }}
    >
      {open ? (
        <div
          className="chat-widget__panel"
          style={{ width: size.width, height: size.height, maxHeight: size.height }}
        >
          {RESIZE_CORNERS.map((corner) => (
            <div
              key={corner}
              className={`chat-widget__resize-handle chat-widget__resize-handle--${corner}${isResizing && activeCorner === corner ? ' chat-widget__resize-handle--active' : ''}`}
              role="presentation"
              aria-hidden="true"
              title="Kéo để đổi kích thước"
              onPointerDown={onResizeStart(corner)}
            />
          ))}
          <header
            className="chat-widget__header chat-widget__drag-handle"
            onPointerDown={handleHeaderPointerDown}
          >
            <div className="chat-widget__title">
              <strong>Chat OpsOne</strong>
              <span className="chat-widget__subtitle">Tra cứu metric · Admin có thể duyệt qua chat</span>
            </div>
            <button
              type="button"
              className="btn btn--ghost btn--xs chat-widget__close"
              aria-label="Thu gọn chat"
              onClick={() => setOpen(false)}
            >
              −
            </button>
          </header>

          <div className="chat-widget__feed" ref={feedRef}>
            {messages.length === 0 && (
              <p className="muted chat-widget__empty">
                Hỏi trạng thái, sự cố, metric… Admin có thể: &quot;Duyệt routing GARENA 10000&quot;
              </p>
            )}
            {messages.map((m, i) => (
              <div key={i} className={`chat-bubble chat-bubble--${m.role}`}>
                <span className="chat-bubble__role">{m.role === 'user' ? 'Bạn' : 'OpsOne'}</span>
                <div className="chat-bubble__body">{m.text}</div>
              </div>
            ))}
            {send.isPending && (
              <div className="chat-bubble chat-bubble--assistant">
                <span className="chat-bubble__role">OpsOne</span>
                <div className="chat-bubble__body chat-bubble__body--pending">Đang xử lý...</div>
              </div>
            )}
          </div>

          <div className="chat-widget__input">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => {
                voice.setTranscript(e.target.value);
                setInput(e.target.value);
              }}
              onKeyDown={onInputKeyDown}
              placeholder="Nhập câu hỏi..."
              rows={3}
            />
            <div className="chat-input__actions">
              {voice.supported && (
                <button
                  type="button"
                  className={`btn btn--mic${voice.micOn ? ' btn--mic-active' : ''}`}
                  aria-label={voice.micOn ? 'Tắt micro' : 'Bật micro — hội thoại liên tục'}
                  title={voice.micOn ? 'Bấm để tắt micro' : 'Bật micro; im lặng 2 giây sẽ gửi từng câu'}
                  onClick={voice.start}
                >
                  {voice.micOn ? 'Mic ●' : 'Mic'}
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
          {voice.supported && voice.micOn && (
            <p className="voice-hint chat-widget__voice">
              Đang nghe liên tục… im lặng 2 giây sẽ gửi. Nói &quot;tắt mic&quot; hoặc &quot;kết thúc cuộc trò chuyện&quot; để tắt.
            </p>
          )}
        </div>
      ) : (
        <button
          type="button"
          className="chat-widget__toggle btn btn--primary chat-widget__drag-handle"
          title="Kéo để đổi góc · Bấm để mở chat"
          onPointerDown={handleTogglePointerDown}
        >
          Chat
        </button>
      )}
    </div>
  );
}
