import React from 'react';
import styles from './Transition.module.css';

interface TransitionProps {
  children: React.ReactNode;
  show: boolean;
  animation?: 'fade' | 'slide' | 'scale' | 'slideUp' | 'slideDown';
  duration?: number;
  unmountOnExit?: boolean;
  className?: string;
}

/**
 * 过渡动画组件
 * 支持多种动画类型，用于页面和组件的过渡效果
 */
export const Transition: React.FC<TransitionProps> = ({
  children,
  show,
  animation = 'fade',
  duration = 300,
  unmountOnExit = true,
  className = '',
}) => {
  const [shouldRender, setShouldRender] = React.useState(show);
  const [animationClass, setAnimationClass] = React.useState('');

  React.useEffect(() => {
    if (show) {
      setShouldRender(true);
      requestAnimationFrame(() => {
        setAnimationClass(styles[`${animation}In`] || '');
      });
    } else if (shouldRender) {
      setAnimationClass(styles[`${animation}Out`] || '');
      const timer = setTimeout(() => {
        if (unmountOnExit) {
          setShouldRender(false);
        }
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [show, animation, duration, shouldRender, unmountOnExit]);

  if (!shouldRender) return null;

  return (
    <div
      className={`${animationClass} ${className}`.trim()}
      style={{ animationDuration: `${duration}ms` }}
    >
      {children}
    </div>
  );
};

/**
 * 条件渲染过渡组件
 * 简化版的 Transition，用于条件渲染场景
 */
export const FadeTransition: React.FC<{
  children: React.ReactNode;
  show: boolean;
  duration?: number;
}> = ({ children, show, duration = 300 }) => (
  <Transition show={show} animation="fade" duration={duration}>
    {children}
  </Transition>
);

export const SlideTransition: React.FC<{
  children: React.ReactNode;
  show: boolean;
  direction?: 'up' | 'down' | 'left' | 'right';
  duration?: number;
}> = ({ children, show, direction = 'up', duration = 300 }) => (
  <Transition show={show} animation={`slide${direction.charAt(0).toUpperCase() + direction.slice(1)}` as any} duration={duration}>
    {children}
  </Transition>
);

export const ScaleTransition: React.FC<{
  children: React.ReactNode;
  show: boolean;
  duration?: number;
}> = ({ children, show, duration = 300 }) => (
  <Transition show={show} animation="scale" duration={duration}>
    {children}
  </Transition>
);

export default Transition;
