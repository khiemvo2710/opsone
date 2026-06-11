export type UserGender = 'male' | 'female';
export type UserAgeGroup = 'young' | 'middle' | 'senior';
export type HonorificForm = 'anh' | 'chi' | 'co' | 'chu' | 'bac' | 'ong' | 'ba' | 'em';

export interface ChatUserProfile {
  displayName?: string;
  gender?: UserGender;
  ageGroup?: UserAgeGroup;
  /** Xưng hô user chọn: Anh, Chị, Cô, Chú, Em, … */
  honorific?: HonorificForm;
}

export function profileKnown(p: ChatUserProfile): boolean {
  if (p.displayName) return true;
  if (p.gender || p.honorific) return true;
  return false;
}

const DEFAULT_AGE_GROUP: UserAgeGroup = 'middle';

/** Chuẩn hóa profile cho avatar — tuổi mặc định trung niên nếu user không nói. */
export function resolveAvatarProfile(
  profile: ChatUserProfile,
): { gender: UserGender; ageGroup: UserAgeGroup } | null {
  if (!profileKnown(profile)) return null;

  let gender = profile.gender;
  if (!gender && profile.displayName) {
    gender = inferGenderFromName(profile.displayName);
  }
  if (!gender) return null;

  return {
    gender,
    ageGroup: profile.ageGroup ?? DEFAULT_AGE_GROUP,
  };
}

function normalizeText(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd');
}

function ageGroupFromAge(age: number): UserAgeGroup {
  if (age < 35) return 'young';
  if (age < 56) return 'middle';
  return 'senior';
}

function inferGender(norm: string): UserGender | undefined {
  if (/\b(toi la nam|dan ong|ong ay|anh trai|bo toi|minh la nam|con trai|la nam)\b/.test(norm)) {
    return 'male';
  }
  if (/\b(toi la nu|dan ba|chi gai|me toi|minh la nu|con gai|la nu)\b/.test(norm)) {
    return 'female';
  }
  if (/\b(?:goi toi la|xung ho|goi minh la)\s+anh\b/.test(norm)) {
    return 'male';
  }
  if (/\b(?:goi toi la|xung ho|goi minh la)\s+(chi|co)\b/.test(norm)) {
    return 'female';
  }
  if (/\b(?:^|\s)(chi|co)\s*$/i.test(norm) || norm === 'chi' || norm === 'co') {
    return 'female';
  }
  if (/\b(?:^|\s)(anh|chu|bac|ong)\s*$/i.test(norm) || norm === 'anh') {
    return 'male';
  }
  if (/\b(?:^|\s)(anh|chu|bac|ong)\s+[\p{L}]/u.test(norm)) {
    return 'male';
  }
  if (/\b(?:^|\s)an\s+(?!gi\b|com\b|tooi\b|uong\b|nhe\b|lay\b|ve\b|roi\b|nua\b|tam\b|cho\b)[\p{L}]/u.test(norm)) {
    return 'male';
  }
  if (/\b(?:^|\s)(chi|ba)\s+[\p{L}]/u.test(norm)) {
    return 'female';
  }
  if (/\b(?:^|\s)co\s+(?!dau\b|the\b|do\b|gi\b|vay\b|ma\b|mot\b|nhieu\b|ca\b|rat\b|no\b|le\b|phai\b|theo\b|ay\b|ta\b|no\b)[\p{L}]/u.test(norm)) {
    return 'female';
  }
  return undefined;
}

const HONORIFIC_WORDS = new Set(['anh', 'chi', 'co', 'chu', 'bac', 'ong', 'ba', 'em']);
const NAME_FILLER_WORDS = new Set([
  'ten', 'toi', 'minh', 'em', 'la', 'moi', 'doi', 'cua', 'goi', 'thanh', 'lai', 'la',
  'cap', 'nhat', 'sua', 'avatar', 'profile', 'thong', 'tin', 'update', 'cua', 'ay', 'ta',
]);

