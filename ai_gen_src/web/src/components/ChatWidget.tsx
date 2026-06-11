import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import { useVoiceInput, primeSpeechRecognition, VOICE_WAKE_WORD } from '../hooks/useVoiceInput';
import { useOpsOneWake } from '../hooks/useOpsOneWake';
import { useChatDock } from '../hooks/useChatDock';
import { RESIZE_CORNERS, useChatResize } from '../hooks/useChatResize';
import { formatChatTime } from '../utils/datetimeLocal';
import { ASSISTANT_NAME } from '../utils/assistantIdentity';
import { userBubbleLabel, applyProfileUpdate, inferProfileFromSingleMessage, profileChanged, type ChatUserProfile } from '../utils/chatUserProfile';
import { CHAT_INTRO_MESSAGE } from '../utils/chatIntro';
import {
  buildOnboardingReply,
  buildProfileUpdateReply,
  isLikelyOpsQuery,
  isSkipOnboarding,
  mergeProfileFromUserTexts,
  onboardingFinished,
  shouldHandleAsProfileUpdate,
} from '../utils/chatOnboarding';
import { ChatAvatar } from './ChatAvatar';

interface ChatMessage {
  role: 'user' | 'assistant';
  text: string;
  at: number;
}

function appendMessage(role: ChatMessage['role'], text: string): ChatMessage {
  return { role, text, at: Date.now() };
}

function ChatBubbleRow({
  role,
  text,
  at,
  pending = false,
  userProfile,
  sessionSeed,
  userDisplayName,
}: {
  role: ChatMessage['role'];
  text: string;
  at?: number;
  pending?: boolean;
  userProfile?: ChatUserProfile;
  sessionSeed?: string;
  userDisplayName?: string;
}) {
  const isUser = role === 'user';
  const name = isUser ? userDisplayName || 'Bạn' : ASSISTANT_NAME;
  const timeLabel = at != null ? formatChatTime(at) : '';

  return (
    <div className={`chat-bubble chat-bubble--${role}`}>
      <ChatAvatar role={role} userProfile={userProfile} sessionSeed={sessionSeed} />
      <div className="chat-bubble__content">
        <div className="chat-bubble__meta">
          <span className="chat-bubble__name">{name}</span>
          {timeLabel && (
            <time className="chat-bubble__time" dateTime={new Date(at!).toISOString()}>
              {timeLabel}
            </time>
          )}
        </div>
        <div className={`chat-bubble__body${pending ? ' chat-bubble__body--pending' : ''}`}>{text}</div>
      </div>
    </div>
  );
}

