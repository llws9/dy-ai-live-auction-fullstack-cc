// jest stub for src/utils/env.ts —— 避免 jest 解析 import.meta
module.exports = {
  IS_DEV: false,
  IS_PROD: false,
  ENV: {
    API_BASE_URL: '',
    GROWTHBOOK_API_HOST: 'http://localhost:3200',
    GROWTHBOOK_CLIENT_KEY: 'dev-client-key',
  },
};
