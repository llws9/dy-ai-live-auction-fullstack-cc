import { useFeatureIsOn, useFeatureValue } from '@growthbook/growthbook-react';

/**
 * 检查特性开关是否开启
 */
export function useFeatureOn(featureKey: string): boolean {
  return useFeatureIsOn(featureKey);
}

/**
 * 获取特性值
 */
export function useFeatureVal<T extends string | number | boolean | null>(
  featureKey: string,
  defaultValue: T
): T {
  return useFeatureValue(featureKey, defaultValue) as T;
}

/**
 * 获取实验变体（用于父子实验）
 */
export function useExperimentLayer(
  parentKey: string,
  childKey?: string
): { parentVariant: string | null; childVariant: string | null } {
  const parentValue = useFeatureValue(parentKey, null);
  const childValue = useFeatureValue(childKey || '__disabled_child_experiment__', null);

  return {
    parentVariant: parentValue as string | null,
    childVariant: childKey ? (childValue as string | null) : null,
  };
}
