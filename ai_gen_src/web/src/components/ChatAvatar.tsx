import type { CSSProperties } from 'react';
import {
  avatarSpriteCell,
  resolveAvatarProfile,
  type ChatUserProfile,
} from '../utils/chatUserProfile';

const OPSONE_LOGO = '/favicon-64.png';
const AVATAR_SHEET = '/chat-avatars.png';
const COLS = 7;
const ROWS = 4;

function spriteStyle(col: number, row: number): CSSProperties {
  const x = COLS <= 1 ? 0 : (col / (COLS - 1)) * 100;
  const y = ROWS <= 1 ? 0 : (row / (ROWS - 1)) * 100;
  return {
    backgroundImage: `url(${AVATAR_SHEET})`,
    backgroundSize: `${COLS * 100}% ${ROWS * 100}%`,
    backgroundPosition: `${x}% ${y}%`,
  };
}

interface Props {
  role: 'user' | 'assistant';
  userProfile?: ChatUserProfile;
  sessionSeed?: string;
}

export function ChatAvatar({ role, userProfile = {}, sessionSeed = 'opsone' }: Props) {
  if (role === 'assistant') {
    return (
      <div className="chat-bubble__avatar chat-bubble__avatar--assistant" aria-hidden="true">
        <img src={OPSONE_LOGO} alt="" className="chat-bubble__avatar-logo" />
      </div>
    );
  }

  const resolved = resolveAvatarProfile(userProfile);
  const cell = avatarSpriteCell(userProfile, sessionSeed);
  if (!resolved || !cell) {
    return (
      <div
        className="chat-bubble__avatar chat-bubble__avatar--user chat-bubble__avatar--unknown"
        aria-hidden="true"
        title="Chưa xác định"
      >
        <span className="chat-bubble__avatar-unknown-icon" aria-hidden="true" />
      </div>
    );
  }

  const ageLabel =
    resolved.ageGroup === 'young'
      ? 'Trẻ'
      : resolved.ageGroup === 'middle'
        ? 'Trung niên'
        : 'Lớn tuổi';

  return (
    <div
      className="chat-bubble__avatar chat-bubble__avatar--user chat-bubble__avatar--sprite"
      style={spriteStyle(cell.col, cell.row)}
      aria-hidden="true"
      title={`${resolved.gender === 'male' ? 'Nam' : 'Nữ'} · ${ageLabel}`}
    />
  );
}
