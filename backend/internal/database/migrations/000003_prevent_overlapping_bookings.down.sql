ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_no_provider_time_overlap;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_provider_id_booking_date_start_time_key
    UNIQUE (provider_id, booking_date, start_time);
