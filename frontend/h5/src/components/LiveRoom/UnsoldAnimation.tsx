import React, { useEffect } from 'react';
import styles from './UnsoldAnimation.module.css';

interface UnsoldAnimationProps {
  onAnimationEnd: () => void;
}

export const UnsoldAnimation: React.FC<UnsoldAnimationProps> = ({ onAnimationEnd }) => {
  useEffect(() => {
    // The total animation takes ~3.5s (3s delay + 0.5s fadeOut in CSS). 
    // Give it a bit more buffer to let the DOM unmount gracefully.
    const timer = setTimeout(onAnimationEnd, 3800);
    return () => clearTimeout(timer);
  }, [onAnimationEnd]);

  return (
    <div className={styles.container} data-testid="unsold-animation">
      <div className={styles.shatterText} data-text="遗憾流拍">
        遗憾流拍
      </div>
      <div className={styles.subtitle}>
        本场竞拍未达成成交
      </div>
    </div>
  );
};
