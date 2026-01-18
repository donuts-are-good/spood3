.headers on
.mode column

-- Serum ad visibility depends on there being at least one shop_items row with item_type='serum'.
-- This reports whether those rows exist and lists them.

SELECT
  COUNT(*) AS serum_item_count
FROM shop_items
WHERE item_type = 'serum';

SELECT
  id,
  name,
  item_type,
  price,
  emoji,
  effect_value,
  created_at
FROM shop_items
WHERE item_type = 'serum'
ORDER BY id;









