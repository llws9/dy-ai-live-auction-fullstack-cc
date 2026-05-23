import React from 'react';
import styles from './Skeleton.module.css';

interface SkeletonProps {
  /** 骨架屏形状 */
  variant?: 'text' | 'circular' | 'rectangular';
  /** 宽度 */
  width?: string | number;
  /** 高度 */
  height?: string | number;
  /** 自定义类名 */
  className?: string;
  /** 动画效果 */
  animation?: 'pulse' | 'wave' | 'none';
}

/**
 * 骨架屏组件
 * 用于加载状态占位显示
 */
export const Skeleton: React.FC<SkeletonProps> = ({
  variant = 'text',
  width,
  height,
  className,
  animation = 'pulse',
}) => {
  const skeletonClasses = [
    styles.skeleton,
    styles[variant],
    animation !== 'none' ? styles[animation] : '',
    className || '',
  ].filter(Boolean).join(' ');

  const style: React.CSSProperties = {
    width: typeof width === 'number' ? `${width}px` : width,
    height: typeof height === 'number' ? `${height}px` : height,
  };

  return <div className={skeletonClasses} style={style} />;
};

export default Skeleton;
