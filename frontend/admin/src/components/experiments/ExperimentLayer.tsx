import React from 'react';
import { useExperimentVariant, useFeatureIsOnByKey, useFeatureValueByKey } from '@/shared/growthbook/useFeature';

interface ExperimentLayerProps {
  parentKey: string;
  childKey?: string;
  children: (parentVariant: string | null, childVariant: string | null) => React.ReactNode;
}

/**
 * ExperimentLayer 组件
 * 用于实现父子实验的 layering 功能
 *
 * 使用示例:
 * <ExperimentLayer parentKey="new-auction-ui-theme" childKey="bid-button-color">
 *   {(parentVariant, childVariant) => (
 *     parentVariant === 'modern'
 *       ? <ModernUI buttonColor={childVariant} />
 *       : <ClassicUI />
 *   )}
 * </ExperimentLayer>
 */
export function ExperimentLayer({
  parentKey,
  childKey,
  children
}: ExperimentLayerProps) {
  const { parentVariant, childVariant } = useExperimentVariant(parentKey, childKey);

  return <>{children(parentVariant, childVariant)}</>;
}

interface FeatureFlagProps {
  featureKey: string;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

/**
 * FeatureFlag 组件
 * 根据特性开关决定是否渲染内容
 *
 * 使用示例:
 * <FeatureFlag featureKey="new-feature">
 *   <NewFeatureUI />
 * </FeatureFlag>
 */
export function FeatureFlag({
  featureKey,
  children,
  fallback = null
}: FeatureFlagProps) {
  const isOn = useFeatureIsOnByKey(featureKey);

  if (isOn) {
    return <>{children}</>;
  }

  return <>{fallback}</>;
}

interface FeatureValueProps<T> {
  featureKey: string;
  defaultValue: T;
  children: (value: T) => React.ReactNode;
}

/**
 * FeatureValue 组件
 * 获取特性值并传递给子组件
 *
 * 使用示例:
 * <FeatureValue featureKey="button-color" defaultValue="blue">
 *   {(color) => <Button style={{ backgroundColor: color }}>Click</Button>}
 * </FeatureValue>
 */
export function FeatureValue<T extends string | number | boolean | null>({
  featureKey,
  defaultValue,
  children,
}: FeatureValueProps<T>) {
  const value = useFeatureValueByKey(featureKey, defaultValue);

  return <>{children(value)}</>;
}

export default ExperimentLayer;