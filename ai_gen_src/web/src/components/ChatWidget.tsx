import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError, eventsUrl } from '../api/client';
import type { DashboardOverview } from '../types/api';
import { useVoiceInput, primeSpeechRecognition, VOICE_WAKE_WORD } from '../hooks/useVoiceInput';
import { useOpsOneWake } from '../hooks/useOpsOneWake';
import { useChatDock } from '../hooks/useChatDock';
import { RESIZE_CORNERS, useChatResize } from '../hooks/useChatResize';
import { formatChatTime } from '../utils/datetimeLocal';
import { ASSISTANT_NAME } from '../utils/assistantIdentity';
import { userBubbleLabel, inferProfileFromSingleMessage, type ChatUserProfile } from '../utils/chatUserProfile';
import { CHAT_INTRO_MESSAGE } from '../utils/chatIntro';
import { ChatAvatar } from './ChatAvatar';
import { RedSkuScrollNav } from './RedSkuScrollNav';
import { useScrollNav } from '../context/ScrollNavContext';
import { useAuth, chatHistoryStorageKey } from '../context/AuthContext';

interface ChatMessage {
  role: 'user' | 'assistant';
  text: string;
  at: number;
}

function appendMessage(role: ChatMessage['role'], text: string): ChatMessage {
  return { role, text, at: Date.now() };
}

// isAutoMode reports whether the auto_action means the system handles it automatically.
function isAutoMode(autoAction: string | undefined): boolean {
  return autoAction === 'auto' || autoAction === 'time_window';
}

// Format a routing plan item into a detail string.
function formatPlanDetail(plan: Record<string, unknown>): string {
  const productCode = String(plan.product_code || '');
  const skuCode = String(plan.sku_code || '');
  const reasonVI = String(plan.reason_vi || '');
  const proposed = plan.proposed_pct as Record<string, number> | undefined;

  let pctStr = '';
  if (proposed && typeof proposed === 'object') {
    const parts: string[] = [];
    Object.entries(proposed).forEach(([provider, pct]) => {
      parts.push(`${provider}: ${Math.round(pct)}%`);
    });
    if (parts.length > 0) pctStr = ` → ${parts.join(' / ')}`;
  }

  let detail = `${productCode}/${skuCode}${pctStr}`;
  if (reasonVI) detail += ` (${reasonVI})`;
  return detail;
}

