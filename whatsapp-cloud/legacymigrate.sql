-- >>>>>>>>>>>>>>>>>>>>>>>>>> whatsapp-cloud/legacymigrate.sql <<<<<<<<<<<<<<<<<<<<<<<<<<<
-- This file is used to migrate legacy data from the old WhatsApp Cloud database schema to
-- the new schema.
-- It is executed as part of the upgrade process to ensure compatibility with the latest version
-- of the database.

-- Change the Type of membership in mx_user_profile_old, we need to create a new type to
-- avoid conflicts, and then alter the column to use the new type.
CREATE TYPE membership_old AS ENUM ('join', 'leave', 'invite', 'ban', 'knock');

ALTER TABLE mx_user_profile_old
ALTER COLUMN membership
TYPE membership_old
USING (membership::text::membership_old);

-- >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> portal <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO portal (
    id,
    other_user_id,
    mxid,
    relay_login_id,
    receiver,
    room_type,
    name,
    name_is_custom,
    name_set,
    avatar_set,
    topic_set,
    bridge_id,
    parent_receiver,
    topic,
    avatar_id,
    avatar_hash,
    avatar_mxc,
    in_space,
    metadata
)
SELECT
    portal_old.phone_id || '@s.whatsapp.net', -- id
    portal_old.phone_id, -- other_user_id
    puppet_old.custom_mxid, -- mxid
    portal_old.app_business_id, -- relay_login_id
    portal_old.app_business_id, -- receiver
    'dm', -- room_type
    puppet_old.display_name, -- name
    false, -- name_is_custom
    true, -- name_set
    false, -- avatar_set
    false, -- topic_set
    '', -- bridge_id
    '', -- parent_receiver
    '', -- topic
    '', -- avatar_id
    '', -- avatar_hash
    '', -- avatar_mxc
    false, -- in_space
    '{}' -- metadata
FROM portal_old
INNER JOIN puppet_old on portal_old.phone_id = puppet_old.phone_id;

-- >>>>>>>>>>>>>>>>>>>>>>>>>>>> Ghost <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO ghost (
    bridge_id,
    id,
    name,
    avatar_id,
    avatar_hash,
    avatar_mxc,
    name_set,
    avatar_set,
    contact_info_set,
    is_bot,
    identifiers,
    metadata
)
SELECT
    '', -- bridge_id
    portal_old.phone_id, -- id
    puppet_old.display_name, -- name
    '', -- avatar_id,
    '', -- avatar_hash
    '', -- avatar_mxc
    true, -- name_set
    false, -- avatar_set
    false, -- contact_info_set
    false, -- is_bot
    jsonb_build_array('tel:+' || portal_old.phone_id), -- identifiers
    -- only: postgres
    jsonb_build_object(
        'last_sync', 0
    ) -- metadata
FROM portal_old
INNER JOIN puppet_old on portal_old.phone_id = puppet_old.phone_id;

-- >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> message <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO message (
    id,
    mxid,
    room_id,
    room_receiver,
    sender_id,
    sender_mxid,
    double_puppeted,
    edit_count,
    timestamp,
    bridge_id,
    part_id,
    metadata
)
SELECT
    whatsapp_message_id, -- id
    event_mxid, -- mxid
    phone_id || '@s.whatsapp.net', -- room_id
    app_business_id, -- room_receiver
    phone_id, -- sender_id
    sender, -- sender_mxid
    false, -- double_puppeted
    0, -- edit_count
    EXTRACT(EPOCH FROM (created_at AT TIME ZONE 'UTC')) * 1000, -- timestamp
    '', -- bridge_id
    '', -- part_id,
    '{}' -- metadata
FROM message_old
WHERE event_mxid<>'';

-- >>>>>>>>>>>>>>>>>>>>>>>>>>> user <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO "user" (mxid,management_room,bridge_id)
SELECT
    mxid, -- mxid
    notice_room, -- management_room
    '' -- bridge_id
FROM matrix_user_old
WHERE mxid<>'';

-- >>>>>>>>>>>>>>>>>>>>>>>>>>> user_login <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO user_login (
    bridge_id,
    user_mxid,
    id,
    remote_name,
    space_room,
    metadata,
    remote_profile
)
SELECT
    '', -- bridge_id
    admin_user, -- user_mxid
    waba_id, -- id
    waba_id, -- remote_name
    '', -- space_room
    -- only: postgres
    jsonb_build_object
    (
        'waba_id', waba_id,
        'business_phone_id', business_phone_id,
        'page_access_token', page_access_token
    ), -- metadata
    '{}' -- remote_profile
FROM wb_application
WHERE waba_id<>'';

-- >>>>>>>>>>>>>>>>>>>>>>>>>>> user_portal <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO user_portal (
    user_mxid,
    login_id,
    portal_id,
    in_space,
    preferred,
    portal_receiver,
    bridge_id
)
SELECT
    relay_user_id, -- mxid
    app_business_id, -- login_id
    phone_id || '@s.whatsapp.net', -- portal_id
    false, -- in_space
    false, -- preferred
    app_business_id, -- portal_receiver
    '' -- bridge_id
FROM portal_old
WHERE mxid<>'' AND app_business_id<>'';

-- >>>>>>>>>>>>>>>>>>>>>>>>>>> reaction <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
INSERT INTO reaction (
    message_id,
    message_part_id,
    sender_id,
    sender_mxid,
    emoji_id,
    room_id,
    room_receiver,
    mxid,
    timestamp,
    emoji,
    metadata
)
SELECT
    whatsapp_message_id, -- message_id
    '', -- message_part_id
    sender, -- sender_id
    sender, -- sender_mxid
    '', -- emoji_id
    reaction_old.room_id, -- room_id
    '', -- room_receiver
    event_mxid, -- mxid
    EXTRACT(EPOCH FROM (created_at AT TIME ZONE 'UTC')) * 1000, -- timestamp
    reaction, -- emoji
    '{}' -- metadata
FROM reaction_old
INNER JOIN portal_old on reaction_old.room_id = portal_old.mxid
WHERE event_mxid<>'' AND whatsapp_message_id<>'' AND reaction<>'';


-- >>>>>>>>>>>>>>>>>>>>>>>>>>> Delete the old membership type <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
DROP TYPE IF EXISTS membership;
