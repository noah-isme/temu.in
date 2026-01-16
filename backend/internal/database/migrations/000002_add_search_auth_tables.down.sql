BEGIN;

DROP INDEX IF EXISTS idx_services_category_real;
ALTER TABLE services DROP COLUMN IF EXISTS category;
DROP INDEX IF EXISTS idx_services_price;
DROP INDEX IF EXISTS idx_services_name;

DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS password_resets;

ALTER TABLE users DROP COLUMN IF EXISTS verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS is_verified;

COMMIT;
