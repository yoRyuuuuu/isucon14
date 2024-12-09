package main

import (
	"database/sql"
	"errors"
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	ride := &Ride{}
	matched := &Chair{}

	// Combine the selection of an available ride and chair into a single query
	query := `
		SELECT r.id AS ride_id, c.id AS chair_id
		FROM rides r
		JOIN chairs c ON c.is_active = TRUE
		LEFT JOIN ride_statuses rs ON rs.ride_id = r.id
		WHERE r.chair_id IS NULL
		GROUP BY r.id, c.id
		HAVING COUNT(rs.chair_sent_at) < 6
		ORDER BY r.created_at, RAND()
		LIMIT 1
	`
	row := db.QueryRowContext(ctx, query)
	if err := row.Scan(&ride.ID, &matched.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
