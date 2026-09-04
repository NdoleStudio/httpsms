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

-- MCP integration test user. Every functional MCP OAuth flow authenticates as
-- this Firebase UID. Its primary API key is never rotated by any test, so the
-- suite can be re-run against a live stack without re-seeding.
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

-- Dedicated user for the MCP API-key rotation tests. It is the ONLY user whose
-- primary API key is ever rotated, so rotation can never invalidate a key any
-- other test authenticates with. The rotation tests never assume the seeded
-- value below is still current: they always derive the current key by rotating
-- once first, which is what makes the suite re-runnable without a reset.
INSERT INTO users (id, email, api_key, timezone, subscription_name, created_at, updated_at)
VALUES (
    'mcp-rotation-user-id',
    'mcp-rotation@httpsms.com',
    'mcp-rotation-user-api-key',
    'UTC',
    'pro-monthly',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Dedicated user for the MCP user-data isolation assertions. Nothing in the
-- suite ever creates a phone, thread, or message for this user, so "this user
-- sees nothing" is a stable, meaningful assertion on every run.
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

-- Immutable seeded MCP data owned by 'mcp-test-user-id'. No test ever mutates
-- or deletes these two rows: they are what the MCP user-data isolation test
-- asserts the MCP user can see and the isolated user cannot, and what the
-- suite's preflight validates before running any MCP test. The +1888555xxxx
-- range can never collide with the +1800555xxxx numbers randomPhoneNumber()
-- generates at runtime.
INSERT INTO phones (
    id, user_id, phone_number, messages_per_minute, sim,
    max_send_attempts, message_expiration_seconds, unarchive_thread,
    created_at, updated_at
) VALUES (
    'a1b2c3d4-0000-4000-8000-000000000001',
    'mcp-test-user-id',
    '+18885550101',
    60,
    'SIM1',
    2,
    600,
    false,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

INSERT INTO message_threads (
    id, user_id, owner, contact, is_archived, unread_count,
    color, status, last_message_content,
    created_at, updated_at, order_timestamp
) VALUES (
    'a1b2c3d4-0000-4000-8000-000000000002',
    'mcp-test-user-id',
    '+18885550101',
    '+18885550202',
    false,
    0,
    'indigo',
    'received',
    'Seeded MCP integration thread',
    NOW(),
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
