import { useTheme } from '../../store/themeContext';
import styles from './ThemeToggle.module.css';

interface ThemeToggleProps {
  /** 视觉尺寸：默认 'md'（44×44，独立浮层用）；'sm' 为 32×32，适合内嵌进页面 header */
  size?: 'sm' | 'md';
}

function ThemeToggle({ size = 'md' }: ThemeToggleProps) {
  const { theme, toggle } = useTheme();
  const isDark = theme === 'dark';
  // 采用动态 label 模式表达"按下后会发生什么"，状态信息已由 label 翻转承载，
  // 故不再叠加 aria-pressed，避免读屏器朗读出歧义状态描述。
  const ariaLabel = isDark ? '切换到浅色模式' : '切换到深色模式';
  const icon = isDark ? '☀' : '☾';

  const className = size === 'sm' ? `${styles.toggle} ${styles.sm}` : styles.toggle;

  return (
    <button
      type="button"
      className={className}
      aria-label={ariaLabel}
      onClick={toggle}
    >
      <span className={styles.icon} aria-hidden="true">
        {icon}
      </span>
    </button>
  );
}

export default ThemeToggle;
