import React from 'react';
import styles from './GlitchCountdown.module.css';

interface GlitchCountdownProps {
  timeLeft: number;
}

export const GlitchCountdown: React.FC<GlitchCountdownProps> = ({ timeLeft }) => {
  if (timeLeft > 5 || timeLeft <= 0) {
    return null;
  }

  // We use key={timeLeft} to re-trigger the CSS animation every time the number changes
  return (
    <div className={styles.glitchContainer} data-testid="glitch-countdown">
      <div 
        key={timeLeft}
        className={styles.glitchNumber}
        data-text={timeLeft}
      >
        {timeLeft}
      </div>
    </div>
  );
};
