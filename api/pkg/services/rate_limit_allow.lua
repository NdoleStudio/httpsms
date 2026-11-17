local key = KEYS[1]
local limit, cost, window = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3])
local stored = redis.call('GET', key)
local count, ttl = 0, 0

if stored then
    count = tonumber(stored)
    if not string.match(stored, '^%d+$') or not count or count > 9007199254740991
        or (#stored > 1 and string.sub(stored, 1, 1) == '0') then
        return redis.error_reply('invalid hourly rate limit counter')
    end
    ttl = redis.call('PTTL', key)
    if ttl < 0 or ttl > window then
        return redis.error_reply('invalid hourly rate limit expiry')
    end
end

if cost > limit - count then
    return {0, ttl}
end

if stored then
    redis.call('INCRBY', key, cost)
else
    redis.call('SET', key, cost, 'PX', window)
end
return {1, 0}
