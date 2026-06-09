import {
  LIVE_ROOM_FOOTPRINTS_KEY,
  getLiveRoomFootprints,
  recordLiveRoomFootprint,
} from '../liveRoomFootprints';

describe('liveRoomFootprints', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.spyOn(Date, 'now').mockReturnValue(1781020000000);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('records a normalized live room footprint', () => {
    recordLiveRoomFootprint({
      live_stream_id: 3,
      name: '玉石夜拍',
      cover: 'https://example.com/cover.jpg',
    });

    expect(getLiveRoomFootprints()).toEqual([
      {
        live_stream_id: 3,
        name: '玉石夜拍',
        cover: 'https://example.com/cover.jpg',
        enteredAt: 1781020000000,
      },
    ]);
  });

  it('deduplicates by live_stream_id and moves the latest entry to the top', () => {
    recordLiveRoomFootprint({ live_stream_id: 1, name: '旧直播', cover: '' });
    jest.spyOn(Date, 'now').mockReturnValue(1781020005000);
    recordLiveRoomFootprint({ live_stream_id: 2, name: '新直播', cover: '' });
    jest.spyOn(Date, 'now').mockReturnValue(1781020010000);
    recordLiveRoomFootprint({ live_stream_id: 1, name: '旧直播更新', cover: 'next.jpg' });

    expect(getLiveRoomFootprints().map((item) => item.live_stream_id)).toEqual([1, 2]);
    expect(getLiveRoomFootprints()[0]).toMatchObject({
      live_stream_id: 1,
      name: '旧直播更新',
      cover: 'next.jpg',
      enteredAt: 1781020010000,
    });
  });

  it('keeps only the latest 10 records', () => {
    for (let i = 1; i <= 12; i += 1) {
      jest.spyOn(Date, 'now').mockReturnValue(1781020000000 + i);
      recordLiveRoomFootprint({ live_stream_id: i, name: `直播 ${i}`, cover: '' });
    }

    const records = getLiveRoomFootprints();
    expect(records).toHaveLength(10);
    expect(records[0].live_stream_id).toBe(12);
    expect(records[9].live_stream_id).toBe(3);
  });

  it('fails closed when stored JSON is invalid', () => {
    localStorage.setItem(LIVE_ROOM_FOOTPRINTS_KEY, '{bad json');
    expect(getLiveRoomFootprints()).toEqual([]);
  });
});
