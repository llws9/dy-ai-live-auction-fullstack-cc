import React from 'react';
import styles from './Card.module.css';

interface CardProps {
  /** 卡片内容 */
  children: React.ReactNode;
  /** 卡片变体 */
  variant?: 'default' | 'elevated' | 'outlined';
  /** 内边距大小 */
  padding?: 'none' | 'sm' | 'md' | 'lg';
  /** 点击事件 */
  onClick?: () => void;
  /** 自定义类名 */
  className?: string;
  /** 测试 ID */
  testId?: string;
}

/**
 * 可复用的卡片组件
 * 支持多种变体和内边距配置
 */
export const Card: React.FC<CardProps> = ({
  variant = 'default',
  padding = 'md',
  children,
  onClick,
  className,
  testId,
}) => {
  const cardClasses = [
    styles.card,
    styles[variant],
    styles[`padding-${padding}`],
    onClick ? styles.clickable : '',
    className || '',
  ].filter(Boolean).join(' ');

  return (
    <div
      className={cardClasses}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => e.key === 'Enter' && onClick() : undefined}
      data-testid={testId}
    >
      {children}
    </div>
  );
};

export default Card;
