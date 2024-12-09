package main

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	rides := []*Ride{}
	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 10`); err != nil {
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
	}

	chairJoinChairLocation := []*ChairJoinChairLocation{}
	if err := db.SelectContext(ctx, &chairJoinChairLocation, `
		 SELECT chairs.*, chair_locations.latitude, chair_locations.longitude FROM chairs
				INNER JOIN chair_locations ON chairs.id = chair_locations.chair_id
				WHERE chairs.is_active = TRUE;
	`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	for _, ride := range rides {
		// 一番近い椅子を探す
		var matched *ChairJoinChairLocation
		var idx int
		minDistance := math.MaxInt
		for i, chair := range chairJoinChairLocation {
			distance := abs(chair.Latitude-ride.PickupLatitude) + abs(chair.Longitude-ride.PickupLongitude)
			if matched == nil || distance < minDistance {
				matched = chair
				minDistance = distance
				idx = i
			}
		}

		empty := false
		if err := db.GetContext(ctx, &empty, "SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE", matched.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if !empty {
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			// マッチング成功
			// スライスからマッチングした椅子を削除
			chairJoinChairLocation = append(chairJoinChairLocation[:idx], chairJoinChairLocation[idx+1:]...)
			if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}

		if len(chairJoinChairLocation) == 0 {
			break
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
