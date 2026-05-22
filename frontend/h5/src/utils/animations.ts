// utils/animations.ts

export interface AnimationConfig {
  keyframes: Keyframe[];
  options: KeyframeAnimationOptions;
}

// 出价成功动画
export const bidSuccessAnimation: AnimationConfig = {
  keyframes: [
    { transform: 'scale(1)', opacity: 1 },
    { transform: 'scale(1.15)', opacity: 0.9 },
    { transform: 'scale(1)', opacity: 1 },
  ],
  options: {
    duration: 300,
    easing: 'ease-out',
    fill: 'forwards',
  },
};

// 价格变化动画
export const priceChangeAnimation: AnimationConfig = {
  keyframes: [
    { transform: 'translateY(0)', opacity: 1 },
    { transform: 'translateY(-10px)', opacity: 0.8 },
    { transform: 'translateY(0)', opacity: 1 },
  ],
  options: {
    duration: 400,
    easing: 'ease-out',
    fill: 'forwards',
  },
};

// 延时触发动画
export const delayTriggeredAnimation: AnimationConfig = {
  keyframes: [
    { transform: 'scale(1)', boxShadow: '0 2px 8px rgba(0,0,0,0.1)' },
    { transform: 'scale(1.05)', boxShadow: '0 4px 16px rgba(255,77,79,0.3)' },
    { transform: 'scale(1)', boxShadow: '0 2px 8px rgba(0,0,0,0.1)' },
  ],
  options: {
    duration: 600,
    easing: 'ease-out',
    fill: 'forwards',
  },
};

// 竞拍结束动画
export const auctionEndedAnimation: AnimationConfig = {
  keyframes: [
    { transform: 'scale(1)', opacity: 1 },
    { transform: 'scale(1.2)', opacity: 0.7 },
    { transform: 'scale(1.1)', opacity: 0.9 },
    { transform: 'scale(1)', opacity: 1 },
  ],
  options: {
    duration: 800,
    easing: 'ease-out',
    fill: 'forwards',
  },
};

// 应用动画
export const applyAnimation = (
  element: HTMLElement,
  animation: AnimationConfig
): Animation => {
  return element.animate(animation.keyframes, animation.options);
};

// 创建CSS动画类
export const createAnimationStyles = (): string => `
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @keyframes pulse {
    0% { transform: scale(1); }
    50% { transform: scale(1.05); }
    100% { transform: scale(1); }
  }

  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-5px); }
    75% { transform: translateX(5px); }
  }

  .animate-fade-in {
    animation: fadeIn 0.3s ease-out;
  }

  .animate-pulse {
    animation: pulse 0.6s ease-out;
  }

  .animate-shake {
    animation: shake 0.3s ease-out;
  }

  .hardware-accelerated {
    transform: translateZ(0);
    will-change: transform, opacity;
    backface-visibility: hidden;
  }
`;

// 性能监控
export class AnimationPerformanceMonitor {
  private fps: number = 60;
  private frameCount: number = 0;
  private lastTime: number = performance.now();
  private callback?: (fps: number) => void;

  start(callback?: (fps: number) => void) {
    this.callback = callback;
    this.measure();
  }

  private measure = () => {
    this.frameCount++;
    const currentTime = performance.now();
    const elapsed = currentTime - this.lastTime;

    if (elapsed >= 1000) {
      this.fps = Math.round((this.frameCount * 1000) / elapsed);
      this.frameCount = 0;
      this.lastTime = currentTime;

      if (this.callback) {
        this.callback(this.fps);
      }
    }

    requestAnimationFrame(this.measure);
  };

  getFPS(): number {
    return this.fps;
  }

  shouldAnimate(): boolean {
    return this.fps >= 30;
  }
}
