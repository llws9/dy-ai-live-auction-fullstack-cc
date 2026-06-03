import { useLiveChatStore } from '../liveChatStore';

describe('liveChatStore', () => {
  beforeEach(() => {
    useLiveChatStore.getState().reset();
  });

  it('appends incoming messages with cap of 200', () => {
    const { receive } = useLiveChatStore.getState();
    for (let i = 0; i < 250; i++) {
      receive({
        live_stream_id: 1,
        user_id: i,
        user_name: 'u' + i,
        text: 'hi',
        sent_at: Date.now(),
      });
    }
    expect(useLiveChatStore.getState().history).toHaveLength(200);
    expect(useLiveChatStore.getState().history[0].user_id).toBe(50);
  });

  it('cooldown returns true within 1 second of send', () => {
    const { markSent, isCoolingDown } = useLiveChatStore.getState();
    markSent();
    expect(isCoolingDown()).toBe(true);
  });
});
