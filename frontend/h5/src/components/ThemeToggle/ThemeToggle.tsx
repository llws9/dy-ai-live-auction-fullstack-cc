import { useTheme } from '../../store/themeContext';
import styles from './ThemeToggle.module.css';

function ThemeToggle() {
  const { theme, toggle } = useTheme();
  const isDark = theme === 'dark';
  // 采用动态 label 模式表达"按下后会发生什么"，状态信息已由 label 翻转承载，
  // 故不再叠加 aria-pressed，避免读屏器朗读出歧义状态描述。
  const ariaLabel = isDark ? '切换到浅色模式' : '切换到深色模式';
  const icon = isDark ? '☀' : '☾';

  return (
    <button
      type="button"
      className={styles.toggle}
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
