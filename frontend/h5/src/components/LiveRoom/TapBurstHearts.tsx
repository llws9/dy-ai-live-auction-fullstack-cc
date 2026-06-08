import React, { useState, useCallback, useEffect } from 'react';
import styles from './TapBurstHearts.module.css';

interface Heart { id: number; x: number; y: number; tx: string; ty: string; tx2: string; ty2: string; rot: string; rot2: string; }

export const TapBurstHearts: React.FC = () => {
  const [hearts, setHearts] = useState<Heart[]>([]);

  const handleDoubleClick = useCallback((e: MouseEvent) => {
    // Ignore if clicking on buttons or dock
    if ((e.target as HTMLElement).closest('button, .dock, .sheet, a, [data-interactive="true"]')) return;

    const newHearts = Array.from({ length: 5 }).map((_, i) => ({
      id: Date.now() + i + Math.random(),
      x: e.clientX,
      y: e.clientY,
      tx: `${(Math.random() - 0.5) * 100}px`,
      ty: `-${Math.random() * 50 + 50}px`,
      tx2: `${(Math.random() - 0.5) * 150}px`,
      ty2: `-${Math.random() * 100 + 100}px`,
      rot: `${(Math.random() - 0.5) * 60}deg`,
      rot2: `${(Math.random() - 0.5) * 120}deg`,
    }));
    setHearts(prev => [...prev.slice(-20), ...newHearts]);
    // TODO: Need WS event to broadcast like
  }, []);

  useEffect(() => {
    window.addEventListener('dblclick', handleDoubleClick);
    return () => window.removeEventListener('dblclick', handleDoubleClick);
  }, [handleDoubleClick]);

  return (
    <div className={styles.container} data-testid="tap-burst-hearts">
      {hearts.map(h => (
        <div key={h.id} className={styles.heart} style={{
          left: h.x, top: h.y,
          '--tx': h.tx, '--ty': h.ty, '--tx2': h.tx2, '--ty2': h.ty2,
          '--rot': h.rot, '--rot2': h.rot2
        } as React.CSSProperties}>❤️</div>
      ))}
    </div>
  );
};
