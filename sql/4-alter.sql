CREATE INDEX status_created_at_idx ON ride_statuses (ride_id, created_at DESC);

CREATE INDEX chair_id_updated_at_idx ON rides (chair_id, updated_at);

DROP TABLE IF EXISTS distance;

CREATE TABLE distance (
  chair_id VARCHAR(26) NOT NULL COMMENT '割り当てられた椅子ID',
  created_at DATETIME (6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '要求日時',
  latitude INTEGER NOT NULL COMMENT '経度',
  longitude INTEGER NOT NULL COMMENT '緯度',
  distance INTEGER COMMENT '移動距離'
);

DROP TABLE IF EXISTS distance_table;

CREATE TABLE distance_table (
  char_id VARCHAR(26) NOT NULL COMMENT '割り当てられた椅子ID',
  total_distance INTEGER NOT NULL COMMENT '移動距離',
  total_distance_updated_at DATETIME (6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '更新日時'
);

INSERT INTO
  distance (
    chair_id,
    created_at,
    latitude,
    longitude,
    distance
  )
SELECT
  chair_id,
  created_at,
  latitude,
  longitude,
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

SELECT
  chair_id,
  SUM(IFNULL (distance, 0)) AS total_distance,
  MAX(created_at) AS total_distance_updated_at
FROM
  distance
GROUP BY
  chair_id;
