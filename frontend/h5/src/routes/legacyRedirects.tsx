import { Navigate, useParams } from 'react-router-dom';

export function LegacyAuctionRedirect() {
  const { id } = useParams<{ id: string }>();

  return <Navigate to={id ? `/detail?id=${encodeURIComponent(id)}` : '/detail'} replace />;
}

export function LegacyResultRedirect() {
  const { id } = useParams<{ id: string }>();

  return <Navigate to={id ? `/result?id=${encodeURIComponent(id)}` : '/result'} replace />;
}
