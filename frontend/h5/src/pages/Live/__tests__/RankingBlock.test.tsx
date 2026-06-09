import { render, screen } from '@testing-library/react';
import RankingBlock from '../RankingBlock';

describe('RankingBlock', () => {
  const formatMoney = (amount: number) => amount.toString();

  it('does not mark the current user as leading when ranked second', () => {
    render(
      <RankingBlock
        ranking={[
          { rank: 1, user_id: 101, user_name: '领先买家', amount: 120 },
          { rank: 2, user_id: 9, user_name: '测试用户', amount: 110 },
        ]}
        isAuthenticated
        userId={9}
        myRankIndex={1}
        myBidAmount={110}
        formatMoney={formatMoney}
      />,
    );

    expect(screen.getByText('我自己')).toBeInTheDocument();
    expect(screen.queryByText('我自己 (当前领先)')).not.toBeInTheDocument();
  });
});
