ALTER TABLE users DROP CONSTRAINT IF EXISTS check_status;
ALTER TABLE users
    ADD CONSTRAINT check_status
    CHECK (status::text = ANY (ARRAY[
        'active'::varchar,
        'inactive'::varchar,
        'banned'::varchar
    ]::text[]));
