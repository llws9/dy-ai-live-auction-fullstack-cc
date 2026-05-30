import styles from './BadgeDot.module.css';

interface BadgeDotProps {
  count?: number;
  max?: number;
  dot?: boolean;
  ariaLabel?: string;
  className?: string;
}

const BadgeDot: React.FC<BadgeDotProps> = ({
  count = 0,
  max = 99,
  dot = false,
  ariaLabel,
  className = '',
}) => {
  // 如果不是红点模式，且 count <= 0，则不展示
  if (!dot && count <= 0) {
    return null;
  }

  const displayText = count > max ? `${max}+` : String(count);
  const classes = [styles.badge, dot ? styles.dot : styles.count, className].filter(Boolean).join(' ');

  return (
    <span 
      className={classes} 
      aria-label={ariaLabel || (dot ? '有新提醒' : `${displayText} 条待处理提醒`)}
    >
      {!dot && displayText}
    </span>
  );
};

export default BadgeDot;
