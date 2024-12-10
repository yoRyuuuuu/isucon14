package main

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…

	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	type ChairJoinChairLocation struct {
		Chair
		Latitude  int `db:"latitude"`
		Longitude int `db:"longitude"`
		Distance  int
	}

	// アクティブな椅子の最新の位置情報を取得
	query := `
		SELECT chairs.*, chair_locations.latitude, chair_locations.longitude FROM chairs INNER JOIN
			(SELECT chair_id, max(created_at) as latest
			FROM chair_locations
			GROUP BY chair_id) AS sub ON chairs.id = sub.chair_id
			INNER JOIN chair_locations ON chair_locations.chair_id = chairs.id AND chair_locations.created_at = sub.latest
			WHERE chairs.is_active = TRUE
	`

	chairs := []ChairJoinChairLocation{}
	if err := db.SelectContext(ctx, &chairs, query); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// 乗車地点からの距離でソート
	sort.Slice(chairs, func(i, j int) bool {
		left := chairs[i]
		right := chairs[j]
		return calculateDistance(ride.DestinationLatitude, ride.DestinationLongitude, left.Latitude, left.Longitude) < calculateDistance(ride.DestinationLatitude, ride.DestinationLongitude, right.Latitude, right.Longitude)
	})

	// 一番近い椅子を探す
	for _, chair := range chairs {
		empty := false
		if err := db.GetContext(ctx, &empty, `SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE`, chair.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if !empty {
			if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", chair.ID, ride.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
