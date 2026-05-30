import React, { useState, useEffect, useRef } from 'react';

interface LazyImageProps {
  src: string;
  alt: string;
  placeholder?: string;
  style?: React.CSSProperties;
  className?: string;
  onLoad?: () => void;
  onError?: () => void;
}

const LazyImage: React.FC<LazyImageProps> = ({
  src,
  alt,
  placeholder = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 180"%3E%3Crect fill="%23f0f0f0" width="400" height="180"/%3E%3Ctext fill="%23999" font-family="sans-serif" font-size="18" x="50%25" y="50%25" dominant-baseline="middle" text-anchor="middle"%3E加载中...%3C/text%3E%3C/svg%3E',
  style,
  className,
  onLoad,
  onError,
}) => {
  const [imageSrc, setImageSrc] = useState<string>(placeholder);
  const [isLoaded, setIsLoaded] = useState(false);
  const [isInView, setIsInView] = useState(false);
  const imgRef = useRef<HTMLImageElement>(null);

  useEffect(() => {
    // 创建 Intersection Observer
    const currentImg = imgRef.current;
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsInView(true);
            // 一旦进入视口，停止观察
            if (currentImg) {
              observer.unobserve(currentImg);
            }
          }
        });
      },
      {
        rootMargin: '50px', // 提前50px开始加载
        threshold: 0.01, // 1% 进入视口即触发
      }
    );

    // 开始观察
    if (currentImg) {
      observer.observe(currentImg);
    }

    // 清理
    return () => {
      if (currentImg) {
        observer.unobserve(currentImg);
      }
    };
  }, []);

  useEffect(() => {
    // 当进入视口时，加载真实图片
    if (isInView && src) {
      const img = new Image();
      img.src = src;
      img.onload = () => {
        setImageSrc(src);
        setIsLoaded(true);
        if (onLoad) onLoad();
      };
      img.onerror = () => {
        console.error('图片加载失败:', src);
        if (onError) onError();
      };
    }
  }, [isInView, src, onLoad, onError]);

  return (
    <img
      ref={imgRef}
      src={imageSrc}
      alt={alt}
      style={{
        ...style,
        transition: 'opacity 0.3s ease-in-out',
        opacity: isLoaded ? 1 : 0.7,
      }}
      className={className}
    />
  );
};

export default LazyImage;
