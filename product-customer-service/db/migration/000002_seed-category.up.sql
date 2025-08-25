INSERT INTO product_categories (name)
VALUES 
  ('Nhẫn'),
  ('Dây Chuyền'),
  ('Mặt Dây Chuyền'),
  ('Bông Tai'),
  ('Lắc'),
  ('Vòng'),
  ('Charm'),
  ('Dây Cổ')
ON CONFLICT (name) 
DO UPDATE SET name = EXCLUDED.name;