const MALE_HONORIFICS = new Set(['anh', 'chu', 'bac', 'ong']);
const FEMALE_HONORIFICS = new Set(['chi', 'ba', 'co']);
const ALL_HONORIFICS = new Set([...MALE_HONORIFICS, ...FEMALE_HONORIFICS, 'em']);

/** Sau xưng hô — không phải tên (đại từ, trợ từ). */
const PRONOUN_AFTER_HONORIFIC = new Set([
  'ay', 'ta', 'no', 'minh', 'toi', 'ban', 'nguoi', 'do', 'kia', 'day', 'nay', 'dau', 'the', 'ma',
  'oi', 'a', 'ha', 'nhe', 'nha', 'vay', 'roi', 'nua', 've', 'di', 'va', 'mot', 'cu', 'nhu',
]);

const NAME_STOP_WORDS = new Set([
  'tuoi', 'sinh', 'nam', 'nu', 'male', 'female', 'cap', 'nhat', 'avatar', 'doi', 'sua',
  'co', 'dau', 'nha', 'khong', 'the', 'gi', 'vui', 'long', 'xin', 'hay', 'di', 'duoc',
  'lai', 'la', 'ten', 'ay', 'ta', 'no', 'minh', 'toi', 'ban', 'nguoi', 'em',
]);

const HONORIFIC_LABEL: Record<HonorificForm, string> = {
  anh: 'Anh',
  chi: 'Chị',
  co: 'Cô',
  chu: 'Chú',
  bac: 'Bác',
  ong: 'Ông',
  ba: 'Bà',
  em: 'Em',
};

function isThirdPersonPhrase(norm: string): boolean {
  return /\b(?:co|chi|anh|chu|bac|ong|ba|no)\s+(?:ay|ta|do|kia|nay)\b/.test(norm);
}

/** Sửa lỗi ASR/voice phổ biến trước khi parse profile. */
function normalizeVoiceInput(raw: string): string {
  return raw
    .replace(/(^|[\s,.!?;:])([ăa]n)(\s+)(?=[\p{L}]{2,})/giu, '$1anh$3')
    .replace(/(^|[\s,.!?;:])(chi)(\s+)(?=[\p{L}]{2,})/giu, '$1chị$3');
}

function honorificToGender(token: string): UserGender | undefined {
  const key = normalizeText(token);
  if (key === 'an') return 'male';
  if (MALE_HONORIFICS.has(key)) return 'male';
  if (FEMALE_HONORIFICS.has(key)) return 'female';
  return undefined;
}

function toHonorificForm(token: string): HonorificForm | undefined {
  const key = normalizeText(token);
  if (key === 'an') return 'anh';
  if (ALL_HONORIFICS.has(key)) return key as HonorificForm;
  return undefined;
}

/** Vế tên ghép phổ biến — không phải xưng hô khi đứng sau tên (Lan Anh, Minh Anh). */
const NAME_COMPOUND_PARTS = new Set(['anh', 'chi']);