export function ChatWidget() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sessionId, setSessionId] = useState(() => crypto.randomUUID());
  const sessionIdRef = useRef(sessionId);
  sessionIdRef.current = sessionId;
  const [userProfile, setUserProfile] = useState<ChatUserProfile>({});
  const [speechPrimed, setSpeechPrimed] = useState(false);
  const speechPrimingRef = useRef(false);
  const onboardingCompleteRef = useRef(false);
  const userProfileRef = useRef<ChatUserProfile>({});
  const introShownRef = useRef(false);
  const feedRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const widgetRef = useRef<HTMLDivElement>(null);
  const { corner, floatPos, isDragging, onDragStart, onDragMove, onDragEnd, consumeDragClick } =
    useChatDock(widgetRef);
  const { size, isResizing, activeCorner, onResizeStart, onResizeMove, onResizeEnd } =
    useChatResize(widgetRef);
  const queryClient = useQueryClient();

  const scrollFeedToBottom = useCallback(() => {
    const feed = feedRef.current;
    if (!feed) return;
    feed.scrollTop = feed.scrollHeight;
  }, []);

  const send = useMutation({
    mutationFn: (payload: {
      message: string;
      sessionId: string;
      userDisplayName?: string;
      inputSource?: 'text' | 'voice';
      sttRaw?: string;
    }) =>
      api<{ reply: string }>('/chat', {
        method: 'POST',
        body: JSON.stringify({
          message: payload.message,
          session_id: payload.sessionId,
          user_display_name: payload.userDisplayName || undefined,
          input_source: payload.inputSource || 'text',
          stt_raw: payload.sttRaw || undefined,
        }),
      }),
    onSuccess: (data) => {
      setMessages((prev) => [...prev, appendMessage('assistant', data.reply)]);
      void queryClient.invalidateQueries({ queryKey: ['dashboard-overview'] });
      void queryClient.invalidateQueries({ queryKey: ['maintenance'] });
    },
    onError: (err: Error) => {
      const text = err instanceof ApiClientError ? err.message : 'Chat thất bại — thử lại.';
      setMessages((prev) => [...prev, appendMessage('assistant', text)]);
    },
  });

  const sendRef = useRef(send);
  sendRef.current = send;
  const messagesRef = useRef<ChatMessage[]>([]);
  messagesRef.current = messages;
  const lastSubmitRef = useRef<{ text: string; at: number } | null>(null);

  const processUserMessage = useCallback((msg: string, opts?: { inputSource?: 'text' | 'voice'; sttRaw?: string }) => {
    const trimmed = msg.trim();
    if (!trimmed || sendRef.current.isPending) return;

    const now = Date.now();
    const last = lastSubmitRef.current;
    if (last && last.text === trimmed && now - last.at < 1500) return;
    lastSubmitRef.current = { text: trimmed, at: now };

    setInput('');
    voiceRef.current?.setTranscript('');

    const prevProfile = userProfileRef.current;
    const userTexts = [
      ...messagesRef.current.filter((m) => m.role === 'user').map((m) => m.text),
      trimmed,
    ];

    let nextProfile: ChatUserProfile;
    if (onboardingCompleteRef.current) {
      if (shouldHandleAsProfileUpdate(trimmed, true)) {
        nextProfile = applyProfileUpdate(prevProfile, trimmed);
      } else {
        nextProfile = prevProfile;
      }
    } else {
      nextProfile = mergeProfileFromUserTexts(userTexts, {});
      const latest = inferProfileFromSingleMessage(trimmed);
      if (latest.displayName) nextProfile.displayName = latest.displayName;
      if (latest.honorific) nextProfile.honorific = latest.honorific;
      if (latest.gender) nextProfile.gender = latest.gender;
      if (latest.ageGroup) nextProfile.ageGroup = latest.ageGroup;
    }

    if (
      profileChanged(prevProfile, nextProfile) &&
      nextProfile.displayName &&
      nextProfile.displayName !== prevProfile.displayName
    ) {
      const newSessionId = crypto.randomUUID();
      sessionIdRef.current = newSessionId;
      setSessionId(newSessionId);
    }

    userProfileRef.current = nextProfile;
    setUserProfile(nextProfile);

    const chatPayload = {
      message: trimmed,
      sessionId: sessionIdRef.current,
      userDisplayName: userBubbleLabel(nextProfile),
      inputSource: opts?.inputSource ?? 'text',
      sttRaw: opts?.sttRaw,
    };

    let localAssistantReply: ChatMessage | null = null;
    let callBackend = false;

    if (onboardingCompleteRef.current) {
      if (shouldHandleAsProfileUpdate(trimmed, true)) {
        localAssistantReply = appendMessage(
          'assistant',
          buildProfileUpdateReply(nextProfile, prevProfile, trimmed),
        );
      } else {
        callBackend = true;
      }
    } else {
      localAssistantReply = appendMessage('assistant', buildOnboardingReply(nextProfile, trimmed));
      if (onboardingFinished(nextProfile, trimmed)) {
        onboardingCompleteRef.current = true;
        if (isLikelyOpsQuery(trimmed) && !isSkipOnboarding(trimmed)) {
          callBackend = true;
        }
      }
    }

    setMessages((prev) => {
      const withUser = [...prev, appendMessage('user', trimmed)];
      return localAssistantReply ? [...withUser, localAssistantReply] : withUser;
    });

    if (callBackend) {
      sendRef.current.mutate(chatPayload);
    }
  }, []);

  const processUserMessageRef = useRef(processUserMessage);
  processUserMessageRef.current = processUserMessage;

  useEffect(() => {
    if (!open || introShownRef.current) return;
    introShownRef.current = true;
    setMessages([appendMessage('assistant', CHAT_INTRO_MESSAGE)]);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [open]);

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => scrollFeedToBottom());
  }, [messages, send.isPending, open, scrollFeedToBottom]);

  const ensureSpeechPrimed = useCallback(async () => {
    if (speechPrimed || speechPrimingRef.current) return speechPrimed;
    speechPrimingRef.current = true;
    try {
      const ok = await primeSpeechRecognition();
      if (ok) setSpeechPrimed(true);
      return ok;
    } finally {
      speechPrimingRef.current = false;
    }
  }, [speechPrimed]);

  const closeChat = useCallback(() => setOpen(false), []);

  const voiceCallbacksRef = useRef({
    onCloseChat: closeChat,
    onEndSession: () => {},
  });
  voiceCallbacksRef.current.onCloseChat = closeChat;

  const voiceRef = useRef<{ setTranscript: (t: string) => void; stop: () => void; ensureMicOn: () => void } | null>(
    null,
  );

  const handleAloWake = useCallback((remainder: string) => {
    setOpen(true);
    window.setTimeout(() => {
      voiceRef.current?.ensureMicOn();
      if (remainder) {
        voiceRef.current?.setTranscript(remainder);
        setInput(remainder);
      }
      inputRef.current?.focus();
    }, 150);
  }, []);

  const handleMicOn = useCallback(() => {
    handleAloWake('');
  }, [handleAloWake]);

  const voice = useVoiceInput({
    onTranscript: setInput,
    onCloseChat: () => voiceCallbacksRef.current.onCloseChat(),
    onMicOn: handleMicOn,
    onEndSession: () => voiceCallbacksRef.current.onEndSession(),
    onSubmit: (raw) =>
      processUserMessageRef.current(raw.trim(), { inputSource: 'voice', sttRaw: raw.trim() }),
  });

  voiceRef.current = voice;

  voiceCallbacksRef.current.onEndSession = () => {
    voiceRef.current?.stop();
    setOpen(false);
    setInput('');
    voiceRef.current?.setTranscript('');
  };

  const wake = useOpsOneWake({
    enabled: voice.supported && !voice.micOn,
    speechPrimed,
    onWake: handleAloWake,
  });

  useEffect(() => {
    if (!voice.supported || speechPrimed) return;

    const primeOnGesture = () => {
      void ensureSpeechPrimed();
    };

    const opts: AddEventListenerOptions = { capture: true };
    document.addEventListener('pointerdown', primeOnGesture, opts);
    document.addEventListener('touchstart', primeOnGesture, opts);
    document.addEventListener('keydown', primeOnGesture, opts);

    return () => {
      document.removeEventListener('pointerdown', primeOnGesture, opts);
      document.removeEventListener('touchstart', primeOnGesture, opts);
      document.removeEventListener('keydown', primeOnGesture, opts);
    };
  }, [voice.supported, speechPrimed, ensureSpeechPrimed]);

  /** Trình duyệt đã cấp mic trước đó → thử prime ngay khi load (F5 vẫn nghe "alo"). */
  useEffect(() => {
    if (!voice.supported || speechPrimed) return;
    void ensureSpeechPrimed();
  }, [voice.supported, speechPrimed, ensureSpeechPrimed]);

  const submitMessage = useCallback(() => {
    processUserMessageRef.current(input.trim());
  }, [input]);

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
    void ensureSpeechPrimed();
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
              <strong>Chat {ASSISTANT_NAME}</strong>
              <span className="chat-widget__subtitle">
                Nói &quot;{VOICE_WAKE_WORD}&quot; hoặc &quot;bật mic&quot; để mở chat + bật mic · &quot;đóng chat&quot; · &quot;tắt mic&quot; / &quot;bye bye&quot; để thoát
              </span>
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
            {messages.map((m, i) => (
              <ChatBubbleRow
                key={`${m.at}-${i}`}
                role={m.role}
                text={m.text}
                at={m.at}
                userProfile={m.role === 'user' ? userProfile : undefined}
                userDisplayName={m.role === 'user' ? userBubbleLabel(userProfile) : undefined}
                sessionSeed={sessionId}
              />
            ))}
            {send.isPending && (
              <ChatBubbleRow role="assistant" text="Đang xử lý..." pending sessionSeed={sessionId} />
            )}
          </div>

          <div className="chat-widget__input">
            <div className="chat-widget__composer">
              <textarea
                ref={inputRef}
                value={input}
                onChange={(e) => {
                  voice.setTranscript(e.target.value);
                  setInput(e.target.value);
                }}
                onKeyDown={onInputKeyDown}
                placeholder="Nhập câu hỏi..."
                rows={1}
              />
              <div className="chat-input__actions">
                {voice.supported && (
                  <button
                    type="button"
                    className={`btn btn--mic${voice.micOn ? ' btn--mic-active' : ''}`}
                    aria-label={voice.micOn ? 'Tắt micro' : 'Bật micro — hội thoại liên tục'}
                    title={voice.micOn ? 'Bấm để tắt micro' : 'Bật micro; im lặng 2 giây sẽ gửi từng câu'}
                    onClick={() => {
                      void ensureSpeechPrimed();
                      voice.start();
                    }}
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
          </div>
          {voice.supported && voice.micOn && (
            <p className="voice-hint chat-widget__voice">
              Mic bật · im lặng 2s gửi · &quot;đóng chat&quot; / &quot;tắt mic&quot;
            </p>
          )}
        </div>
      ) : (
        <button
          type="button"
          className={`chat-widget__toggle btn btn--primary chat-widget__drag-handle${wake.listening ? ' chat-widget__toggle--wake' : ''}${!speechPrimed && voice.supported ? ' chat-widget__toggle--prime' : ''}`}
          title={
            !speechPrimed && voice.supported
              ? `Bấm trang hoặc nút này để bật nghe "${VOICE_WAKE_WORD}"`
              : wake.listening
                ? `Đang nghe "${VOICE_WAKE_WORD}" — nói để mở chat`
                : `Kéo để đổi góc · Nói "${VOICE_WAKE_WORD}" hoặc bấm để mở chat`
          }
          onPointerDown={handleTogglePointerDown}
        >
          <span className="chat-widget__toggle-label">Chat</span>
          {voice.supported && (
            <span className="chat-widget__toggle-mic" aria-hidden="true">
              🎤
            </span>
          )}
        </button>
      )}
    </div>
  );
}
