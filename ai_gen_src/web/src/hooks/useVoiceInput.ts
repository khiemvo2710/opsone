import { useCallback, useEffect, useRef, useState } from 'react';

type SpeechRecognitionCtor = new () => SpeechRecognition;

/** Im lặng sau lần nghe cuối → tự gửi tin nhắn chat. */
export const VOICE_SILENCE_MS = 2000;

/** Không có onresult trong khoảng này → khởi động lại STT (tránh mic “đơ”). */
const WATCHDOG_INTERVAL_MS = 5000;
const WATCHDOG_STALE_MS = 12000;

/** Nói các cụm này → bật mic (mở chat nếu đang đóng). */
export const VOICE_MIC_ON_PHRASES = [
  'bật mic',
  'bat mic',
  'mở mic',
  'mo mic',
  'bật micro',
  'bat micro',
  'mở micro',
  'mo micro',
  'bật microphone',
  'mở microphone',
] as const;

/** Nói các cụm này → đóng khung chat (mic vẫn bật). */
export const VOICE_CLOSE_CHAT_PHRASES = [
  'đóng chat',
  'dong chat',
  'thu gọn chat',
  'thu gon chat',
  'ẩn chat',
  'an chat',
  'đóng khung chat',
  'dong khung chat',
] as const;

/** Nói các cụm này → tắt mic và đóng chat. */
export const VOICE_END_SESSION_PHRASES = [
  'tắt mic',
  'tắt micro',
  'tắt microphone',
  'bye bye',
  'baibai',
  'bye',
] as const;

export const VOICE_WAKE_WORD = 'alo';

function phraseMatches(norm: string, phrase: string): boolean {
  const p = normalizeForVoiceMatch(phrase);
  return norm === p || norm.endsWith(` ${p}`) || norm.includes(` ${p} `) || norm.startsWith(`${p} `);
}

function matchesAnyPhrase(norm: string, phrases: readonly string[]): boolean {
  if (!norm) return false;
  return phrases.some((phrase) => phraseMatches(norm, phrase));
}

export function matchesCloseChatPhrase(text: string): boolean {
  return matchesAnyPhrase(normalizeForVoiceMatch(text), VOICE_CLOSE_CHAT_PHRASES);
}

export function matchesMicOnPhrase(text: string): boolean {
  return matchesAnyPhrase(normalizeForVoiceMatch(text), VOICE_MIC_ON_PHRASES);
}

export function matchesEndSessionPhrase(text: string): boolean {
  return matchesAnyPhrase(normalizeForVoiceMatch(text), VOICE_END_SESSION_PHRASES);
}

function resolveVoiceCommand(text: string): 'mic_on' | 'close_chat' | 'end_session' | null {
  const norm = normalizeForVoiceMatch(text);
  if (!norm) return null;
  if (matchesEndSessionPhrase(norm)) return 'end_session';
  if (matchesMicOnPhrase(norm)) return 'mic_on';
  if (matchesCloseChatPhrase(norm)) return 'close_chat';
  return null;
}

