import React from 'react';
import styles from './Loading.module.css';

interface LoadingProps {
  /** 加载提示文字 */
  text?: string;
  /** 尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 全屏模式 */
  fullscreen?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 加载动画组件
 */
export const Loading: React.FC<LoadingProps> = ({
  text,
  size = 'md',
  fullscreen = false,
  className,
}) => {
  const containerClasses = [
    styles.container,
    fullscreen ? styles.fullscreen : '',
    className || '',
  ].filter(Boolean).join(' ');

  return (
    <div className={containerClasses}>
      <div className={`${styles.spinner} ${styles[size]}`} />
      {text && <span className={styles.text}>{text}</span>}
    </div>
  );
};

export default Loading;
