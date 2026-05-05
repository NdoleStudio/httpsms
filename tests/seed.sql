-- Seed test data for integration tests
-- Run AFTER GORM has migrated the schema (i.e., after API starts)

-- Test user
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'test-user-id',
    'test@httpsms.com',
    'test-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- System user (for event queue auth)
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'system-user-id',
    'system@httpsms.com',
    'system-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;
