CREATE INDEX status_created_at_idx ON ride_statuses(ride_id ,created_at DESC);

CREATE INDEX chair_id_updated_at_idx ON rides (chair_id, updated_at);
