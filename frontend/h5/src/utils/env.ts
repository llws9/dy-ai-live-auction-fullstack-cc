// 统一封装运行环境标志，避免 prod 代码直接依赖 vite 专属的 import.meta，
// 让 jest（CommonJS）和 vite（ESM）都能解析。
//
// 在 jest 中，moduleNameMapper 会把本文件替换为常量 stub，避免触发 import.meta 解析错误。

export const IS_DEV: boolean =
  typeof import.meta !== 'undefined' && (import.meta as any)?.env?.DEV === true;

export const IS_PROD: boolean =
  typeof import.meta !== 'undefined' && (import.meta as any)?.env?.PROD === true;

export const ENV = {
  API_BASE_URL: ((import.meta as any)?.env?.VITE_API_BASE_URL as string | undefined) ?? '',
  GROWTHBOOK_API_HOST:
    ((import.meta as any)?.env?.VITE_GROWTHBOOK_API_HOST as string | undefined) ??
    'http://localhost:3200',
  GROWTHBOOK_CLIENT_KEY:
    ((import.meta as any)?.env?.VITE_GROWTHBOOK_CLIENT_KEY as string | undefined) ?? 'dev-client-key',
};
