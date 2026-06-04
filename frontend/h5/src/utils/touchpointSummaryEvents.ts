export const TOUCHPOINT_SUMMARY_INVALIDATED_EVENT = 'touchpoint-summary-invalidated';

export function notifyTouchpointSummaryInvalidated() {
  window.dispatchEvent(new CustomEvent(TOUCHPOINT_SUMMARY_INVALIDATED_EVENT));
}
