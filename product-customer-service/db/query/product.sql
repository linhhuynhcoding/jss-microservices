-- name: ListProducts :many
SELECT * FROM products ORDER BY id LIMIT $1 OFFSET $2;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: CreateProduct :one
INSERT INTO products (
  name, code, category_id, weight, gold_price_at_time, labor_cost, stone_cost, 
  markup_rate, selling_price, warranty_period, image, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
)
RETURNING *;

-- name: UpsertProduct :one
INSERT INTO products (
  name, code, category_id, weight, gold_price_at_time, labor_cost, stone_cost, 
  markup_rate, selling_price, warranty_period, image, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  category_id = EXCLUDED.category_id,
  weight = EXCLUDED.weight,
  gold_price_at_time = EXCLUDED.gold_price_at_time,
  labor_cost = EXCLUDED.labor_cost,
  stone_cost = EXCLUDED.stone_cost,
  markup_rate = EXCLUDED.markup_rate,
  selling_price = EXCLUDED.selling_price,
  warranty_period = EXCLUDED.warranty_period,
  image = EXCLUDED.image,
  updated_at = NOW()
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;
