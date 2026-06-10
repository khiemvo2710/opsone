import { useCallback, useEffect, useRef, useState } from 'react';

type SpeechRecognitionCtor = new () => SpeechRecognition;

function getSpeechRecognition(): SpeechRecognitionCtor | null {
  const w = window as Window & {
    SpeechRecognition?: SpeechRecognitionCtor;
    webkitSpeechRecognition?: SpeechRecognitionCtor;
  };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

export type VoiceState = 'idle' | 'listening' | 'processing';

export function useVoiceInput(onResult: (text: string) => void) {
  const [supported, setSupported] = useState(false);
  const [state, setState] = useState<VoiceState>('idle');
  const [transcript, setTranscript] = useState('');
  const transcriptRef = useRef('');

  useEffect(() => {
    setSupported(getSpeechRecognition() !== null);
  }, []);

  const stop = useCallback(() => {
    setState('idle');
  }, []);

  const start = useCallback(() => {
    const Ctor = getSpeechRecognition();
    if (!Ctor) return;

    const recognition = new Ctor();
    recognition.lang = 'vi-VN';
    recognition.interimResults = true;
    recognition.continuous = false;
    transcriptRef.current = '';

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
      transcriptRef.current = trimmed;
      setTranscript(trimmed);
    };
    recognition.onend = () => {
      setState('processing');
      const finalText = transcriptRef.current;
      if (finalText) {
        onResult(finalText);
      }
      setState('idle');
    };
    recognition.onerror = () => setState('idle');

    recognition.start();
  }, [onResult]);

  return { supported, state, transcript, setTranscript, start, stop };
}
