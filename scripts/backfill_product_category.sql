-- T006 一次性历史数据修复脚本
-- 目标：
-- 1. 补齐缺失的 categories 主数据
-- 2. 仅回填可稳定判定的历史空分类商品
-- 3. 输出仍需人工处理的商品清单

START TRANSACTION;

INSERT INTO categories (name, code, description, sort_order, status, created_at, updated_at)
VALUES
  ('数码电子', 'ELECTRONICS', '智能手机、电脑、数码配件等电子产品', 0, 1, NOW(3), NOW(3)),
  ('服装配饰', 'CLOTHING', '男装、女装、鞋帽、箱包等服饰配件', 1, 1, NOW(3), NOW(3)),
  ('家居生活', 'HOME', '家具、家电、厨具、装饰品等生活用品', 2, 1, NOW(3), NOW(3)),
  ('美妆护肤', 'BEAUTY', '化妆品、护肤品、香水等美容产品', 3, 1, NOW(3), NOW(3)),
  ('食品饮料', 'FOOD', '零食、饮料、生鲜、保健品等食品', 4, 1, NOW(3), NOW(3)),
  ('运动户外', 'SPORTS', '运动器材、户外装备、健身用品', 5, 1, NOW(3), NOW(3)),
  ('母婴用品', 'BABY', '婴儿用品、童装、玩具、孕产用品', 6, 1, NOW(3), NOW(3)),
  ('珠宝首饰', 'JEWELRY', '黄金、钻石、翡翠、珍珠等珠宝首饰', 7, 1, NOW(3), NOW(3)),
  ('图书文具', 'BOOKS', '书籍、杂志、文具、办公用品', 8, 1, NOW(3), NOW(3)),
  ('汽车用品', 'AUTOS', '汽车配件、车载用品、保养工具', 9, 1, NOW(3), NOW(3)),
  ('宠物用品', 'PET', '宠物食品、宠物用品、宠物玩具', 10, 1, NOW(3), NOW(3)),
  ('艺术品', 'ART', '字画、雕塑、收藏品、工艺品', 11, 1, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  sort_order = VALUES(sort_order),
  status = VALUES(status),
  updated_at = VALUES(updated_at);

-- 可稳定判定的历史商品：
-- 1) ID=1 名称/描述均指向“珠宝”
-- 2) ID=880201 Fixed Price Demo Jade Bracelet 明确是手镯类珠宝
UPDATE products
SET category_id = (SELECT id FROM categories WHERE code = 'JEWELRY')
WHERE id IN (1, 880201) AND category_id IS NULL;

COMMIT;

-- 验证输出：当前分类主数据与剩余待人工处理商品
SELECT id, name, code, status, sort_order
FROM categories
ORDER BY sort_order, id;

SELECT p.id, p.name, p.category_id, c.name AS category_name
FROM products p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.id IN (1, 880201)
ORDER BY p.id;

SELECT id, name, LEFT(description, 120) AS description_preview, created_at
FROM products
WHERE category_id IS NULL
ORDER BY created_at DESC, id DESC;
