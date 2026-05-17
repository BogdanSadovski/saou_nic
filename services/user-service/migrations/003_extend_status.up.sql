-- Extend the user.status enum so admin-service can transition users
-- through the full lifecycle (suspend / ban / pending verification)
-- without the existing CHECK rejecting the write.

ALTER TABLE users DROP CONSTRAINT IF EXISTS check_status;

ALTER TABLE users
    ADD CONSTRAINT check_status
    CHECK (status::text = ANY (ARRAY[
        'active'::varchar,
        'inactive'::varchar,
        'suspended'::varchar,
        'banned'::varchar,
        'pending'::varchar
    ]::text[]));
