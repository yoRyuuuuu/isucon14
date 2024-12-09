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
	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	type chairJoinChairLocation struct {
		ChairID    string `db:"id"`
		Latitude  int    `db:"latitude"`
		Longitude int    `db:"longitude"`
	}

	possibleChairs := []*chairJoinChairLocation{}

	query := `SELECT c.id, cl.latitude, cl.longitude FROM chairs AS c INNER JOIN chair_locations AS cl ON c.id = cl.chair_id WHERE c.is_active = TRUE LIMIT 10`

	if err := db.SelectContext(ctx, &possibleChairs, query); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	var matchedID string
	minDistance := math.MaxInt
	empty := false
	for _, chair := range possibleChairs {
		distance := abs(chair.Latitude-ride.PickupLatitude) + abs(chair.Longitude-ride.PickupLongitude)
		if distance < minDistance {
			minDistance = distance
			matchedID = chair.ChairID
		}

		if err := db.GetContext(ctx, &empty, "SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE", matchedID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if empty {
			break
		}
	}
	if !empty {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matchedID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
