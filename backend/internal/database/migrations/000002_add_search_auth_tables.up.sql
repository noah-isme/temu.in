BEGIN;

-- 1. Add Email Verification to Users
ALTER TABLE users ADD COLUMN is_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN verification_token VARCHAR(255);

-- 2. Password Resets Table
CREATE TABLE password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_password_resets_token_hash ON password_resets(token_hash);
CREATE INDEX idx_password_resets_user_id ON password_resets(user_id);

-- 3. Favorites Table
CREATE TABLE favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    service_id UUID REFERENCES services(id) ON DELETE CASCADE,
    provider_id UUID REFERENCES service_providers(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_favorite_target CHECK (service_id IS NOT NULL OR provider_id IS NOT NULL),
    CONSTRAINT uq_favorite_service UNIQUE (user_id, service_id),
    CONSTRAINT uq_favorite_provider UNIQUE (user_id, provider_id)
);

CREATE INDEX idx_favorites_user_id ON favorites(user_id);

-- 4. Search Indexes for Services
ALTER TABLE services ADD COLUMN category VARCHAR(100);
CREATE INDEX idx_services_name ON services USING gin(to_tsvector('english', name)); -- Full text search support
CREATE INDEX idx_services_price ON services(price);
CREATE INDEX idx_services_category ON services(category) WHERE category IS NOT NULL;

COMMIT;
