import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type RefObject,
} from 'react';

const STORAGE_KEY = 'opsone-chat-size';
const DEFAULT_WIDTH = 800;
const DEFAULT_HEIGHT = 780;
const MIN_WIDTH = 320;
const MIN_HEIGHT = 360;
const MAX_WIDTH = 1200;
const MAX_HEIGHT = 900;
const VIEWPORT_MARGIN = 32;
const TOP_CLEARANCE = 72;

export type ResizeCorner = 'nw' | 'ne' | 'sw' | 'se';

export const RESIZE_CORNERS: ResizeCorner[] = ['nw', 'ne', 'sw', 'se'];

export interface ChatPanelSize {
  width: number;
  height: number;
}

function loadSize(): ChatPanelSize {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { width: DEFAULT_WIDTH, height: DEFAULT_HEIGHT };
    const parsed = JSON.parse(raw) as Partial<ChatPanelSize>;
    if (typeof parsed.width === 'number' && typeof parsed.height === 'number') {
      return clampSize(parsed.width, parsed.height);
    }
  } catch {
    /* ignore */
  }
  return { width: DEFAULT_WIDTH, height: DEFAULT_HEIGHT };
}

function clampSize(width: number, height: number): ChatPanelSize {
  const maxW = Math.min(MAX_WIDTH, window.innerWidth - VIEWPORT_MARGIN);
  const maxH = Math.min(MAX_HEIGHT, window.innerHeight - TOP_CLEARANCE - VIEWPORT_MARGIN);
  return {
    width: Math.round(Math.min(Math.max(MIN_WIDTH, width), maxW)),
    height: Math.round(Math.min(Math.max(MIN_HEIGHT, height), maxH)),
  };
}

function sizeFromCorner(
  corner: ResizeCorner,
  originWidth: number,
  originHeight: number,
  dx: number,
  dy: number,
): ChatPanelSize {
  switch (corner) {
    case 'se':
      return clampSize(originWidth + dx, originHeight + dy);
    case 'sw':
      return clampSize(originWidth - dx, originHeight + dy);
    case 'ne':
      return clampSize(originWidth + dx, originHeight - dy);
    case 'nw':
      return clampSize(originWidth - dx, originHeight - dy);
  }
}

interface ResizeState {
  pointerId: number;
  corner: ResizeCorner;
  startX: number;
  startY: number;
  originWidth: number;
  originHeight: number;
}

export function useChatResize(containerRef: RefObject<HTMLElement | null>) {
  const [size, setSize] = useState<ChatPanelSize>(loadSize);
  const [isResizing, setIsResizing] = useState(false);
  const [activeCorner, setActiveCorner] = useState<ResizeCorner | null>(null);
  const resizeRef = useRef<ResizeState | null>(null);

  useEffect(() => {
    const onWindowResize = () => setSize((prev) => clampSize(prev.width, prev.height));
    window.addEventListener('resize', onWindowResize);
    return () => window.removeEventListener('resize', onWindowResize);
  }, []);

  const onResizeStart = useCallback(
    (corner: ResizeCorner) => (e: ReactPointerEvent) => {
      if (e.button !== 0) return;
      e.preventDefault();
      e.stopPropagation();
      resizeRef.current = {
        pointerId: e.pointerId,
        corner,
        startX: e.clientX,
        startY: e.clientY,
        originWidth: size.width,
        originHeight: size.height,
      };
      setActiveCorner(corner);
      setIsResizing(true);
      containerRef.current?.setPointerCapture(e.pointerId);
    },
    [containerRef, size.height, size.width],
  );

  const onResizeMove = useCallback((e: ReactPointerEvent) => {
    const r = resizeRef.current;
    if (!r || r.pointerId !== e.pointerId) return;
    const dx = e.clientX - r.startX;
    const dy = e.clientY - r.startY;
    setSize(sizeFromCorner(r.corner, r.originWidth, r.originHeight, dx, dy));
  }, []);

  const onResizeEnd = useCallback(
    (e: ReactPointerEvent) => {
      const r = resizeRef.current;
      if (!r || r.pointerId !== e.pointerId) return false;
      resizeRef.current = null;
      setActiveCorner(null);
      setIsResizing(false);
      setSize((prev) => {
        const next = clampSize(prev.width, prev.height);
        try {
          localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        } catch {
          /* ignore */
        }
        return next;
      });
      try {
        containerRef.current?.releasePointerCapture(e.pointerId);
      } catch {
        /* ignore */
      }
      return true;
    },
    [containerRef],
  );

  return {
    size,
    isResizing,
    activeCorner,
    onResizeStart,
    onResizeMove,
    onResizeEnd,
  };
}
