export const LIVE_ROOM_FOOTPRINTS_KEY = 'h5.liveRoomFootprints';

const FOOTPRINT_LIMIT = 10;

export interface LiveRoomFootprint {
  live_stream_id: number;
  name: string;
  cover: string;
  enteredAt: number;
}

export type LiveRoomFootprintInput = Omit<LiveRoomFootprint, 'enteredAt'>;

function canUseLocalStorage() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function normalizeRecord(value: unknown): LiveRoomFootprint | null {
  if (!value || typeof value !== 'object') return null;

  const record = value as Partial<LiveRoomFootprint>;
  const liveStreamID = Number(record.live_stream_id);
  const enteredAt = Number(record.enteredAt);

  if (!Number.isFinite(liveStreamID) || liveStreamID <= 0 || !Number.isFinite(enteredAt)) {
    return null;
  }

  return {
    live_stream_id: liveStreamID,
    name: String(record.name || '直播间'),
    cover: String(record.cover || ''),
    enteredAt,
  };
}

export function getLiveRoomFootprints(): LiveRoomFootprint[] {
  if (!canUseLocalStorage()) return [];

  try {
    const raw = window.localStorage.getItem(LIVE_ROOM_FOOTPRINTS_KEY);
    if (!raw) return [];

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];

    return parsed
      .map(normalizeRecord)
      .filter((record): record is LiveRoomFootprint => Boolean(record))
      .sort((a, b) => b.enteredAt - a.enteredAt)
      .slice(0, FOOTPRINT_LIMIT);
  } catch {
    return [];
  }
}

export function recordLiveRoomFootprint(input: LiveRoomFootprintInput) {
  if (!canUseLocalStorage()) return;

  const liveStreamID = Number(input.live_stream_id);
  if (!Number.isFinite(liveStreamID) || liveStreamID <= 0) return;

  const nextRecord: LiveRoomFootprint = {
    live_stream_id: liveStreamID,
    name: input.name || '直播间',
    cover: input.cover || '',
    enteredAt: Date.now(),
  };

  const records = [
    nextRecord,
    ...getLiveRoomFootprints().filter((record) => record.live_stream_id !== liveStreamID),
  ].slice(0, FOOTPRINT_LIMIT);

  try {
    window.localStorage.setItem(LIVE_ROOM_FOOTPRINTS_KEY, JSON.stringify(records));
  } catch {
    // localStorage may be full or disabled; footprints are optional UI state.
  }
}
