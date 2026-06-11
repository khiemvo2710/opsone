import { useCallback, useEffect, useRef, useState } from 'react';
import {
  getSpeechRecognition,
  matchesAloWake,
  matchesMicOnPhrase,
  stripAloWakePrefix,
} from './useVoiceInput';

const WAKE_COOLDOWN_MS = 2500;
const WAKE_RESTART_MS = 400;

function collectTranscript(event: SpeechRecognitionEvent): string {
  let text = '';
  for (let i = 0; i < event.results.length; i += 1) {
    text += event.results[i][0].transcript;
  }
  return text.trim();
}

function disposeRecognition(recognition: SpeechRecognition | null) {
  if (!recognition) return;
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
}

export interface OpsOneWakeOptions {
  /** Nghe wake word khi mic chat chua bat. */
  enabled: boolean;
  /** Da xin quyen mic/STT sau thao tac user. */
  speechPrimed: boolean;
  onWake: (remainder: string) => void;
  onPermissionBlocked?: () => void;
}

/** Nghe nen wake word "alo" -> mo chat + bat mic. */
export function useOpsOneWake({
  enabled,
  speechPrimed,
  onWake,
  onPermissionBlocked,
}: OpsOneWakeOptions) {
  const [listening, setListening] = useState(false);
  const [permissionBlocked, setPermissionBlocked] = useState(false);
  const activeRef = useRef(false);
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const restartTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastWakeAtRef = useRef(0);
  const onWakeRef = useRef(onWake);
  const onPermissionBlockedRef = useRef(onPermissionBlocked);
  const spawnRef = useRef<() => void>(() => {});

  useEffect(() => {
    onWakeRef.current = onWake;
  }, [onWake]);

  useEffect(() => {
    onPermissionBlockedRef.current = onPermissionBlocked;
  }, [onPermissionBlocked]);

  useEffect(() => {
    if (speechPrimed) {
      setPermissionBlocked(false);
    }
  }, [speechPrimed]);

  const clearRestartTimer = useCallback(() => {
    if (restartTimerRef.current !== null) {
      clearTimeout(restartTimerRef.current);
      restartTimerRef.current = null;
    }
  }, []);

  const stop = useCallback(() => {
    activeRef.current = false;
    setListening(false);
    clearRestartTimer();
    disposeRecognition(recognitionRef.current);
    recognitionRef.current = null;
  }, [clearRestartTimer]);

  const triggerWake = useCallback(
    (transcript: string) => {
      const now = Date.now();
      if (now - lastWakeAtRef.current < WAKE_COOLDOWN_MS) return;
      lastWakeAtRef.current = now;

      const remainder = stripAloWakePrefix(transcript);
      stop();
      onWakeRef.current(remainder);
    },
    [stop],
  );

  const spawn = useCallback(() => {
    if (!activeRef.current) return;

    const Ctor = getSpeechRecognition();
    if (!Ctor) {
      stop();
      return;
    }

    disposeRecognition(recognitionRef.current);
    recognitionRef.current = null;

    const recognition = new Ctor();
    recognitionRef.current = recognition;
    recognition.lang = 'vi-VN';
    recognition.interimResults = true;
    recognition.continuous = true;

    recognition.onstart = () => {
      if (!activeRef.current) return;
      setListening(true);
    };

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      if (!activeRef.current) return;
      const text = collectTranscript(event);
      if (!text || (!matchesAloWake(text) && !matchesMicOnPhrase(text))) return;
      triggerWake(text);
    };

    recognition.onend = () => {
      if (recognitionRef.current === recognition) {
        recognitionRef.current = null;
      }
      setListening(false);
      if (!activeRef.current) return;
      clearRestartTimer();
      restartTimerRef.current = setTimeout(() => {
        restartTimerRef.current = null;
        spawnRef.current();
      }, WAKE_RESTART_MS);
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (recognitionRef.current === recognition) {
        recognitionRef.current = null;
      }
      setListening(false);
      if (!activeRef.current) return;
      if (event.error === 'aborted') return;
      if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
        setPermissionBlocked(true);
        setListening(false);
        onPermissionBlockedRef.current?.();
        clearRestartTimer();
        return;
      }
      clearRestartTimer();
      restartTimerRef.current = setTimeout(() => {
        restartTimerRef.current = null;
        spawnRef.current();
      }, WAKE_RESTART_MS);
    };

    try {
      recognition.start();
    } catch {
      if (!activeRef.current) return;
      clearRestartTimer();
      restartTimerRef.current = setTimeout(() => {
        restartTimerRef.current = null;
        spawnRef.current();
      }, WAKE_RESTART_MS);
    }
  }, [clearRestartTimer, stop, triggerWake]);

  useEffect(() => {
    spawnRef.current = spawn;
  }, [spawn]);

  useEffect(() => {
    if (!enabled) {
      stop();
      return;
    }

    const Ctor = getSpeechRecognition();
    if (!Ctor) {
      stop();
      return;
    }

    activeRef.current = true;
    spawn();

    return () => {
      stop();
    };
  }, [enabled, speechPrimed, spawn, stop]);

  return {
    listening,
    permissionBlocked,
    supported: getSpeechRecognition() !== null,
  };
}