// Format pending suggestions from SSE event into a context-aware system message.
// Plans with auto_action=auto/time_window → "system handled" (no action needed).
// Plans with auto_action=recommend_only    → "needs approval".
function formatSuggestionSystemMessage(data: Record<string, unknown>): string {
  if (!data.has_suggestions) return '';

  const allPlans = Array.isArray(data.routing_plans) ? (data.routing_plans as Record<string, unknown>[]) : [];
  const maintenance = Array.isArray(data.maintenance_suggestions) ? (data.maintenance_suggestions as Record<string, unknown>[]) : [];

  const autoPlans = allPlans.filter((p) => isAutoMode(p.auto_action as string | undefined));
  const manualPlans = allPlans.filter((p) => !isAutoMode(p.auto_action as string | undefined));

  const lines: string[] = [];

  // ── Auto-handled plans ────────────────────────────────────────────────────
  if (autoPlans.length > 0) {
    lines.push('🤖 **Hệ thống đã tự động điều phối routing**\n');
    lines.push('🔄 **Chi tiết:**');
    autoPlans.slice(0, 3).forEach((p) => { lines.push(`   • ${formatPlanDetail(p)}`); });
    if (autoPlans.length > 3) lines.push(`   • ...và ${autoPlans.length - 3} thay đổi khác`);
    lines.push('');
    lines.push('💡 Gõ "xem metric" để kiểm tra chỉ số sau routing');
    lines.push('');
  }

  // ── Plans needing manual approval ─────────────────────────────────────────
  if (manualPlans.length > 0) {
    lines.push('📢 **Có đề xuất routing cần duyệt**\n');
    lines.push('🔄 **Đề xuất Routing:**');
    manualPlans.slice(0, 3).forEach((p) => { lines.push(`   • ${formatPlanDetail(p)}`); });
    if (manualPlans.length > 3) lines.push(`   • ...và ${manualPlans.length - 3} kế hoạch khác`);
    lines.push('');
  }

  // ── Maintenance suggestions ───────────────────────────────────────────────
  if (maintenance.length > 0) {
    lines.push('🔧 **Đề xuất Bảo trì cần duyệt:**');
    maintenance.slice(0, 3).forEach((m) => {
      const productCode = String(m.product_code || '');
      const skuCode = String(m.sku_code || '');
      const detail = String(m.detail || '');
      let detailStr = `${productCode}/${skuCode}`;
      if (detail) detailStr += ` — ${detail}`;
      lines.push(`   • ${detailStr}`);
    });
    if (maintenance.length > 3) lines.push(`   • ...và ${maintenance.length - 3} bảo trì khác`);
    lines.push('');
  }

  // ── Action footer — only when human approval is needed ────────────────────
  if (manualPlans.length > 0 || maintenance.length > 0) {
    lines.push('💡 **Hành động:**');
    lines.push('   • Gõ "xem pending" để xem chi tiết');
    lines.push('   • Admin: duyệt/từ chối ngay hoặc vào Dashboard');
  }

  return lines.join('\n');
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
  const scrollNavCtx = useScrollNav();
  const { session } = useAuth();
  const micAllowed = session?.micAllowed ?? true;
  const [open, setOpen] = useState(false);

  // Load persisted history for current user.
  // Suggestion messages (📢 / 🤖 auto-routing) are stripped on load — they are
  // re-injected on mount via the /suggestions check if still relevant.
  // This prevents stale proposals from reappearing after logout / login.
  const [messages, setMessages] = useState<ChatMessage[]>(() => {
    if (!session?.name) return [];
    try {
      const raw = localStorage.getItem(chatHistoryStorageKey(session.name));
      if (!raw) return [];
      const parsed = JSON.parse(raw) as ChatMessage[];
      if (!Array.isArray(parsed)) return [];
      return parsed.filter(
        (m) =>
          !(
            m.role === 'assistant' &&
            (m.text.includes('📢') || m.text.includes('🤖 **Hệ thống đã tự động'))
          )
      );
    } catch {
      return [];
    }
  });
  const [input, setInput] = useState('');
  const [sessionId] = useState(() => crypto.randomUUID());
  const sessionIdRef = useRef(sessionId);
  sessionIdRef.current = sessionId;
  // Profile cố định từ login — không thay đổi trong chat
  const staticProfile = session?.name
    ? (() => {
        const p = inferProfileFromSingleMessage(session.name);
        if (!p.displayName) p.displayName = session.name;
        return p;
      })()
    : {};
  const [userProfile] = useState<ChatUserProfile>(staticProfile);
  const [speechPrimed, setSpeechPrimed] = useState(false);
  const speechPrimingRef = useRef(false);
  const userProfileRef = useRef<ChatUserProfile>(staticProfile);
  // If history was loaded from storage, skip the intro message
  const introShownRef = useRef(messages.length > 0);
  const feedRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const widgetRef = useRef<HTMLDivElement>(null);
  const { corner, floatPos, isDragging, onDragStart, onDragMove, onDragEnd, consumeDragClick } =
    useChatDock(widgetRef);
  const { size, isResizing, activeCorner, onResizeStart, onResizeMove, onResizeEnd } =
    useChatResize(widgetRef);
  const queryClient = useQueryClient();

  // Watch dashboard overview — khi has_pending_suggestions=false, xóa chat message đề xuất ngay.
  // Dashboard là nguồn authoritative — cùng logic với dashboard bar.
  const dashboardHasPendingRef = useRef<boolean | undefined>(undefined);
  const { data: dashboardOverview } = useQuery<DashboardOverview>({
    queryKey: ['dashboard-overview'],
    queryFn: () => api<DashboardOverview>('/dashboard/overview'),
    staleTime: 55_000,
    refetchInterval: 60_000,
  });
  useEffect(() => {
    const hasPending = dashboardOverview?.has_pending_suggestions;
    dashboardHasPendingRef.current = hasPending;
    // Strict false check — undefined means "not loaded yet", true means "pending".
    // Run on every dashboardOverview update (not just when field changes) so that
    // each 60s refetch re-clears if dashboard still says no pending suggestions.
    if (hasPending === false) {
      setMessages((prev) =>
        prev.filter(
          (m) =>
            !(
              m.role === 'assistant' &&
              (m.text.includes('📢') || m.text.includes('🤖 **Hệ thống đã tự động'))
            )
        )
      );
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardOverview]); // depend on whole object — runs every refetch

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
      // Invalidate queries to trigger automatic refetch from server
      void queryClient.invalidateQueries({ queryKey: ['dashboard-overview'] });
      void queryClient.invalidateQueries({ queryKey: ['maintenance'] });
      void queryClient.invalidateQueries({ queryKey: ['incidents'] });
      void queryClient.invalidateQueries({ queryKey: ['routing-plans'] });
      void queryClient.invalidateQueries({ queryKey: ['health-status'] });

      // After maintenance/routing action, refetch queries to get updated data
      // without reloading the page (preserves chat history)
      const shouldRefetch = [
        'Đã bật bảo trì',           // Set maintenance
        'Đã gia hạn',               // Extend maintenance
        'Đã từ chối',               // Reject maintenance/routing
        'Đã duyệt',                 // Approve maintenance/routing
        'Đã mở lại',                // Reopen service
        'Đã trả',                   // Restore baseline (Đã trả routing baseline)
        'Đã cập nhật',              // Set scope auto
        'Đã gia hạn bảo trì',       // Extended maintenance (full phrase)
        'Đã duyệt bảo trì',         // Approved maintenance (full phrase)
        'Đã duyệt kế hoạch routing', // Approved routing plan
      ].some((keyword) => data.reply.includes(keyword));

      if (shouldRefetch) {
        // Refetch queries after 1.2 seconds to show confirmation message first,
        // then silently refresh data in background without page reload
        window.setTimeout(async () => {
          await Promise.all([
            queryClient.refetchQueries({ queryKey: ['dashboard-overview'] }),
            queryClient.refetchQueries({ queryKey: ['maintenance'] }),
            queryClient.refetchQueries({ queryKey: ['incidents'] }),
            queryClient.refetchQueries({ queryKey: ['routing-plans'] }),
            queryClient.refetchQueries({ queryKey: ['health-status'] }),
          ]);
        }, 1200);
      }
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

    const profile = userProfileRef.current;

    const chatPayload = {
      message: trimmed,
      sessionId: sessionIdRef.current,
      userDisplayName: userBubbleLabel(profile),
      inputSource: opts?.inputSource ?? 'text',
      sttRaw: opts?.sttRaw,
    };

    setMessages((prev) => [...prev, appendMessage('user', trimmed)]);
    sendRef.current.mutate(chatPayload);
  }, []);

  const processUserMessageRef = useRef(processUserMessage);
  processUserMessageRef.current = processUserMessage;

  // Persist chat history to localStorage whenever messages change
  useEffect(() => {
    if (!session?.name) return;
    // Don't persist if only the intro message is present (no real conversation yet)
    const hasRealMessages = messages.some((m) => m.role === 'user');
    if (!hasRealMessages && messages.length <= 1) return;
    try {
      localStorage.setItem(chatHistoryStorageKey(session.name), JSON.stringify(messages));
    } catch {
      // storage full or unavailable — ignore
    }
  }, [messages, session?.name]);

  useEffect(() => {
    if (!open || introShownRef.current || messages.length > 0) return;
    introShownRef.current = true;
    setMessages([appendMessage('assistant', CHAT_INTRO_MESSAGE)]);
  }, [open, messages.length]);

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

  const voiceCallbacksRef = useRef({
    onCloseChat: () => {},
    onEndSession: () => {},
  });

  const voiceRef = useRef<{ setTranscript: (t: string) => void; stop: () => void; ensureMicOn: () => void } | null>(
    null,
  );

  const handleAloWake = useCallback((remainder: string) => {
    setOpen(true);
    window.setTimeout(() => {
      // Bật mic nếu chưa bật (khi đang bật rồi thì không restart)
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

  // "đóng chat" và "tắt mic / bye bye" cùng hành vi: dừng mic + đóng chat
  const endVoiceSession = () => {
    voiceRef.current?.stop();
    setOpen(false);
    setInput('');
    voiceRef.current?.setTranscript('');
  };
  voiceCallbacksRef.current.onCloseChat = endVoiceSession;
  voiceCallbacksRef.current.onEndSession = endVoiceSession;

  const wake = useOpsOneWake({
    enabled: (voice.supported && micAllowed) && !voice.micOn,
    speechPrimed,
    onWake: handleAloWake,
  });

  // Suggestions sync: check /suggestions on mount, on every SSE cycle, and every 20s
  // so the chat clears stale proposals quickly when the situation recovers —
  // without waiting for the next 5-min agent cycle.
  useEffect(() => {
    let es: EventSource | null = null;
    let mounted = true;

    const isSuggestionMessage = (m: ChatMessage) =>
      m.role === 'assistant' &&
      (m.text.includes('📢') || m.text.includes('🤖 **Hệ thống đã tự động'));

    const handleSuggestionData = (data: { has_suggestions?: boolean; [k: string]: unknown }) => {
      if (!mounted) return;
      // Dashboard is authoritative: if it already confirmed no pending suggestions,
      // don't let a stale /suggestions poll re-inject the message.
      if (!data.has_suggestions || dashboardHasPendingRef.current === false) {
        // Tình hình đã ổn → xóa message đề xuất cũ khỏi chat ngay lập tức.
        setMessages((prev) => prev.filter((m) => !isSuggestionMessage(m)));
        return;
      }
      setOpen(true);
      const message = formatSuggestionSystemMessage(data);
      if (message) {
        setMessages((prev) => {
          const lastMsg = prev[prev.length - 1];
          if (lastMsg && isSuggestionMessage(lastMsg)) return prev;
          return [...prev, appendMessage('assistant', message)];
        });
        requestAnimationFrame(() => scrollFeedToBottom());
      }
    };

    const pollSuggestions = async () => {
      try {
        const data = await api<{ has_suggestions?: boolean }>('/suggestions');
        handleSuggestionData(data);
      } catch { /* network error — ignore */ }
    };

    // Immediate check on mount.
    pollSuggestions();

    // Always poll every 20 s — catches plan cancellations that happen between agent cycles.
    // SSE supplements this by reacting to new plans immediately when a cycle fires.
    const pollTimer = setInterval(pollSuggestions, 20_000);

    try {
      es = new EventSource(eventsUrl());
      es.addEventListener('pending_suggestions', (event) => {
        try { handleSuggestionData(JSON.parse(event.data) as { has_suggestions?: boolean }); }
        catch { /* ignore parse errors */ }
      });
      es.onerror = () => { es?.close(); es = null; };
    } catch { /* SSE unavailable — polling covers it */ }

    return () => {
      mounted = false;
      es?.close();
      clearInterval(pollTimer);
    };
  }, [scrollFeedToBottom]);

  useEffect(() => {
    if (!(voice.supported && micAllowed) || speechPrimed) return;

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
  }, [(voice.supported && micAllowed), speechPrimed, ensureSpeechPrimed]);

  // Auto-prime removed: calling primeSpeechRecognition() on mount triggers the
  // browser's native mic permission popup immediately on page load, even when
  // the user already granted permission at login. Priming now only happens on
  // first user gesture (pointerdown / keydown above).

  // Auto-refetch dashboard data every 1 minute (60 seconds) to keep UI up-to-date
  // This simulates automatic polling without reloading the page or losing chat history
  useEffect(() => {
    const refetchInterval = setInterval(() => {
      void Promise.all([
        queryClient.refetchQueries({ queryKey: ['dashboard-overview'] }),
        queryClient.refetchQueries({ queryKey: ['maintenance'] }),
        queryClient.refetchQueries({ queryKey: ['incidents'] }),
        queryClient.refetchQueries({ queryKey: ['routing-plans'] }),
        queryClient.refetchQueries({ queryKey: ['health-status'] }),
      ]).catch(() => {
        // Silently ignore refetch errors
      });
    }, 60 * 1000); // 60 seconds = 1 minute

    return () => clearInterval(refetchInterval);
  }, [queryClient]);

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
      {!open && scrollNavCtx.data && (
        // eslint-disable-next-line jsx-a11y/no-static-element-interactions
        <div
          onPointerDown={(e) => e.stopPropagation()}
          onPointerUp={(e) => e.stopPropagation()}
        >
          <RedSkuScrollNav
            rows={scrollNavCtx.data.rows}
            thresholdsByProduct={scrollNavCtx.data.thresholdsByProduct}
          />
        </div>
      )}
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
