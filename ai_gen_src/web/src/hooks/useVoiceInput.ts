import { useCallback, useEffect, useRef, useState } from 'react';

type SpeechRecognitionCtor = new () => SpeechRecognition;

/** Im lặng sau lần nghe cuối → tự gửi tin nhắn chat. */
export const VOICE_SILENCE_MS = 2000;

/** Không có onresult trong khoảng này → khởi động lại STT (tránh mic “đơ”). */
const WATCHDOG_INTERVAL_MS = 5000;
const WATCHDOG_STALE_MS = 12000;

/** Nói các cụm này → tắt mic, không gửi chat. */
export const VOICE_MIC_OFF_PHRASES = [
  'tắt mic',
  'tắt micro',
  'tắt microphone',
  'dừng mic',
  'thôi mic',
  'ngừng nghe',
  'ngừng mic',
  'kết thúc cuộc trò chuyện',
  'kết thúc hội thoại',
  'kết thúc chat',
  'dừng nghe',
  'tắt đi',
] as const;

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
  const norm = normalizeForVoiceMatch(text);
  if (!norm) return false;
  return VOICE_MIC_OFF_PHRASES.some((phrase) => {
    const p = normalizeForVoiceMatch(phrase);
    return norm === p || norm.endsWith(` ${p}`) || norm.includes(` ${p} `) || norm.startsWith(`${p} `);
  });
}

function getSpeechRecognition(): SpeechRecognitionCtor | null {
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
  /** Thời gian im lặng trước khi tự gửi (ms). */
  silenceMs?: number;
}

export function useVoiceInput({ onTranscript, onSubmit, silenceMs = VOICE_SILENCE_MS }: VoiceInputOptions) {
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
  const spawnRecognitionRef = useRef<() => void>(() => {});

  useEffect(() => {
    onTranscriptRef.current = onTranscript;
    onSubmitRef.current = onSubmit;
  }, [onTranscript, onSubmit]);

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

  const submitFromSilence = useCallback(() => {
    const cmd = transcriptRef.current.trim();
    if (!cmd) return;

    clearSilenceTimer();
    if (matchesMicOffPhrase(cmd)) {
      clearTranscriptEverywhere();
      stop();
      return;
    }

    clearTranscriptEverywhere();
    onSubmitRef.current(cmd);

    // Chặn onresult cũ; khởi động session STT mới để ô nhập không dính câu trước.
    ignoreResultsUntilRef.current = Date.now() + 800;
    restartAfterSubmit();
  }, [clearSilenceTimer, clearTranscriptEverywhere, restartAfterSubmit, stop]);

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

      if (matchesMicOffPhrase(trimmed)) {
        clearSilenceTimer();
        clearTranscriptEverywhere();
        stop();
        return;
      }

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
  }, [clearRestartTimer, clearSilenceTimer, clearTranscriptEverywhere, scheduleRestart, scheduleSilenceSubmit, stop]);

  useEffect(() => {
    spawnRecognitionRef.current = spawnRecognition;
  }, [spawnRecognition]);

  const start = useCallback(() => {
    if (micSessionRef.current) {
      stop();
      return;
    }

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
  }, [scheduleWatchdog, spawnRecognition, stop]);

  const setTranscriptSafe = useCallback((text: string) => {
    transcriptRef.current = text;
    setTranscript(text);
    onTranscriptRef.current?.(text);
  }, []);

  const state: VoiceState = micOn ? 'listening' : 'idle';

  return { supported, micOn, state, transcript, setTranscript: setTranscriptSafe, start, stop };
}
