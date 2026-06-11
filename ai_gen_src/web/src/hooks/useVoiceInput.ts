import { useCallback, useEffect, useRef, useState } from 'react';

type SpeechRecognitionCtor = new () => SpeechRecognition;

/** Im lặng sau lần nghe cuối → tự gửi tin nhắn chat. */
export const VOICE_SILENCE_MS = 2000;

function getSpeechRecognition(): SpeechRecognitionCtor | null {
  const w = window as Window & {
    SpeechRecognition?: SpeechRecognitionCtor;
    webkitSpeechRecognition?: SpeechRecognitionCtor;
  };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

export type VoiceState = 'idle' | 'listening' | 'processing';

export interface VoiceInputOptions {
  /** Cập nhật ô nhập khi đang nghe (hoặc khi dừng mà chưa gửi). */
  onTranscript?: (text: string) => void;
  /** Gửi lệnh sau khoảng im lặng VOICE_SILENCE_MS. */
  onSubmit: (text: string) => void;
  /** Thời gian im lặng trước khi tự gửi (ms). */
  silenceMs?: number;
}

export function useVoiceInput({ onTranscript, onSubmit, silenceMs = VOICE_SILENCE_MS }: VoiceInputOptions) {
  const [supported, setSupported] = useState(false);
  const [state, setState] = useState<VoiceState>('idle');
  const [transcript, setTranscript] = useState('');
  const transcriptRef = useRef('');
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const submittedRef = useRef(false);
  const onTranscriptRef = useRef(onTranscript);
  const onSubmitRef = useRef(onSubmit);

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

  const submitFromSilence = useCallback(() => {
    if (submittedRef.current) return;
    const cmd = transcriptRef.current.trim();
    if (!cmd) return;

    submittedRef.current = true;
    clearSilenceTimer();
    transcriptRef.current = '';
    setTranscript('');
    setState('processing');
    recognitionRef.current?.stop();
    onSubmitRef.current(cmd);
  }, [clearSilenceTimer]);

  const scheduleSilenceSubmit = useCallback(() => {
    clearSilenceTimer();
    silenceTimerRef.current = setTimeout(() => {
      silenceTimerRef.current = null;
      submitFromSilence();
    }, silenceMs);
  }, [clearSilenceTimer, silenceMs, submitFromSilence]);

  useEffect(() => () => clearSilenceTimer(), [clearSilenceTimer]);

  const stop = useCallback(() => {
    clearSilenceTimer();
    recognitionRef.current?.stop();
    recognitionRef.current = null;
    setState('idle');
  }, [clearSilenceTimer]);

  const start = useCallback(() => {
    const Ctor = getSpeechRecognition();
    if (!Ctor) return;

    clearSilenceTimer();
    recognitionRef.current?.stop();

    const recognition = new Ctor();
    recognitionRef.current = recognition;
    recognition.lang = 'vi-VN';
    recognition.interimResults = true;
    recognition.continuous = true;
    transcriptRef.current = '';
    submittedRef.current = false;

    recognition.onstart = () => {
      setState('listening');
      setTranscript('');
    };
    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let text = '';
      for (let i = event.resultIndex; i < event.results.length; i += 1) {
        text += event.results[i][0].transcript;
      }
      const trimmed = text.trim();
      if (!trimmed) return;

      transcriptRef.current = trimmed;
      setTranscript(trimmed);
      onTranscriptRef.current?.(trimmed);
      scheduleSilenceSubmit();
    };
    recognition.onend = () => {
      clearSilenceTimer();
      recognitionRef.current = null;
      if (!submittedRef.current) {
        const finalText = transcriptRef.current.trim();
        if (finalText) {
          onTranscriptRef.current?.(finalText);
        }
      }
      setState('idle');
    };
    recognition.onerror = () => {
      clearSilenceTimer();
      recognitionRef.current = null;
      setState('idle');
    };

    recognition.start();
  }, [clearSilenceTimer, scheduleSilenceSubmit]);

  return { supported, state, transcript, setTranscript, start, stop };
}
