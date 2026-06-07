import { repairUtf8Mojibake } from '../../utils/textEncoding';

const DEMO_BUYER_NAMES: Record<number, string> = {
  9101: '演示买家A',
  9102: '演示买家B',
};

export function normalizeUserName(userID?: number | null, userName?: string | null): string | undefined {
  if (typeof userName === 'string' && userName.trim()) {
    return repairUtf8Mojibake(userName);
  }
  if (typeof userID === 'number') {
    return DEMO_BUYER_NAMES[userID];
  }
  return undefined;
}
