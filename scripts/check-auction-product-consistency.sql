-- 只读检查：auction.product_id 在 products 中缺失。
SELECT a.id AS auction_id, a.product_id, a.creator_id
FROM auctions a
LEFT JOIN products p ON p.id = a.product_id
WHERE p.id IS NULL;

-- 只读检查：auction.creator_id 与 products.owner_id 不一致。
SELECT a.id AS auction_id, a.product_id, a.creator_id, p.owner_id
FROM auctions a
JOIN products p ON p.id = a.product_id
WHERE a.creator_id IS NOT NULL
  AND p.owner_id IS NOT NULL
  AND a.creator_id <> p.owner_id;

-- 只读检查：同一商品存在多条活跃竞拍。
SELECT product_id, COUNT(*) AS active_count
FROM auctions
WHERE status IN (0, 1, 2)
GROUP BY product_id
HAVING COUNT(*) > 1;
