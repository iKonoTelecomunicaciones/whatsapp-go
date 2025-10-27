ALTER TABLE IF EXISTS mx_user_profile_old
ALTER COLUMN membership
TYPE membership
USING (membership::text::membership);

-- Migrate user profiles if old table exists and has data
INSERT INTO mx_user_profile (room_id, user_id, membership, displayname, avatar_url)
SELECT room_id, user_id, membership, COALESCE(displayname, ''), COALESCE(avatar_url, '')
FROM mx_user_profile_old;

-- Migrate room state if old table exists and has data
INSERT INTO mx_room_state (room_id, power_levels, encryption, members_fetched)
SELECT
    room_id,
    COALESCE(power_levels::jsonb, '{}'::jsonb),
    COALESCE(encryption::jsonb, '{}'::jsonb),
    COALESCE(has_full_member_list, false)
FROM mx_room_state_old;