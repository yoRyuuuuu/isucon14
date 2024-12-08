CREATE INDEX status_created_at_idx ON ride_statuses (ride_id, created_at DESC);

CREATE INDEX chair_id_updated_at_idx ON rides (chair_id, updated_at);

DROP TABLE IF EXISTS distance;

CREATE TABLE distance (
  chair_id VARCHAR(26) NOT NULL COMMENT '割り当てられた椅子ID',
  created_at DATETIME (6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '要求日時',
  distance INTEGER COMMENT '移動距離'
);

INSERT INTO
  distance (chair_id, created_at, distance)
SELECT
  chair_id,
  created_at,
  ABS(
    latitude - LAG (latitude) OVER (
      PARTITION BY
        chair_id
      ORDER BY
        created_at
    )
  ) + ABS(
    longitude - LAG (longitude) OVER (
      PARTITION BY
        chair_id
      ORDER BY
        created_at
    )
  ) AS distance
FROM
  chair_locations;
