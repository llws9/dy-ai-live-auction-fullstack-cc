import React, { useEffect, useState } from 'react';
import './bid-success-animation.css';

interface Ribbon {
  id: number;
  color: string;
  shape: string;
  tx: string;
  ty: string;
  rot: string;
}

export const IntroAnimation: React.FC = () => {
  const [ribbons, setRibbons] = useState<Ribbon[]>([]);

  useEffect(() => {
    const colors = ['#F59E0B', '#EF4444', '#10B981', '#3B82F6', '#8B5CF6', '#EC4899', '#FCD34D'];
    const shapes = ['rect', 'circle', 'long'];

    setRibbons(Array.from({ length: 80 }).map((_, index) => {
      const angle = Math.random() * Math.PI * 2;
      const velocity = 300 + Math.random() * 500;
      const shape = shapes[Math.floor(Math.random() * shapes.length)];

      return {
        id: index,
        color: colors[Math.floor(Math.random() * colors.length)],
        shape,
        tx: `${Math.cos(angle) * velocity}px`,
        ty: `${Math.sin(angle) * velocity}px`,
        rot: `${(Math.random() - 0.5) * 1080}deg`,
      };
    }));
  }, []);

  return (
    <div className="intro-container">
      <div className="gavel-wrapper">
        <svg width="200" height="200" viewBox="0 0 100 100" fill="none" style={{ filter: 'drop-shadow(0 15px 25px rgba(0,0,0,0.3))' }}>
          <rect x="44" y="30" width="12" height="55" rx="4" fill="#8B5A2B" />
          <rect x="44" y="30" width="6" height="55" rx="2" fill="#6B4423" />
          <rect x="15" y="15" width="70" height="34" rx="8" fill="var(--accent)" />
          <rect x="10" y="20" width="12" height="24" rx="4" fill="#D97706" />
          <rect x="78" y="20" width="12" height="24" rx="4" fill="#D97706" />
          <rect x="42" y="15" width="16" height="34" fill="#FDE68A" />
        </svg>
      </div>
      <div className="shockwave" />
      {ribbons.map((ribbon) => (
        <div
          key={ribbon.id}
          className={`ribbon shape-${ribbon.shape}`}
          style={{
            backgroundColor: ribbon.color,
            '--tx': ribbon.tx,
            '--ty': ribbon.ty,
            '--rot': ribbon.rot,
          } as React.CSSProperties}
        />
      ))}
    </div>
  );
};
