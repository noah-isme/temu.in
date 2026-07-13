CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_provider_id_booking_date_start_time_key;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_no_provider_time_overlap
    EXCLUDE USING gist (
        provider_id WITH =,
        tsrange(booking_date + start_time, booking_date + end_time, '[)') WITH &&
    )
    WHERE (deleted_at IS NULL AND status <> 'cancelled');
