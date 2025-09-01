-- name: CreateOrderRecord :one
INSERT INTO order_record (
  customer_id, product_id, order_id, quantity, status
) VALUES (
  $1, $2, $3, $4, COALESCE($5, 'pending')
)
RETURNING *;

-- name: GetOrderRecord :one
SELECT *
FROM order_record
WHERE customer_id = $1
  AND product_id  = $2
  AND order_id    = $3;

-- name: ListOrderRecords :many
SELECT *
FROM order_record
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateOrderRecord :one
UPDATE order_record
SET 
  status     = COALESCE($4, status),
  updated_at = now()
WHERE customer_id = $1
  AND product_id  = $2
  AND order_id    = $3
RETURNING *;

-- name: DeleteOrderRecord :one
DELETE FROM order_record
WHERE customer_id = $1
  AND product_id  = $2
  AND order_id    = $3
RETURNING *;
