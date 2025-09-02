package domain

import (
    "time"
)

// OrderItem represents a purchased product within an order.  It records
// the product identifier, quantity and the unit price at the time of
// purchase.  Storing unit_price allows accurate reconciliation even if
// product prices change later.
type OrderItem struct {
    ProductID int32   `bson:"product_id" json:"product_id"`
    Quantity  int32   `bson:"quantity" json:"quantity"`
    UnitPrice float64 `bson:"unit_price" json:"unit_price"`
}

// Order represents an order placed by a customer and created by a staff
// member.  In addition to the MongoDB ObjectID (omitted here), each order
// has a sequential integer ID used when interfacing with the product and
// loyalty services.  Voucher codes applied to the order are persisted.
type Order struct {
    OrderID       int32       `bson:"order_id" json:"order_id"`
    CustomerID    int32       `bson:"customer_id" json:"customer_id"`
    StaffID       string      `bson:"staff_id" json:"staff_id"`
    Items         []OrderItem `bson:"items" json:"items"`
    VoucherCodes  []string    `bson:"voucher_codes" json:"voucher_codes"`
    TotalPrice    float64     `bson:"total_price" json:"total_price"`
    DiscountAmount float64    `bson:"discount_amount" json:"discount_amount"`
    FinalPrice    float64     `bson:"final_price" json:"final_price"`
    ShippingCost  float64     `bson:"shipping_cost" json:"shipping_cost"`
    CreatedAt     time.Time   `bson:"created_at" json:"created_at"`
}