function normalizeForVoiceMatch(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd')
    .replace(/[^\p{L}\p{N}\s]/gu, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function matchesMicOffPhrase(text: string): boolean {
  return matchesEndSessionPhrase(text);
}

/** Các biến thể STT của wake word "alo". */
const ALO_WAKE_ALIASES = 'alo|a\\s*lo|allo';
const ALO_VOCATIVE = 'oi|ne|a|nha|nhe|day|di|em|ban|no';
const ALO_LEADING = 'oi|hey|a|e|em|no|ua|nay';

const ALO_WAKE_NAME_RE = new RegExp(
  `(?:^|\\s)(?:${ALO_WAKE_ALIASES})(?:\\s+(?:${ALO_VOCATIVE}))*`,
);

const ALO_WAKE_HEAD_RE = new RegExp(
  `^(?:(?:${ALO_LEADING})\\s+)*(?:${ALO_WAKE_ALIASES})(?:\\s+(?:${ALO_VOCATIVE}))*`,
);

/** Phát hiện gọi "alo" để mở chat + mic. */
export function matchesAloWake(text: string): boolean {
  const norm = normalizeForVoiceMatch(text);
  if (!norm) return false;
  if (ALO_WAKE_HEAD_RE.test(norm)) return true;
  if (!ALO_WAKE_NAME_RE.test(norm)) return false;
  return norm.split(/\s+/).length <= 6;
}

/** @deprecated Dùng matchesAloWake */
export const matchesZalopayWake = matchesAloWake;

/** Bỏ phần "alo" — phần còn lại đưa vào ô chat. */
export function stripAloWakePrefix(text: string): string {
  const norm = normalizeForVoiceMatch(text);
  if (!norm || !matchesAloWake(norm)) return norm;
  return norm
    .replace(ALO_WAKE_NAME_RE, ' ')
    .replace(new RegExp(`^(?:(?:${ALO_LEADING})\\s+)*`), '')
    .replace(/\s+/g, ' ')
    .trim();
}

/** @deprecated Dùng stripAloWakePrefix */
export const stripZalopayWakePrefix = stripAloWakePrefix;

/** Xin quyền mic/STT sau thao tác user (click) — cần trước khi nghe nền "alo". */
export function primeSpeechRecognition(): Promise<boolean> {
  const Ctor = getSpeechRecognition();
  if (!Ctor) return Promise.resolve(false);

  return new Promise((resolve) => {
    let settled = false;
    const finish = (ok: boolean) => {
      if (settled) return;
      settled = true;
      resolve(ok);
    };

    const recognition = new Ctor();
    recognition.lang = 'vi-VN';
    recognition.interimResults = false;
    recognition.continuous = false;

    const cleanup = () => {
      recognition.onstart = null;
      recognition.onresult = null;
      recognition.onend = null;
      recognition.onerror = null;
      try {
        recognition.abort();
      } catch {
        try {
          recognition.stop();
        } catch {
          /* noop */
        }
      }
    };

    recognition.onstart = () => {
      window.setTimeout(() => {
        cleanup();
        finish(true);
      }, 120);
    };

    recognition.onend = () => finish(true);

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      cleanup();
      finish(event.error !== 'not-allowed' && event.error !== 'service-not-allowed');
    };

    try {
      recognition.start();
    } catch {
      cleanup();
      finish(false);
      return;
    }

    window.setTimeout(() => {
      cleanup();
      finish(true);
    }, 1500);
  });
}

export function getSpeechRecognition(): SpeechRecognitionCtor | null {
  const w = window as Window & {
    SpeechRecognition?: SpeechRecognitionCtor;
    webkitSpeechRecognition?: SpeechRecognitionCtor;
  };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

function detachRecognition(recognition: SpeechRecognition) {
  recognition.onstart = null;
  recognition.onresult = null;
  recognition.onend = null;
  recognition.onerror = null;
}

function disposeRecognition(recognition: SpeechRecognition | null) {
  if (!recognition) return;
  detachRecognition(recognition);
  try {
    recognition.abort();
  } catch {
    try {
      recognition.stop();
    } catch {
      /* already stopped */
    }
  }
}

function collectTranscript(event: SpeechRecognitionEvent): string {
  let text = '';
  for (let i = 0; i < event.results.length; i += 1) {
    text += event.results[i][0].transcript;
  }
  return text.trim();
}

export type VoiceState = 'idle' | 'listening';

export interface VoiceInputOptions {
  /** Cập nhật ô nhập khi đang nghe. */
  onTranscript?: (text: string) => void;
  /** Gửi lệnh sau khoảng im lặng VOICE_SILENCE_MS (mic vẫn bật để hội thoại tiếp). */
  onSubmit: (text: string) => void;
  /** "Đóng chat" — thu gọn khung chat, mic vẫn bật. */
  onCloseChat?: () => void;
  /** "Bật mic" / "mở mic" — mở chat + bật mic. */
  onMicOn?: () => void;
  /** "Tắt mic" / "bye bye" — tắt mic và đóng chat. */
  onEndSession?: () => void;
  /** Thời gian im lặng trước khi tự gửi (ms). */
  silenceMs?: number;
}

export function useVoiceInput({
  onTranscript,
  onSubmit,
  onCloseChat,
  onMicOn,
  onEndSession,
  silenceMs = VOICE_SILENCE_MS,
}: VoiceInputOptions) {
  const [supported, setSupported] = useState(false);
  const [micOn, setMicOn] = useState(false);
  const [transcript, setTranscript] = useState('');
  const transcriptRef = useRef('');
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const restartTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const watchdogTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const micSessionRef = useRef(false);
  const ignoreResultsUntilRef = useRef(0);
  const recognitionEpochRef = useRef(0);
  const lastResultAtRef = useRef(0);
  const restartBackoffRef = useRef(150);
  const onTranscriptRef = useRef(onTranscript);
  const onSubmitRef = useRef(onSubmit);
  const onCloseChatRef = useRef(onCloseChat);
  const onMicOnRef = useRef(onMicOn);
  const onEndSessionRef = useRef(onEndSession);
  const spawnRecognitionRef = useRef<() => void>(() => {});

  useEffect(() => {
    onTranscriptRef.current = onTranscript;
    onSubmitRef.current = onSubmit;
    onCloseChatRef.current = onCloseChat;
    onMicOnRef.current = onMicOn;
    onEndSessionRef.current = onEndSession;
  }, [onTranscript, onSubmit, onCloseChat, onMicOn, onEndSession]);

  useEffect(() => {
    setSupported(getSpeechRecognition() !== null);
  }, []);

  const clearSilenceTimer = useCallback(() => {
    if (silenceTimerRef.current !== null) {
      clearTimeout(silenceTimerRef.current);
      silenceTimerRef.current = null;
    }
  }, []);

  const clearRestartTimer = useCallback(() => {
    if (restartTimerRef.current !== null) {
      clearTimeout(restartTimerRef.current);
      restartTimerRef.current = null;
    }
  }, []);

  const clearWatchdogTimer = useCallback(() => {
    if (watchdogTimerRef.current !== null) {
      clearTimeout(watchdogTimerRef.current);
      watchdogTimerRef.current = null;
    }
  }, []);

  const clearTranscriptEverywhere = useCallback(() => {
    transcriptRef.current = '';
    setTranscript('');
    onTranscriptRef.current?.('');
  }, []);

  const stop = useCallback(() => {
    micSessionRef.current = false;
    recognitionEpochRef.current += 1;
    setMicOn(false);
    ignoreResultsUntilRef.current = 0;
    lastResultAtRef.current = 0;
    restartBackoffRef.current = 150;
    clearSilenceTimer();
    clearRestartTimer();
    clearWatchdogTimer();
    disposeRecognition(recognitionRef.current);
    recognitionRef.current = null;
    clearTranscriptEverywhere();
  }, [clearRestartTimer, clearSilenceTimer, clearTranscriptEverywhere, clearWatchdogTimer]);

  const restartAfterSubmit = useCallback(() => {
    if (!micSessionRef.current) return;
    disposeRecognition(recognitionRef.current);
    recognitionRef.current = null;
    clearRestartTimer();
    restartTimerRef.current = setTimeout(() => {
      restartTimerRef.current = null;
      if (micSessionRef.current) spawnRecognitionRef.current();
    }, 80);
  }, [clearRestartTimer]);

  const applyVoiceCommand = useCallback(
    (text: string): boolean => {
      const cmd = resolveVoiceCommand(text);
      if (!cmd) return false;

      clearSilenceTimer();
      clearTranscriptEverywhere();

      if (cmd === 'end_session') {
        stop();
        onEndSessionRef.current?.();
        return true;
      }

      if (cmd === 'mic_on') {
        onMicOnRef.current?.();
        ignoreResultsUntilRef.current = Date.now() + 800;
        if (!micSessionRef.current) {
          restartAfterSubmit();
        } else {
          clearTranscriptEverywhere();
          restartAfterSubmit();
        }
        return true;
      }

      onCloseChatRef.current?.();
      ignoreResultsUntilRef.current = Date.now() + 800;
      restartAfterSubmit();
      return true;
    },
    [clearSilenceTimer, clearTranscriptEverywhere, restartAfterSubmit, stop],
  );

  const submitFromSilence = useCallback(() => {
    const cmd = transcriptRef.current.trim();
    if (!cmd) return;

    clearSilenceTimer();
    if (applyVoiceCommand(cmd)) return;

    clearTranscriptEverywhere();
    onSubmitRef.current(cmd);

    // Chặn onresult cũ; khởi động session STT mới để ô nhập không dính câu trước.
    ignoreResultsUntilRef.current = Date.now() + 800;
    restartAfterSubmit();
  }, [applyVoiceCommand, clearSilenceTimer, clearTranscriptEverywhere, restartAfterSubmit]);

  const scheduleSilenceSubmit = useCallback(() => {
    clearSilenceTimer();
    silenceTimerRef.current = setTimeout(() => {
      silenceTimerRef.current = null;
      submitFromSilence();
    }, silenceMs);
  }, [clearSilenceTimer, silenceMs, submitFromSilence]);

  const scheduleRestart = useCallback(
    (delayMs: number) => {
      if (!micSessionRef.current) return;
      clearRestartTimer();
      restartTimerRef.current = setTimeout(() => {
        restartTimerRef.current = null;
        if (micSessionRef.current) spawnRecognitionRef.current();
      }, delayMs);
    },
    [clearRestartTimer],
  );

  const scheduleWatchdog = useCallback(() => {
    clearWatchdogTimer();
    watchdogTimerRef.current = setTimeout(() => {
      watchdogTimerRef.current = null;
      if (!micSessionRef.current) return;

      const stale =
        recognitionRef.current === null ||
        (lastResultAtRef.current > 0 && Date.now() - lastResultAtRef.current > WATCHDOG_STALE_MS);

      if (stale) {
        disposeRecognition(recognitionRef.current);
        recognitionRef.current = null;
        scheduleRestart(80);
      }

      if (micSessionRef.current) scheduleWatchdog();
    }, WATCHDOG_INTERVAL_MS);
  }, [clearWatchdogTimer, scheduleRestart]);

  useEffect(
    () => () => {
      clearSilenceTimer();
      clearRestartTimer();
      clearWatchdogTimer();
    },
    [clearRestartTimer, clearSilenceTimer, clearWatchdogTimer],
  );

  const spawnRecognition = useCallback(() => {
    const Ctor = getSpeechRecognition();
    if (!Ctor || !micSessionRef.current) return;

    clearRestartTimer();
    disposeRecognition(recognitionRef.current);
    recognitionRef.current = null;

    const epoch = recognitionEpochRef.current + 1;
    recognitionEpochRef.current = epoch;
    const recognition = new Ctor();
    recognitionRef.current = recognition;
    recognition.lang = 'vi-VN';
    recognition.interimResults = true;
    recognition.continuous = true;

    recognition.onstart = () => {
      if (recognitionEpochRef.current !== epoch || !micSessionRef.current) return;
      restartBackoffRef.current = 150;
      setMicOn(true);
    };

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      if (recognitionEpochRef.current !== epoch || !micSessionRef.current) return;
      if (Date.now() < ignoreResultsUntilRef.current) return;

      const trimmed = collectTranscript(event);
      if (!trimmed) return;

      if (applyVoiceCommand(trimmed)) return;

      lastResultAtRef.current = Date.now();
      transcriptRef.current = trimmed;
      setTranscript(trimmed);
      onTranscriptRef.current?.(trimmed);
      scheduleSilenceSubmit();
    };

    recognition.onend = () => {
      if (recognitionEpochRef.current !== epoch) return;
      if (recognitionRef.current === recognition) recognitionRef.current = null;
      if (!micSessionRef.current) {
        setMicOn(false);
        return;
      }
      scheduleRestart(restartBackoffRef.current);
      restartBackoffRef.current = Math.min(restartBackoffRef.current + 100, 800);
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (recognitionEpochRef.current !== epoch) return;
      if (recognitionRef.current === recognition) recognitionRef.current = null;

      if (!micSessionRef.current) {
        setMicOn(false);
        return;
      }

      if (event.error === 'aborted') return;

      if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
        stop();
        return;
      }

      const delay = event.error === 'no-speech' ? 200 : Math.min(restartBackoffRef.current + 200, 1200);
      restartBackoffRef.current = delay;
      scheduleRestart(delay);
    };

    try {
      recognition.start();
    } catch {
      if (!micSessionRef.current) return;
      scheduleRestart(Math.min(restartBackoffRef.current + 200, 1200));
    }
  }, [applyVoiceCommand, clearRestartTimer, scheduleRestart, scheduleSilenceSubmit, stop]);

  useEffect(() => {
    spawnRecognitionRef.current = spawnRecognition;
  }, [spawnRecognition]);

  const beginMicSession = useCallback(() => {
    const Ctor = getSpeechRecognition();
    if (!Ctor) return;

    micSessionRef.current = true;
    recognitionEpochRef.current += 1;
    ignoreResultsUntilRef.current = 0;
    lastResultAtRef.current = Date.now();
    restartBackoffRef.current = 150;
    transcriptRef.current = '';
    setTranscript('');
    setMicOn(true);
    scheduleWatchdog();
    spawnRecognition();
  }, [scheduleWatchdog, spawnRecognition]);

  const start = useCallback(() => {
    if (micSessionRef.current) {
      stop();
      return;
    }
    beginMicSession();
  }, [beginMicSession, stop]);

  const ensureMicOn = useCallback(() => {
    if (micSessionRef.current) return;
    beginMicSession();
  }, [beginMicSession]);

  const setTranscriptSafe = useCallback((text: string) => {
    transcriptRef.current = text;
    setTranscript(text);
    onTranscriptRef.current?.(text);
  }, []);

  const state: VoiceState = micOn ? 'listening' : 'idle';

  return { supported, micOn, state, transcript, setTranscript: setTranscriptSafe, start, ensureMicOn, stop };
}
