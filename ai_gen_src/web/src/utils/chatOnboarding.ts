import {
  inferProfileFromSingleMessage,
  inferProfileFromMessages,
  isNameIntroduction,
  profileChanged,
  profileKnown,
  userBubbleLabel,
  type ChatUserProfile,
} from './chatUserProfile';

function normalize(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd')
    .trim();
}

export function isSkipOnboarding(text: string): boolean {
  const norm = normalize(text);
  return /\b(bo qua|boqua|skip|khong can|de sau|lan sau|thoi khong can|khong can dau)\b/.test(norm);
}

export function isLikelyOpsQuery(text: string): boolean {
  const norm = normalize(text);
  if (isProfileUpdateRequest(text)) return false;
  return /\b(routing|metric|bao tri|maintenance|incident|su co|health|trang thai|dashboard|duyet|approve|reject|topup|garena|zing|mobi|vina|viettel|esale|imedia|shoppay|pending|gd)\b/.test(
    norm,
  );
}

/** User muốn đổi tên / avatar / xưng hô sau onboarding. */
export function isProfileUpdateRequest(text: string): boolean {
  const norm = normalize(text);
  return (
    /\b(doi ten|doiten|cap nhat avatar|doi avatar|sua avatar|sua ten|ten moi|doi lai ten|cap nhat ten|thay doi ten|cap nhat profile|doi lai avatar|cap nhat thong tin|sua thong tin ca nhan)\b/.test(
      norm,
    ) ||
    /\b(minh ten (la|moi|doi)|ten minh (la|moi|doi)|goi (minh|toi) la)\b/.test(norm) ||
    (/\b(xung ho|goi em la)\b/.test(norm) && /\b(anh|chi|chu|bac|ong|co|em)\b/.test(norm)) ||
    (/\b(doi|cap nhat|sua)\b/.test(norm) &&
      /\b(ten|avatar|tuoi|tuoi|sinh nam|xung ho)\b/.test(norm))
  );
}

function countProfileFields(p: ChatUserProfile): number {
  return [p.displayName, p.gender, p.ageGroup].filter(Boolean).length;
}

/** Câu trả lời đủ tên + tuổi/giới tính — coi là cập nhật profile. */
export function isImplicitProfileRefresh(text: string): boolean {
  return countProfileFields(inferProfileFromSingleMessage(text)) >= 2;
}

export function shouldHandleAsProfileUpdate(text: string, onboardingComplete: boolean): boolean {
  if (!onboardingComplete || isSkipOnboarding(text)) return false;
  return isProfileUpdateRequest(text) || isImplicitProfileRefresh(text) || isNameIntroduction(text);
}

export function buildProfileUpdateReply(
  next: ChatUserProfile,
  prev: ChatUserProfile,
  userMsg: string,
): string {
  const changed = profileChanged(prev, next);
  const patch = inferProfileFromSingleMessage(userMsg);

  if (!changed && countProfileFields(patch) === 0) {
    const sample = prev.displayName?.trim() || 'Tuấn';
    return `Dạ được ạ! Bạn nói rõ tên — vd "Anh ${sample}", "${sample}" hoặc "cập nhật avatar cho tôi là ${sample}".`;
  }

  const merged: ChatUserProfile = { ...prev, ...next };
  const prefix = userBubbleLabel(merged);

  if (profileKnown(next)) {
    return `Dạ được ${prefix}! Mình đã cập nhật tên và avatar theo thông tin mới. Bạn cần hỗ trợ gì thêm?`;
  }

  const name = next.displayName ?? prev.displayName;
  if (name) {
    return `Dạ ${name}! Mình đã cập nhật tên. Bạn cần hỗ trợ gì thêm?`;
  }

  return 'Dạ được! Bạn nói tên (vd "Khiêm") hoặc "Anh …" / "Chị …" để mình cập nhật nhé.';
}

export function honorific(profile: ChatUserProfile): string {
  if (profile.honorific) return profile.honorific;
  return '';
}

/** Trả lời làm quen — hỏi từng bước thiếu gì, không nhảy sang công việc. */
export function buildOnboardingReply(profile: ChatUserProfile, userMsg: string): string {
  if (isSkipOnboarding(userMsg)) {
    if (profile.displayName) {
      const h = honorific(profile);
      return h
        ? `Dạ được ạ! Mình sẽ gọi bạn là ${h} ${profile.displayName}. Bạn cần tra cứu metric, sự cố hay routing gì không?`
        : `Dạ được ạ! Mình sẽ gọi bạn là ${profile.displayName}. Bạn cần tra cứu metric, sự cố hay routing gì không?`;
    }
    return 'Dạ được ạ! Bạn cần tra cứu metric, sự cố hay routing gì không?';
  }

  if (profileKnown(profile)) {
    const prefix = userBubbleLabel(profile);
    return `Rất vui được gặp ${prefix}! Avatar của bạn đã được cập nhật. Bạn cần hỗ trợ gì về vận hành hôm nay?`;
  }

  if (isLikelyOpsQuery(userMsg)) {
    return 'Mình sẽ hỗ trợ ngay! Trước tiên — bạn nói tên hoặc "Anh Khiêm" / "Chị Lan" nhé.';
  }
  return 'Bạn nói tên (vd "Khiêm") hoặc "Anh Khiêm" / "Chị Lan" — tuổi kèm theo nếu muốn.';
}

export function mergeProfileFromText(text: string, prev: ChatUserProfile): ChatUserProfile {
  return inferProfileFromMessages([text], prev);
}

export function mergeProfileFromUserTexts(texts: string[], prev: ChatUserProfile = {}): ChatUserProfile {
  return inferProfileFromMessages(texts, prev);
}

export function onboardingFinished(profile: ChatUserProfile, userMsg: string): boolean {
  return isSkipOnboarding(userMsg) || profileKnown(profile);
}
