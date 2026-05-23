import { useEffect, useState } from 'react';

/**
 * 动画状态控制 Hook
 * @param isActive - 是否激活动画
 * @param enterAnimation - 进入动画名称
 * @param exitAnimation - 退出动画名称
 * @param duration - 动画持续时间(ms)
 */
export const useAnimation = (
  isActive: boolean,
  enterAnimation: string = 'fadeIn',
  exitAnimation: string = 'fadeOut',
  duration: number = 300
) => {
  const [shouldRender, setShouldRender] = useState(isActive);
  const [animationClass, setAnimationClass] = useState('');

  useEffect(() => {
    if (isActive) {
      setShouldRender(true);
      setAnimationClass(enterAnimation);
    } else if (shouldRender) {
      setAnimationClass(exitAnimation);
      const timer = setTimeout(() => {
        setShouldRender(false);
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [isActive, enterAnimation, exitAnimation, duration, shouldRender]);

  return { shouldRender, animationClass };
};

/**
 * 组件挂载动画 Hook
 * @param animation - 动画名称
 * @param delay - 延迟时间(ms)
 */
export const useMountAnimation = (animation: string = 'slideUp', delay: number = 0) => {
  const [isAnimated, setIsAnimated] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsAnimated(true);
    }, delay);
    return () => clearTimeout(timer);
  }, [delay]);

  return isAnimated ? animation : '';
};

/**
 * 交错动画 Hook - 用于列表项依次动画
 * @param itemCount - 列表项数量
 * @param baseDelay - 基础延迟(ms)
 * @param staggerDelay - 每项递增延迟(ms)
 */
export const useStaggerAnimation = (
  itemCount: number,
  baseDelay: number = 0,
  staggerDelay: number = 50
) => {
  const [visibleItems, setVisibleItems] = useState<Set<number>>(new Set());

  useEffect(() => {
    setVisibleItems(new Set());
    const timers: ReturnType<typeof setTimeout>[] = [];

    for (let i = 0; i < itemCount; i++) {
      const timer = setTimeout(() => {
        setVisibleItems(prev => new Set(prev).add(i));
      }, baseDelay + i * staggerDelay);
      timers.push(timer);
    }

    return () => timers.forEach(clearTimeout);
  }, [itemCount, baseDelay, staggerDelay]);

  return (index: number) => visibleItems.has(index);
};

/**
 * 滚动触发动画 Hook
 * @param threshold - 触发阈值 (0-1)
 */
export const useScrollAnimation = <T extends HTMLElement>(
  threshold: number = 0.1
) => {
  const [ref, setRef] = useState<React.RefObject<T> | null>(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (!ref?.current) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          if (ref.current) {
            observer.unobserve(ref.current);
          }
        }
      },
      { threshold }
    );

    if (ref.current) {
      observer.observe(ref.current);
    }
    return () => observer.disconnect();
  }, [threshold, ref]);

  return { setRef, isVisible };
};
