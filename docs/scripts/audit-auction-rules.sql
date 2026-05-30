-- ============================================================================
-- audit-auction-rules.sql
--
-- 用途：审计 / 修复 `auction_rules.product_id` 的语义一致性。
--
-- 背景（spec C §4.4 / F-C4）：
--   规则归属于 product；`auction_rules.product_id` 的取值必须存在于
--   `products.id`。历史代码曾出现 product_id↔auction_id 的兼容映射，
--   可能让脏数据写入 `auction_rules.product_id`（实际值是 auctions.id）。
--   表结构本身只有 `product_id` 列，没有 auction_id 列，所以脏数据只可
--   能体现为 "auction_rules.product_id 在 products 中不存在，但在
--   auctions.id 中存在"。
--
-- 使用步骤：
--   1) 先跑 §1 / §2 的 SELECT，落盘结果到 CSV，确认数量与样本。
--   2) §3 修复语句默认在事务里、且只在数量很小时手动执行；先用 §3.0
--      的 SELECT 预览将要回写的值。
--   3) §4 复跑 §1 / §2 的 SELECT，确认结果为 0 行。
--
-- 注意：所有写操作都包在事务里；确认无误再 COMMIT。
-- ============================================================================


-- §1 检测：product_id 在 products 中不存在的孤儿规则 -------------------------
-- 期望结果：空
SELECT r.id          AS rule_id,
       r.product_id  AS rule_product_id,
       r.created_at  AS rule_created_at
FROM   auction_rules AS r
LEFT JOIN products  AS p ON p.id = r.product_id
WHERE  p.id IS NULL
ORDER BY r.id;


-- §2 检测：同一 product 上的重复规则（product_id 期望唯一） ----------------
-- 期望结果：空
SELECT product_id,
       COUNT(*)              AS dup_count,
       GROUP_CONCAT(id ORDER BY id) AS rule_ids
FROM   auction_rules
GROUP  BY product_id
HAVING COUNT(*) > 1;


-- §3 反向修复（仅在 §1 有命中、且命中行的 product_id 实际为 auctions.id 时执行）
-- ----------------------------------------------------------------------------
-- 假设：错误写入的 product_id 值 == auctions.id；通过 auctions.product_id
-- 反查正确归属，再 UPDATE 回写。

-- §3.0 预览：将要回写的 (rule_id, 旧 product_id=auction_id, 新 product_id) ----
SELECT r.id                AS rule_id,
       r.product_id        AS legacy_product_id_eq_auction_id,
       a.product_id        AS correct_product_id
FROM   auction_rules AS r
LEFT JOIN products   AS p ON p.id = r.product_id
JOIN   auctions      AS a ON a.id = r.product_id
WHERE  p.id IS NULL
ORDER BY r.id;

-- §3.1 执行修复（先 START TRANSACTION，确认 §3.0 结果后再 COMMIT） ----------
-- START TRANSACTION;
--
-- UPDATE auction_rules AS r
-- JOIN   auctions      AS a ON a.id = r.product_id
-- LEFT JOIN products   AS p ON p.id = r.product_id
-- SET    r.product_id = a.product_id
-- WHERE  p.id IS NULL;
--
-- -- 复跑 §1 应当返回 0 行
-- -- COMMIT;  -- 或 ROLLBACK;


-- §4 兜底：仍无法修复的孤儿（auctions 中也找不到对应 id） ------------------
-- 这类行已无法从现有数据反推归属，需人工裁决（删除或归档）。
SELECT r.id, r.product_id, r.created_at
FROM   auction_rules AS r
LEFT JOIN products  AS p ON p.id = r.product_id
LEFT JOIN auctions  AS a ON a.id = r.product_id
WHERE  p.id IS NULL
  AND  a.id IS NULL
ORDER BY r.id;
