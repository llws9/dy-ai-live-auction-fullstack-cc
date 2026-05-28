import { useFeatureIsOn, useFeatureValue } from '@growthbook/growthbook-react';

/**
 * 检查特性开关是否开启
 * @param featureKey 特性键名
 * @returns boolean
 */
export function useFeatureIsOnByKey(featureKey: string): boolean {
  return useFeatureIsOn(featureKey);
}

/**
 * 获取特性值
 * @param featureKey 特性键名
 * @param defaultValue 默认值
 * @returns 特性值
 */
export function useFeatureValueByKey<T extends string | number | boolean | null>(
  featureKey: string,
  defaultValue: T
): T {
  return useFeatureValue(featureKey, defaultValue) as T;
}

/**
 * 获取 UI 实验变体（用于父子实验）
 * @param parentKey 父实验键名
 * @param childKey 子实验键名
 * @returns { parentVariant, childVariant }
 */
export function useExperimentVariant(
  parentKey: string,
  childKey?: string
): { parentVariant: string | null; childVariant: string | null } {
  const parentValue = useFeatureValue(parentKey, null);
  const childValue = childKey ? useFeatureValue(childKey, null) : null;

  return {
    parentVariant: parentValue as string | null,
    childVariant: childValue as string | null,
  };
}