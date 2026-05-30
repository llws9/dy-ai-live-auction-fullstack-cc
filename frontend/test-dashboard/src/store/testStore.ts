import { create } from 'zustand';
import type { TestResult } from '../api/test';

interface TestStoreState {
  current: TestResult | null;
  setCurrent: (r: TestResult | null) => void;
}

// 当前正在跑的任务（最近一次启动的）
export const useTestStore = create<TestStoreState>((set) => ({
  current: null,
  setCurrent: (r) => set({ current: r }),
}));
