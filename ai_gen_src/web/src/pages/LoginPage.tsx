import { useState, useRef } from 'react';
import { useAuth } from '../context/AuthContext';
import { ASSISTANT_NAME } from '../utils/assistantIdentity';
import { ChatAvatar } from '../components/ChatAvatar';
import { inferProfileFromSingleMessage } from '../utils/chatUserProfile';

export function LoginPage() {
  const { login } = useAuth();
  const [name, setName] = useState('');
  const [micAllowed, setMicAllowed] = useState(true);
  const [micBlocked, setMicBlocked] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  async function requestMicPermission() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach((t) => t.stop());
      setMicBlocked(false);
    } catch {
      setMicBlocked(true);
    }
  }

  // Derive profile for avatar preview
  const profile = name.trim()
    ? inferProfileFromSingleMessage(name.trim())
    : {};
  if (name.trim() && !profile.displayName) {
    profile.displayName = name.trim();
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Vui lòng nhập tên của bạn');
      inputRef.current?.focus();
      return;
    }
    if (trimmed.length < 2) {
      setError('Tên phải có ít nhất 2 ký tự');
      inputRef.current?.focus();
      return;
    }
    login(trimmed, micAllowed);
  }

  return (
    <div className="login-page">
      <div className="login-card">
        {/* Logo + title */}
        <div className="login-card__brand">
          <img src="/favicon-64.png" alt={ASSISTANT_NAME} className="login-card__logo" width={48} height={48} />
          <h1 className="login-card__title">{ASSISTANT_NAME}</h1>
          <p className="login-card__subtitle">Hệ thống giám sát thanh toán tự động</p>
        </div>

        <form className="login-card__form" onSubmit={handleSubmit} noValidate>
          {/* Avatar preview */}
          <div className="login-card__avatar-preview">
            <ChatAvatar role="user" userProfile={profile} sessionSeed={name || 'opsone'} />
            <span className="login-card__avatar-hint">
              {profile.displayName ? `Xin chào, ${profile.displayName}!` : 'Nhập tên để xem avatar'}
            </span>
          </div>

          {/* Name input */}
          <div className="login-card__field">
            <label htmlFor="login-name" className="login-card__label">
              Tên của bạn
            </label>
            <input
              ref={inputRef}
              id="login-name"
              type="text"
              className={`login-card__input${error ? ' login-card__input--error' : ''}`}
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                if (error) setError('');
              }}
              placeholder="Ví dụ: Khiêm, Anh Tuấn, Chị Lan..."
              autoComplete="off"
              autoFocus
              maxLength={50}
            />
            {error && <span className="login-card__error">{error}</span>}
          </div>

          {/* Mic permission */}
          <label className="login-card__checkbox-row">
            <input
              type="checkbox"
              className="login-card__checkbox"
              checked={micAllowed}
              onChange={(e) => {
                const checked = e.target.checked;
                setMicAllowed(checked);
                if (checked) void requestMicPermission();
                else setMicBlocked(false);
              }}
            />
            <span className="login-card__checkbox-label">
              Cho phép dùng microphone để trò chuyện bằng giọng nói
            </span>
          </label>
          {micAllowed && micBlocked && (
            <p className="login-card__mic-warning">
              ⚠️ Microphone bị chặn bởi trình duyệt. Bấm vào biểu tượng 🎤 trên thanh địa chỉ để cấp quyền, rồi tải lại trang.
            </p>
          )}

          <button type="submit" className="login-card__btn">
            Vào hệ thống
          </button>
        </form>
      </div>
    </div>
  );
}
