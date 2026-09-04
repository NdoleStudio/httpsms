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

-- Test user for API key rotation tests (isolated to avoid mutating the shared test user)
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'rotate-test-user-id',
    'rotate-test@httpsms.com',
    'rotate-test-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- MCP integration test user. Every MCP OAuth flow authenticates as this
-- Firebase UID, so the MCP API-key rotation tests can rotate its primary key
-- without invalidating the shared 'test-user-api-key' used by every other
-- integration test.
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'mcp-test-user-id',
    'mcp-test@httpsms.com',
    'mcp-test-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Dedicated user for the MCP rate-limit test. Rate-limit budgets are per user
-- and per tool, so exhausting a bucket here can never starve the tools the
-- functional MCP tests call as 'mcp-test-user-id'.
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'mcp-rate-limit-user-id',
    'mcp-rate-limit@httpsms.com',
    'mcp-rate-limit-user-api-key',
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
