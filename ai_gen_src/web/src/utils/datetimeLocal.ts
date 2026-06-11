function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

export function toDatetimeLocalValue(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

/** ISO/RFC3339 từ API → datetime-local value. */
export function isoToDatetimeLocalValue(iso: string | null | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return '';
  return toDatetimeLocalValue(d);
}

function hour12Parts(hour24: number): { hour: number; period: 'AM' | 'PM' } {
  const period: 'AM' | 'PM' = hour24 >= 12 ? 'PM' : 'AM';
  const hour = hour24 % 12 === 0 ? 12 : hour24 % 12;
  return { hour, period };
}

function toHour24(hour12: number, period: string): number | null {
  const p = period.trim().toUpperCase();
  if (p !== 'AM' && p !== 'PM') return null;
  if (hour12 < 1 || hour12 > 12) return null;
  if (p === 'AM') return hour12 === 12 ? 0 : hour12;
  return hour12 === 12 ? 12 : hour12 + 12;
}

function expandYear2(twoDigit: number): number {
  return twoDigit >= 70 ? 1900 + twoDigit : 2000 + twoDigit;
}

/** Hiển thị kiểu Việt Nam: dd/mm/yyyy hh:mm AM/PM. */
export function formatDatetimeVi(localValue: string): string {
  if (!localValue) return '';
  const d = new Date(localValue);
  if (!Number.isFinite(d.getTime())) return '';
  const { hour, period } = hour12Parts(d.getHours());
  return `${pad2(d.getDate())}/${pad2(d.getMonth() + 1)}/${d.getFullYear()} ${pad2(hour)}:${pad2(d.getMinutes())} ${period}`;
}

/** Gọn hơn cho ô hẹp: dd/mm/yy hh:mm AM/PM. */
export function formatDatetimeViCompact(localValue: string): string {
  if (!localValue) return '';
  const d = new Date(localValue);
  if (!Number.isFinite(d.getTime())) return '';
  const { hour, period } = hour12Parts(d.getHours());
  const yy = String(d.getFullYear()).slice(-2);
  return `${pad2(d.getDate())}/${pad2(d.getMonth() + 1)}/${yy} ${pad2(hour)}:${pad2(d.getMinutes())} ${period}`;
}

function parseDatetimeParts(
  day: number,
  month: number,
  year: number,
  hour: number,
  minute: number,
): string | null {
  if (
    month < 1 ||
    month > 12 ||
    day < 1 ||
    day > 31 ||
    hour < 0 ||
    hour > 23 ||
    minute < 0 ||
    minute > 59
  ) {
    return null;
  }
  const d = new Date(year, month - 1, day, hour, minute, 0, 0);
  if (d.getFullYear() !== year || d.getMonth() !== month - 1 || d.getDate() !== day) {
    return null;
  }
  return toDatetimeLocalValue(d);
}

/** Parse chuỗi dd/mm/yyyy hh:mm AM/PM (hoặc 24h) → datetime-local value. */
export function parseDatetimeVi(text: string): string | null {
  const trimmed = text.trim();
  const m12Short = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{2})\s+(\d{1,2}):(\d{2})\s*(AM|PM)$/i);
  if (m12Short) {
    const hour24 = toHour24(Number(m12Short[4]), m12Short[6]);
    if (hour24 == null) return null;
    return parseDatetimeParts(
      Number(m12Short[1]),
      Number(m12Short[2]),
      expandYear2(Number(m12Short[3])),
      hour24,
      Number(m12Short[5]),
    );
  }
  const m12 = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})\s+(\d{1,2}):(\d{2})\s*(AM|PM)$/i);
  if (m12) {
    const hour24 = toHour24(Number(m12[4]), m12[6]);
    if (hour24 == null) return null;
    return parseDatetimeParts(
      Number(m12[1]),
      Number(m12[2]),
      Number(m12[3]),
      hour24,
      Number(m12[5]),
    );
  }
  const m24 = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})\s+(\d{1,2}):(\d{2})$/);
  if (!m24) return null;
  return parseDatetimeParts(
    Number(m24[1]),
    Number(m24[2]),
    Number(m24[3]),
    Number(m24[4]),
    Number(m24[5]),
  );
}

/** Map API value (TIME legacy or DATETIME) to datetime-local input value. */
export function fromApiDateTime(v: string | undefined, fallbackHour: number, fallbackMin = 0): string {
  if (!v) {
    const d = new Date();
    d.setHours(fallbackHour, fallbackMin, 0, 0);
    return toDatetimeLocalValue(d);
  }
  if (/^\d{1,2}:\d{2}/.test(v) && !v.includes('T') && !v.includes('-')) {
    const [h, m] = v.split(':').map(Number);
    const d = new Date();
    d.setHours(h, m, 0, 0);
    return toDatetimeLocalValue(d);
  }
  const d = new Date(v);
  if (!Number.isFinite(d.getTime())) {
    const fb = new Date();
    fb.setHours(fallbackHour, fallbackMin, 0, 0);
    return toDatetimeLocalValue(fb);
  }
  return toDatetimeLocalValue(d);
}

export function defaultScopeAutoWindow(): { windowStart: string; windowEnd: string } {
  const start = new Date();
  start.setHours(8, 0, 0, 0);
  const end = new Date();
  end.setHours(18, 0, 0, 0);
  if (end.getTime() <= start.getTime()) {
    end.setDate(end.getDate() + 1);
  }
  return { windowStart: toDatetimeLocalValue(start), windowEnd: toDatetimeLocalValue(end) };
}

export function datetimeRangeError(start: string, end: string): string | null {
  const s = new Date(start).getTime();
  const e = new Date(end).getTime();
  if (!Number.isFinite(s) || !Number.isFinite(e)) {
    return 'Thời gian không hợp lệ';
  }
  if (e <= s) {
    return 'Kết thúc phải sau bắt đầu';
  }
  return null;
}

/** Thời gian tin chat — hôm nay chỉ giờ, ngày khác kèm dd/mm/yyyy. */
export function formatChatTime(ms: number): string {
  const d = new Date(ms);
  if (!Number.isFinite(d.getTime())) return '';
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  const { hour, period } = hour12Parts(d.getHours());
  const time = `${pad2(hour)}:${pad2(d.getMinutes())} ${period}`;
  if (sameDay) return time;
  return `${pad2(d.getDate())}/${pad2(d.getMonth() + 1)}/${d.getFullYear()} ${time}`;
}
