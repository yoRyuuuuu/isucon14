package main

import (
	"database/sql"
	"errors"
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// 最も待たせているリクエストを取得
	rides := []Ride{}
	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 10`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if len(rides) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 空いている椅子を取得
	chairs := []Chair{}
	if err := db.SelectContext(ctx, &chairs, `SELECT * FROM chairs WHERE is_active = TRUE`); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if len(chairs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 最適なマッチングを見つける
	var bestRide *Ride
	var bestChair *Chair
	minDistance := int(^uint(0) >> 1) // 最大整数値

	for _, ride := range rides {
		for _, chair := range chairs {
			chairLocation := &ChairLocation{}
			if err := db.GetContext(ctx, chairLocation, `SELECT * FROM chair_locations WHERE chair_id = ? ORDER BY created_at DESC LIMIT 1`, chair.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			distance := calculateDistance(ride.PickupLatitude, ride.PickupLongitude, chairLocation.Latitude, chairLocation.Longitude)
			if distance < minDistance {
				minDistance = distance
				bestRide = &ride
				bestChair = &chair
			}
		}
	}

	if bestRide == nil || bestChair == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// マッチングを更新
	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", bestChair.ID, bestRide.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