function isLikelyNameToken(token: string, hasPriorNamePart = false): boolean {
  const key = normalizeText(token);
  if (key.length < 2) return false;
  if (NAME_STOP_WORDS.has(key)) return false;
  if (NAME_FILLER_WORDS.has(key)) return false;
  if (PRONOUN_AFTER_HONORIFIC.has(key)) return false;
  if (MALE_HONORIFICS.has(key) || FEMALE_HONORIFICS.has(key)) {
    return hasPriorNamePart && NAME_COMPOUND_PARTS.has(key);
  }
  if (/^\d+$/.test(key)) return false;
  return /^[\p{L}']+$/u.test(token);
}

function hasValidNameParts(parts: string[]): boolean {
  return parts.some((p, idx) => isLikelyNameToken(p, idx > 0));
}

const HONORIFIC_NAME_FILLERS = new Set([
  'ten', 'toi', 'minh', 'la', 'moi', 'doi', 'cua', 'goi', 'thanh', 'lai', 'em',
]);

/** "Anh Khiêm", "Anh tên Khiêm", "Chị tên là Lan", "Cô Lan Anh". */
function parseHonorificNamePhrase(
  raw: string,
): Pick<ChatUserProfile, 'displayName' | 'gender' | 'honorific'> {
  const text = normalizeVoiceInput(raw);
  const words = [...text.matchAll(/[\p{L}']+/gu)].map((m) => ({
    raw: m[0],
    norm: normalizeText(m[0]),
  }));

  for (let i = 0; i < words.length; i += 1) {
    const honorific = toHonorificForm(words[i].norm);
    if (!honorific) continue;

    let j = i + 1;
    if (words[j]?.norm && PRONOUN_AFTER_HONORIFIC.has(words[j].norm)) continue;

    while (j < words.length && HONORIFIC_NAME_FILLERS.has(words[j].norm)) {
      j += 1;
    }

    const gender = honorificToGender(words[i].norm);

    const nameParts: string[] = [];
    for (; j < words.length; j += 1) {
      if (/^\d+$/.test(words[j].norm)) break;
      if (NAME_STOP_WORDS.has(words[j].norm)) break;
      if (!isLikelyNameToken(words[j].raw, nameParts.length > 0)) break;
      nameParts.push(words[j].raw);
      if (nameParts.length >= 4) break;
    }

    if (nameParts.length === 0) continue;
    const cleaned = cleanNameCandidate(nameParts.join(' '));
    if (cleaned) {
      return {
        displayName: cleaned,
        gender,
        honorific,
      };
    }
  }

  return {};
}

/** Xưng hô user nói rõ ở đầu câu — fallback khi tên lấy qua pattern khác. */
function extractStatedHonorific(norm: string): HonorificForm | undefined {
  const atStart = norm.match(/^(anh|chi|co|chu|bac|ong|ba|em)\b/);
  if (atStart) return toHonorificForm(atStart[1]);

  const withTen = norm.match(/\b(anh|chi|co|chu|bac|ong|ba|em)\s+ten\b/);
  if (withTen) return toHonorificForm(withTen[1]);

  return undefined;
}

/** Xưng hô hiển thị — chỉ khi user nói rõ (Anh/Cô/Chú/Em…), không suy từ giới tính. */
export function userHonorific(profile: ChatUserProfile): string | undefined {
  if (profile.honorific) return HONORIFIC_LABEL[profile.honorific];
  return undefined;
}

/** Nhãn bubble user: "Anh Khiêm", "Khiêm", hoặc "Bạn". */
export function userBubbleLabel(profile: ChatUserProfile): string {
  const name = profile.displayName?.trim();
  const h = userHonorific(profile);
  if (h && name) return `${h} ${name}`;
  if (h) return h;
  if (name) return name;
  return 'Bạn';
}

function stripLeadingNonNameWords(raw: string): string {
  const words = raw.trim().split(/\s+/).filter(Boolean);
  while (words.length > 0) {
    const key = normalizeText(words[0]);
    if (
      NAME_FILLER_WORDS.has(key) ||
      HONORIFIC_WORDS.has(key) ||
      PRONOUN_AFTER_HONORIFIC.has(key) ||
      NAME_STOP_WORDS.has(key)
    ) {
      words.shift();
      continue;
    }
    break;
  }
  return words.join(' ');
}

function stripNamePrefixes(raw: string): string {
  return stripLeadingNonNameWords(raw);
}

function truncateNameAtStopWords(raw: string): string {
  const stop =
    /\s+(?:co|khong|nha|cap|nhat|avatar|lai|cho|thoi|ma|va|hay|di|gi|duoc|dau|the|giup|xin|vui|long|nhe|nha|oi|a|ha|hen|nhe|nhe|nhe)\b.*$/iu;
  return raw.replace(stop, '').trim();
}

function capitalizeWordPreserving(word: string): string {
  if (!word) return word;
  return word.charAt(0).toLocaleUpperCase('vi-VN') + word.slice(1).toLocaleLowerCase('vi-VN');
}

function hasVietnameseDiacritics(text: string): boolean {
  return /[àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵỷđ]/iu.test(text);
}

/** Tên tiếng Việt phổ biến không dấu → có dấu (voice/ASR thường bỏ dấu). */
const VIETNAME_NAME_FORMS: Record<string, string> = {
  binh: 'Bình',
  chi: 'Chi',
  cuong: 'Cường',
  dat: 'Đạt',
  duc: 'Đức',
  dung: 'Dũng',
  ha: 'Hà',
  hanh: 'Hạnh',
  hieu: 'Hiếu',
  hoang: 'Hoàng',
  hoa: 'Hoa',
  hong: 'Hồng',
  huong: 'Hương',
  hung: 'Hùng',
  khanh: 'Khánh',
  khiem: 'Khiêm',
  khoa: 'Khoa',
  lan: 'Lan',
  anh: 'Anh',
  lien: 'Liên',
  linh: 'Linh',
  long: 'Long',
  mai: 'Mai',
  minh: 'Minh',
  my: 'Mỹ',
  nam: 'Nam',
  ngoc: 'Ngọc',
  nhi: 'Nhi',
  nhung: 'Nhung',
  phong: 'Phong',
  phuong: 'Phương',
  quan: 'Quân',
  quynh: 'Quỳnh',
  son: 'Sơn',
  tai: 'Tài',
  thao: 'Thảo',
  thanh: 'Thanh',
  thu: 'Thu',
  trang: 'Trang',
  trung: 'Trung',
  tuan: 'Tuấn',
  van: 'Văn',
  vinh: 'Vinh',
  yen: 'Yến',
  nguyen: 'Nguyễn',
  tran: 'Trần',
  le: 'Lê',
  pham: 'Phạm',
  huynh: 'Huỳnh',
  vo: 'Võ',
  dang: 'Đặng',
  bui: 'Bùi',
  do: 'Đỗ',
  ngo: 'Ngô',
  duong: 'Dương',
  ly: 'Lý',
  thi: 'Thị',
  dinh: 'Đinh',
  truong: 'Trương',
};

function restoreVietnameseWord(word: string): string {
  const mapped = VIETNAME_NAME_FORMS[normalizeText(word)];
  if (mapped) return mapped;
  return capitalizeWordPreserving(word);
}

function formatDisplayName(raw: string): string {
  const stripped = stripNamePrefixes(raw.trim());
  return stripped
    .split(/\s+/)
    .filter(Boolean)
    .map((w) => (hasVietnameseDiacritics(w) ? capitalizeWordPreserving(w) : restoreVietnameseWord(w)))
    .join(' ');
}

function cleanNameCandidate(raw: string): string | undefined {
  let name = truncateNameAtStopWords(raw)
    .replace(/\s*\d{1,2}\s*tuoi\b.*$/iu, '')
    .replace(/\bsinh\s+nam\s*(19|20)\d{2}\b.*$/iu, '')
    .replace(/\b(?:nam|nu|male|female)\b.*$/iu, '')
    .trim();
  name = stripLeadingNonNameWords(stripNamePrefixes(name));
  if (name.length < 2 || name.length > 32) return undefined;
  if (!/^[\p{L}][\p{L}\s]{0,30}$/u.test(name)) return undefined;
  const parts = name.split(/\s+/).filter(Boolean);
  if (!hasValidNameParts(parts)) return undefined;
  return formatDisplayName(name);
}

/** "tên là Lan Anh", "đổi tên thành Khiêm", "cập nhật avatar cho tôi là Tuấn". */
function extractExplicitName(raw: string, norm: string): string | undefined {
  const patterns = [
    /\bten\s+(?:\w+\s+){0,4}(?:lai\s+)?(?:la|thanh)\s+([\p{L}\s']{2,32})/u,
    /\b(?:doi|cap nhat|sua|update)(?:\s+[\p{L}']+){0,12}?\s+(?:la|thanh)\s+([\p{L}\s']{2,32})/u,
    /\b(?:doi|cap nhat|sua|update)(?:\s+\w+){0,8}?(?:ten|avatar)(?:\s+\w+){0,6}?(?:lai\s+)?(?:la|thanh)\s+([\p{L}\s']{2,32})/u,
    /\b(?:doi|cap nhat|sua)\s+(?:lai\s+)?(?:la|thanh)\s+([\p{L}\s']{2,32})/u,
    /\b(?:goi|xung ho)\s+(?:\w+\s+){0,2}(?:la|thanh)\s+([\p{L}\s']{2,32})/u,
    /\b(?:minh|toi|em|ban)\s+lai\s+la\s+([\p{L}\s']{2,32})/u,
    /\b(?:toi|minh|em)\s+la\s+([\p{L}\s']{2,24})\s*$/u,
  ];
  for (const re of patterns) {
    const m = norm.match(re);
    if (!m?.[1]) continue;
    const nameNorm = m[1].trim();
    if (/\b(admin|ops|nv|nhan vien|van hanh)\b/.test(nameNorm)) continue;
    const fromRaw = extractNameFromRaw(raw, nameNorm);
    const cleaned = cleanNameCandidate(fromRaw ?? nameNorm);
    if (cleaned) return cleaned;
  }
  return undefined;
}

/** User giới thiệu tên sau onboarding — "Tên tôi là Tuấn", "tôi là Lan". */
export function isNameIntroduction(text: string): boolean {
  const norm = normalizeText(text);
  if (!norm) return false;
  if (/\b(ten toi la|ten minh la|ten em la|goi toi la|minh ten la|toi ten la|ten cua toi la)\b/.test(norm)) {
    return true;
  }
  if (/\bten\s+(?:\w+\s+){0,4}(?:la|thanh)\s+[\p{L}]{2,}/u.test(norm)) {
    return Boolean(extractExplicitName(text, norm));
  }
  const m = norm.match(/\b(?:toi|minh|em)\s+la\s+([\p{L}]{2,24})\s*$/u);
  if (m && !/\b(admin|ops|nv|nhan vien|van hanh)\b/.test(m[1])) {
    return Boolean(cleanNameCandidate(m[1]));
  }
  return false;
}

function extractNameFromRaw(raw: string, nameNorm: string): string | undefined {
  const targetWords = nameNorm.split(/\s+/).filter(Boolean);
  if (targetWords.length === 0) return undefined;

  const rawParts = [...raw.matchAll(/[\p{L}']+/gu)].map((m) => ({
    raw: m[0],
    norm: normalizeText(m[0]),
  }));
  if (rawParts.length === 0) return undefined;

  for (let i = 0; i <= rawParts.length - targetWords.length; i += 1) {
    let ok = true;
    const slice: string[] = [];
    for (let j = 0; j < targetWords.length; j += 1) {
      if (rawParts[i + j]?.norm !== targetWords[j]) {
        ok = false;
        break;
      }
      slice.push(rawParts[i + j].raw);
    }
    if (ok) return slice.join(' ');
  }
  return undefined;
}

function inferDisplayName(raw: string): string | undefined {
  const voiceRaw = normalizeVoiceInput(raw);
  const norm = normalizeText(voiceRaw);

  const explicit = extractExplicitName(voiceRaw, norm);
  if (explicit) return explicit;

  if (!isThirdPersonPhrase(norm)) {
    const parsed = parseHonorificNamePhrase(voiceRaw);
    if (parsed.displayName) return parsed.displayName;
  }

  const patterns = [
    /\b(?:doi ten|ten moi|sua ten|cap nhat ten|doi lai ten)(?: la| thanh|:)?\s+([^\d,.\n]{2,40})/,
    /\b(?:minh ten|ten cua minh la|ten minh la|ten toi la|goi toi la|minh ten la|toi ten la|ten em la)\s+([^\d,.\n]{2,40})/,
    /\b(?:^|\s)ten\s+(?!(?:toi|minh|em|cua|la|moi|doi)\b)([^\d,.\n]{2,40})/,
    /\b(?:toi la|minh la)\s+(?:anh|chi|co|chu|bac|ong|an|em)\s+([^\d,.\n]{2,40})/,
  ];
  if (!isThirdPersonPhrase(norm)) {
    patterns.push(
      /\b(?:^|\s)(?:anh|an|chu|bac|ong|em)\s+(?!gi\b|com\b|tooi\b)([^\d,.\n]{2,30})/,
      /\b(?:^|\s)(?:chi|ba)\s+([^\d,.\n]{2,30})/,
      /\b(?:^|\s)co\s+(?!dau\b|the\b|do\b|gi\b|vay\b|ma\b|mot\b|nhieu\b|ca\b|rat\b|no\b|le\b|phai\b|theo\b|ay\b|ta\b)([^\d,.\n]{2,30})/,
    );
  }
  for (const re of patterns) {
    const m = norm.match(re);
    if (!m?.[1]) continue;
    const nameNorm = m[1].trim();
    const fromRaw = extractNameFromRaw(voiceRaw, nameNorm);
    const cleaned = cleanNameCandidate(fromRaw ?? nameNorm);
    if (cleaned) return cleaned;
  }
  return undefined;
}

function inferGenderFromName(displayName: string): UserGender | undefined {
  const first = normalizeText(displayName.split(/\s+/)[0] ?? '');
  if (!first) return undefined;
  const maleNames = new Set([
    'khiem', 'minh', 'hung', 'tuan', 'nam', 'duc', 'long', 'hieu', 'khoa', 'dung', 'cuong',
    'thanh', 'phong', 'quan', 'hoang', 'son', 'tai', 'vinh', 'binh', 'trung', 'dat', 'khanh',
  ]);
  const femaleNames = new Set([
    'lan', 'hoa', 'linh', 'mai', 'thu', 'ngoc', 'trang', 'huong', 'yen', 'ha', 'phuong', 'thao',
    'van', 'nhung', 'hanh', 'dung', 'my', 'lien', 'hong', 'nhi', 'quynh', 'anh', 'chi',
  ]);
  if (femaleNames.has(first)) return 'female';
  if (maleNames.has(first)) return 'male';
  return undefined;
}

function inferAgeGroup(norm: string): UserAgeGroup | undefined {
  const ageMatch = norm.match(/\b(\d{1,2})\s*tuoi\b/);
  if (ageMatch) {
    const age = Number.parseInt(ageMatch[1], 10);
    if (age >= 10 && age <= 99) return ageGroupFromAge(age);
  }

  const yearMatch = norm.match(/\bsinh nam\s*(19\d{2}|20\d{2})\b/);
  if (yearMatch) {
    const birthYear = Number.parseInt(yearMatch[1], 10);
    const age = new Date().getFullYear() - birthYear;
    if (age >= 10 && age <= 100) return ageGroupFromAge(age);
  }

  if (/\b(hoc sinh|sinh vien|tuoi tre|20 tuoi)\b/.test(norm)) return 'young';
  if (/\b(trung nien|40 tuoi|50 tuoi|nua doi)\b/.test(norm)) return 'middle';
  if (/\b(nghi huu|60 tuoi|70 tuoi|cao tuoi)\b/.test(norm)) return 'senior';

  return undefined;
}

/** Suy ra profile từ một câu (cho phép ghi đè khi user cập nhật). */
export function inferProfileFromSingleMessage(text: string): ChatUserProfile {
  const voiceText = normalizeVoiceInput(text);
  const norm = normalizeText(voiceText);
  const next: ChatUserProfile = {};
  if (!norm) return next;

  const honorificName = parseHonorificNamePhrase(voiceText);
  const explicitName = extractExplicitName(voiceText, norm);

  if (explicitName) {
    next.displayName = explicitName;
  } else if (!isThirdPersonPhrase(norm) && honorificName.displayName) {
    next.displayName = honorificName.displayName;
  }

  if (honorificName.honorific && (!isThirdPersonPhrase(norm) || explicitName)) {
    next.honorific = honorificName.honorific;
  }
  if (honorificName.gender && (!isThirdPersonPhrase(norm) || explicitName)) {
    next.gender = honorificName.gender;
  }

  if (!next.honorific && !isThirdPersonPhrase(norm)) {
    const stated = extractStatedHonorific(norm);
    if (stated) next.honorific = stated;
  }

  if (!next.displayName) {
    const n = inferDisplayName(voiceText);
    if (n) next.displayName = n;
  }

  if (!next.gender) {
    const g = inferGender(norm);
    if (g) next.gender = g;
  }
  if (!next.gender && next.honorific) {
    const gFromHonorific = honorificToGender(next.honorific);
    if (gFromHonorific) next.gender = gFromHonorific;
  }

  const a = inferAgeGroup(norm);
  if (a) next.ageGroup = a;

  return next;
}

/** Ghi đè field profile từ câu mới nhất khi user yêu cầu cập nhật. */
export function applyProfileUpdate(prev: ChatUserProfile, latestText: string): ChatUserProfile {
  const patch = inferProfileFromSingleMessage(latestText);
  const next: ChatUserProfile = { ...prev };
  if (patch.displayName) next.displayName = patch.displayName;
  if (patch.gender) next.gender = patch.gender;
  if (patch.ageGroup) next.ageGroup = patch.ageGroup;
  if (patch.honorific) next.honorific = patch.honorific;
  return next;
}

export function profileChanged(prev: ChatUserProfile, next: ChatUserProfile): boolean {
  return (
    prev.displayName !== next.displayName ||
    prev.gender !== next.gender ||
    prev.ageGroup !== next.ageGroup ||
    prev.honorific !== next.honorific
  );
}

/** Suy ra giới tính / độ tuổi từ lời user (chỉ cập nhật field chưa có). */
export function inferProfileFromMessages(
  texts: string[],
  prev: ChatUserProfile = {},
): ChatUserProfile {
  const next: ChatUserProfile = { ...prev };
  for (const raw of texts) {
    const patch = inferProfileFromSingleMessage(raw);
    if (patch.displayName && !next.displayName) next.displayName = patch.displayName;
    if (patch.gender && !next.gender) next.gender = patch.gender;
    if (patch.ageGroup && !next.ageGroup) next.ageGroup = patch.ageGroup;
    if (patch.honorific && !next.honorific) next.honorific = patch.honorific;
    if (profileKnown(next)) break;
  }
  return next;
}

function hashSeed(seed: string): number {
  let h = 0;
  for (let i = 0; i < seed.length; i += 1) {
    h = (h * 31 + seed.charCodeAt(i)) >>> 0;
  }
  return h;
}

/** Chọn ô avatar trong lưới 7×4 (nam hàng 0–1, nữ hàng 2–3). */
export function avatarSpriteCell(
  profile: ChatUserProfile,
  sessionSeed = 'opsone',
): { col: number; row: number } | null {
  const resolved = resolveAvatarProfile(profile);
  if (!resolved) return null;
  const { gender, ageGroup } = resolved;
  const h = hashSeed(`${sessionSeed}:${gender}:${ageGroup}`);

  let row = gender === 'male' ? 0 : 2;
  let col = h % 3;

  if (ageGroup === 'middle') {
    col = 3 + (h % 2);
    if (h % 2 === 1) row += 1;
  } else if (ageGroup === 'senior') {
    col = 5 + (h % 2);
    row = gender === 'male' ? 1 : 3;
  }

  return { col, row };
}
