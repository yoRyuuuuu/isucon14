CREATE INDEX status_created_at_idx ON ride_statuses (ride_id, created_at DESC);

CREATE INDEX chair_id_updated_at_idx ON rides (chair_id, updated_at);

DROP TABLE IF EXISTS chair_distance;

CREATE TABLE chair_distance (
  chair_id VARCHAR(26) NOT NULL COMMENT '椅子ID',
  total_distance INTEGER NOT NULL COMMENT '移動距離',
  total_distance_updated_at DATETIME (6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '更新日時',
  PRIMARY KEY (chair_id)
);

INSERT INTO
  chair_distance (
    chair_id,
    total_distance,
    total_distance_updated_at
  )
SELECT
  chair_id,
  SUM(IFNULL (distance, 0)) AS total_distance,
  MAX(created_at) AS total_distance_updated_at
FROM
  (
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
      chair_locations
  ) tmp
GROUP BY
  chair_id;
