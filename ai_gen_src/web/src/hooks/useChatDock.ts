import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent, type RefObject } from 'react';

export type ChatCorner = 'bottom-left' | 'bottom-right' | 'top-left' | 'top-right';

const STORAGE_KEY = 'opsone-chat-corner';
const DRAG_THRESHOLD = 6;
const EDGE_MARGIN = 16;
const TOP_CLEARANCE = 72;

const CORNERS: ChatCorner[] = ['bottom-left', 'bottom-right', 'top-left', 'top-right'];

function loadCorner(): ChatCorner {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && CORNERS.includes(saved as ChatCorner)) {
      return saved as ChatCorner;
    }
  } catch {
    /* ignore */
  }
  return 'bottom-right';
}

function nearestCorner(x: number, y: number): ChatCorner {
  const h = x < window.innerWidth / 2 ? 'left' : 'right';
  const v = y < window.innerHeight / 2 ? 'top' : 'bottom';
  return `${v}-${h}` as ChatCorner;
}

interface DragState {
  pointerId: number;
  startX: number;
  startY: number;
  originLeft: number;
  originTop: number;
  lastLeft: number;
  lastTop: number;
  moved: boolean;
}

export function useChatDock(widgetRef: RefObject<HTMLElement | null>) {
  const [corner, setCorner] = useState<ChatCorner>(loadCorner);
  const [floatPos, setFloatPos] = useState<{ left: number; top: number } | null>(null);
  const dragRef = useRef<DragState | null>(null);
  const lastDragMovedRef = useRef(false);

  const clampPosition = useCallback((left: number, top: number) => {
    const el = widgetRef.current;
    const w = el?.offsetWidth ?? 200;
    const h = el?.offsetHeight ?? 48;
    return {
      left: Math.min(Math.max(EDGE_MARGIN, left), window.innerWidth - w - EDGE_MARGIN),
      top: Math.min(Math.max(TOP_CLEARANCE, top), window.innerHeight - h - EDGE_MARGIN),
    };
  }, [widgetRef]);

  const onDragStart = useCallback(
    (e: ReactPointerEvent) => {
      if (e.button !== 0) return;
      const el = widgetRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      dragRef.current = {
        pointerId: e.pointerId,
        startX: e.clientX,
        startY: e.clientY,
        originLeft: rect.left,
        originTop: rect.top,
        lastLeft: rect.left,
        lastTop: rect.top,
        moved: false,
      };
      lastDragMovedRef.current = false;
      setFloatPos({ left: rect.left, top: rect.top });
      widgetRef.current?.setPointerCapture(e.pointerId);
    },
    [widgetRef],
  );

  const onDragMove = useCallback(
    (e: ReactPointerEvent) => {
      const d = dragRef.current;
      if (!d || d.pointerId !== e.pointerId) return;
      const dx = e.clientX - d.startX;
      const dy = e.clientY - d.startY;
      if (Math.abs(dx) > DRAG_THRESHOLD || Math.abs(dy) > DRAG_THRESHOLD) {
        d.moved = true;
        lastDragMovedRef.current = true;
      }
      const next = clampPosition(d.originLeft + dx, d.originTop + dy);
      d.lastLeft = next.left;
      d.lastTop = next.top;
      setFloatPos(next);
    },
    [clampPosition],
  );

  const onDragEnd = useCallback(
    (e: ReactPointerEvent) => {
      const d = dragRef.current;
      if (!d || d.pointerId !== e.pointerId) return false;
      const el = widgetRef.current;
      const w = el?.offsetWidth ?? 200;
      const h = el?.offsetHeight ?? 48;
      const cx = d.lastLeft + w / 2;
      const cy = d.lastTop + h / 2;
      const next = nearestCorner(cx, cy);
      setCorner(next);
      try {
        localStorage.setItem(STORAGE_KEY, next);
      } catch {
        /* ignore */
      }
      dragRef.current = null;
      setFloatPos(null);
      try {
        widgetRef.current?.releasePointerCapture(e.pointerId);
      } catch {
        /* ignore */
      }
      return d.moved;
    },
    [widgetRef],
  );

  const consumeDragClick = useCallback(() => {
    const moved = lastDragMovedRef.current;
    lastDragMovedRef.current = false;
    return moved;
  }, []);

  return {
    corner,
    floatPos,
    isDragging: floatPos !== null,
    onDragStart,
    onDragMove,
    onDragEnd,
    consumeDragClick,
  };
}
