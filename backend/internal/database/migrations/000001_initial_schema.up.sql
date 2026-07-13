CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'user'
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    token_hash VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS admin_audits (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    actor_id INTEGER NOT NULL REFERENCES users(id),
    action VARCHAR(255) NOT NULL,
    target VARCHAR(255),
    details TEXT
);

CREATE INDEX IF NOT EXISTS idx_admin_audits_actor_id ON admin_audits(actor_id);

CREATE TABLE IF NOT EXISTS service_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    business_name VARCHAR(255) NOT NULL,
    business_type VARCHAR(50) NOT NULL,
    description TEXT,
    address TEXT NOT NULL,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    pricing_info JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_service_providers_user_id ON service_providers(user_id);
CREATE INDEX IF NOT EXISTS idx_service_providers_business_type ON service_providers(business_type);
CREATE INDEX IF NOT EXISTS idx_service_providers_is_active ON service_providers(is_active);
CREATE INDEX IF NOT EXISTS idx_service_providers_deleted_at ON service_providers(deleted_at);

CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    provider_id UUID NOT NULL REFERENCES service_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    duration INTEGER NOT NULL CHECK (duration > 0),
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_services_provider_id ON services(provider_id);
CREATE INDEX IF NOT EXISTS idx_services_is_active ON services(is_active);
CREATE INDEX IF NOT EXISTS idx_services_deleted_at ON services(deleted_at);

CREATE TABLE IF NOT EXISTS availability_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    provider_id UUID NOT NULL REFERENCES service_providers(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (start_time < end_time)
);

CREATE INDEX IF NOT EXISTS idx_availability_schedules_provider_id ON availability_schedules(provider_id);
CREATE INDEX IF NOT EXISTS idx_availability_schedules_day_of_week ON availability_schedules(day_of_week);

CREATE TABLE IF NOT EXISTS availability_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    provider_id UUID NOT NULL REFERENCES service_providers(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    is_available BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT,
    CHECK ((start_time IS NULL AND end_time IS NULL) OR
           (start_time IS NOT NULL AND end_time IS NOT NULL AND start_time < end_time))
);

CREATE INDEX IF NOT EXISTS idx_availability_exceptions_provider_id ON availability_exceptions(provider_id);
CREATE INDEX IF NOT EXISTS idx_availability_exceptions_date ON availability_exceptions(date);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    intent_id VARCHAR(255) UNIQUE,
    client_secret VARCHAR(255) UNIQUE,
    amount BIGINT NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    status VARCHAR(50) NOT NULL DEFAULT 'created'
);

CREATE INDEX IF NOT EXISTS idx_payments_intent_id ON payments(intent_id);
CREATE INDEX IF NOT EXISTS idx_payments_client_secret ON payments(client_secret);

CREATE TABLE IF NOT EXISTS bookings (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    provider_id UUID NOT NULL REFERENCES service_providers(id),
    service_id UUID REFERENCES services(id),
    customer_name VARCHAR(255),
    booking_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    customer_notes TEXT,
    admin_notes TEXT,
    cancellation_reason TEXT,
    payment_status VARCHAR(20) NOT NULL DEFAULT 'unpaid',
    payment_id INTEGER REFERENCES payments(id),
    CHECK (start_time < end_time),
    UNIQUE (provider_id, booking_date, start_time)
);

CREATE INDEX IF NOT EXISTS idx_bookings_provider_date ON bookings(provider_id, booking_date);
CREATE INDEX IF NOT EXISTS idx_bookings_customer ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    booking_id INTEGER NOT NULL UNIQUE REFERENCES bookings(id),
    provider_id UUID NOT NULL REFERENCES service_providers(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT
);

CREATE INDEX IF NOT EXISTS idx_reviews_provider_id ON reviews(provider_id);
CREATE INDEX IF NOT EXISTS idx_reviews_user_id ON reviews(user_id);
