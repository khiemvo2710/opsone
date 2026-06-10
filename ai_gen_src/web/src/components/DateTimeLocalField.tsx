import { useEffect, useRef, useState } from 'react';
import { formatDatetimeVi, formatDatetimeViCompact, parseDatetimeVi } from '../utils/datetimeLocal';

interface Props {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
  compact?: boolean;
}

export function DateTimeLocalField({ label, value, onChange, disabled, className, compact }: Props) {
  const formatDisplay = compact ? formatDatetimeViCompact : formatDatetimeVi;
  const [text, setText] = useState(() => formatDisplay(value));
  const pickerRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setText(compact ? formatDatetimeViCompact(value) : formatDatetimeVi(value));
  }, [value, compact]);

  const commitText = (raw: string) => {
    const parsed = parseDatetimeVi(raw);
    if (parsed) {
      onChange(parsed);
      setText(formatDisplay(parsed));
      return;
    }
    setText(formatDisplay(value));
  };

  const openPicker = () => {
    if (disabled) return;
    pickerRef.current?.showPicker?.();
  };

  const rootClass = [
    className ?? 'datetime-local-field',
    compact ? 'datetime-local-field--compact' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <label className={rootClass}>
      <span className="datetime-local-field__label muted">{label}</span>
      <div className="datetime-local-field__wrap">
        <input
          type="text"
          className="datetime-local-field__input"
          inputMode="numeric"
          placeholder={compact ? 'dd/mm/yy hh:mm AM' : 'dd/mm/yyyy hh:mm AM'}
          value={text}
          disabled={disabled}
          onChange={(e) => setText(e.target.value)}
          onBlur={(e) => commitText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              commitText(text);
            }
          }}
        />
        <button
          type="button"
          className="datetime-local-field__picker-btn"
          disabled={disabled}
          title="Chọn ngày giờ"
          aria-label="Chọn ngày giờ"
          onClick={openPicker}
        >
          <span aria-hidden>📅</span>
        </button>
        <input
          ref={pickerRef}
          type="datetime-local"
          className="datetime-local-field__picker-hidden"
          value={value}
          tabIndex={-1}
          aria-hidden
          style={{ colorScheme: 'dark' }}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
    </label>
  );
}
