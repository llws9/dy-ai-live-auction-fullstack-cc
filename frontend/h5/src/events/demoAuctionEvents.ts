export const DEMO_CONCURRENT_BIDS_COMPLETED_EVENT = 'demo:concurrent-bids-completed';

export interface DemoConcurrentBidsCompletedDetail {
  auctionId: number;
  highestAmount: string;
}

export function dispatchDemoConcurrentBidsCompleted(detail: DemoConcurrentBidsCompletedDetail) {
  window.dispatchEvent(new CustomEvent<DemoConcurrentBidsCompletedDetail>(
    DEMO_CONCURRENT_BIDS_COMPLETED_EVENT,
    { detail },
  ));
}
