import { repairUtf8Mojibake } from '../textEncoding';

describe('repairUtf8Mojibake', () => {
  it('repairs cp1252-style utf8 mojibake back to Chinese', () => {
    expect(repairUtf8Mojibake('è€è±é’»çŸ³æˆ’æŒ‡')).toBe('老花钻石戒指');
    expect(repairUtf8Mojibake('ç²¾é€‰ä¸»çŸ³ï¼Œç«å½©å‡ºè‰²')).toBe('精选主石，火彩出色');
  });

  it('keeps valid Chinese unchanged', () => {
    expect(repairUtf8Mojibake('清代青花瓷瓶')).toBe('清代青花瓷瓶');
  });

  it('keeps ascii text unchanged', () => {
    expect(repairUtf8Mojibake('Auction Lot #9')).toBe('Auction Lot #9');
  });
});
