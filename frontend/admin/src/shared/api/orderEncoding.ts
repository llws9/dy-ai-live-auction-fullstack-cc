import { normalizeUserName } from './userEncoding';

type OrderLike = object & {
  user_id?: number | null;
  user_name?: string | null;
};

export function normalizeOrderText<T extends OrderLike>(order: T): T {
  const userName = normalizeUserName(order.user_id, order.user_name);
  return {
    ...order,
    ...(userName ? { user_name: userName } : {}),
  };
}

export function normalizeOrderListResponse<T extends OrderLike, R extends { list?: T[] }>(response: R): R {
  if (!Array.isArray(response.list)) {
    return response;
  }

  return {
    ...response,
    list: response.list.map(normalizeOrderText),
  };
}
