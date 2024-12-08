package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

func main() {
	mux := setup()
	slog.Info("Listening on :8080")

	ticker := time.NewTicker(2500 * time.Microsecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := queue.insert(context.Background()); err != nil {
					slog.Error("failed to insert queue", err)
				}
			}
		}
	}()

	http.ListenAndServe(":8080", mux)
}

func setup() http.Handler {
	host := os.Getenv("ISUCON_DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("ISUCON_DB_PORT")
	if port == "" {
		port = "3306"
	}
	_, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to convert DB port number from ISUCON_DB_PORT environment variable into int: %v", err))
	}
	user := os.Getenv("ISUCON_DB_USER")
	if user == "" {
		user = "isucon"
	}
	password := os.Getenv("ISUCON_DB_PASSWORD")
	if password == "" {
		password = "isucon"
	}
	dbname := os.Getenv("ISUCON_DB_NAME")
	if dbname == "" {
		dbname = "isuride"
	}

	dbConfig := mysql.NewConfig()
	dbConfig.User = user
	dbConfig.Passwd = password
	dbConfig.Addr = net.JoinHostPort(host, port)
	dbConfig.Net = "tcp"
	dbConfig.DBName = dbname
	dbConfig.ParseTime = true
	dbConfig.InterpolateParams = true
	_db, err := sqlx.Connect("mysql", dbConfig.FormatDSN())
	if err != nil {
		panic(err)
	}
	db = _db

	mux := chi.NewRouter()
	// mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.HandleFunc("POST /api/initialize", postInitialize)

	// app handlers
	{
		mux.HandleFunc("POST /api/app/users", appPostUsers)

		authedMux := mux.With(appAuthMiddleware)
		authedMux.HandleFunc("POST /api/app/payment-methods", appPostPaymentMethods)
		authedMux.HandleFunc("GET /api/app/rides", appGetRides)
		authedMux.HandleFunc("POST /api/app/rides", appPostRides)
		authedMux.HandleFunc("POST /api/app/rides/estimated-fare", appPostRidesEstimatedFare)
		authedMux.HandleFunc("POST /api/app/rides/{ride_id}/evaluation", appPostRideEvaluatation)
		authedMux.HandleFunc("GET /api/app/notification", appGetNotification)
		authedMux.HandleFunc("GET /api/app/nearby-chairs", appGetNearbyChairs)
	}

	// owner handlers
	{
		mux.HandleFunc("POST /api/owner/owners", ownerPostOwners)

		authedMux := mux.With(ownerAuthMiddleware)
		authedMux.HandleFunc("GET /api/owner/sales", ownerGetSales)
		authedMux.HandleFunc("GET /api/owner/chairs", ownerGetChairs)
	}

	// chair handlers
	{
		mux.HandleFunc("POST /api/chair/chairs", chairPostChairs)

		authedMux := mux.With(chairAuthMiddleware)
		authedMux.HandleFunc("POST /api/chair/activity", chairPostActivity)
		authedMux.HandleFunc("POST /api/chair/coordinate", chairPostCoordinate)
		authedMux.HandleFunc("GET /api/chair/notification", chairGetNotification)
		authedMux.HandleFunc("POST /api/chair/rides/{ride_id}/status", chairPostRideStatus)
	}

	// internal handlers
	{
		mux.HandleFunc("GET /api/internal/matching", internalGetMatching)
	}

	return mux
}

type postInitializeRequest struct {
	PaymentServer string `json:"payment_server"`
}

type postInitializeResponse struct {
	Language string `json:"language"`
}

func postInitialize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &postInitializeRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if out, err := exec.Command("../sql/init.sh").CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to initialize: %s: %w", string(out), err))
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE settings SET value = ? WHERE name = 'payment_gateway_url'", req.PaymentServer); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, postInitializeResponse{Language: "go"})
}

type Coordinate struct {
	Latitude  int `json:"latitude"`
	Longitude int `json:"longitude"`
}

func bindJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	buf, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(buf)
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(statusCode)
	buf, marshalError := json.Marshal(map[string]string{"message": err.Error()})
	if marshalError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"marshaling error failed"}`))
		return
	}
	w.Write(buf)

	slog.Error("error response wrote", err)
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}

// chair_locations をキューイングする構造体
type chairLocationQueue struct {
	mu    sync.Mutex
	queue []*QueueData
}

type QueueData struct {
	chair *Chair
	req   *Coordinate
}

func (q *chairLocationQueue) push(d *QueueData) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = append(q.queue, d)
}

func (q *chairLocationQueue) insert(ctx context.Context) error {
	q.mu.Lock()

	if len(q.queue) == 0 {
		q.mu.Unlock()
		return nil
	}

	data := []*QueueData{}
	copy(data, q.queue)

	clear(q.queue)

	q.mu.Unlock()

	tx, err := db.Beginx()
	if err != nil {
		return errors.New("failed to begin transaction: " + err.Error())
	}
	defer tx.Rollback()

	for _, d := range data {
		var err1 error
		beforeChairLocation := &ChairLocation{}
		if err1 = tx.GetContext(
			ctx,
			beforeChairLocation,
			`SELECT * FROM chair_locations WHERE chair_id = ? ORDER BY created_at DESC LIMIT 1`,
			d.chair.ID,
		); err1 != nil && !errors.Is(err1, sql.ErrNoRows) {
			return errors.New("failed to get beforeChairLocation: " + err1.Error())
		}

		if !errors.Is(err1, sql.ErrNoRows) {
			var err error
			distance := &Distance{}
			if err = tx.GetContext(
				ctx,
				distance,
				`SELECT chair_id, total_distance, total_distance_updated_at FROM chair_distance WHERE chair_id = ?`,
				d.chair.ID,
			); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			newDist := abs(beforeChairLocation.Latitude-d.req.Latitude) + abs(beforeChairLocation.Longitude-d.req.Longitude)
			if errors.Is(err, sql.ErrNoRows) {
				if _, err := tx.ExecContext(
					ctx,
					`INSERT INTO chair_distance (chair_id, total_distance, total_distance_updated_at) VALUES (?, ?, CURRENT_TIMESTAMP(6))`,
					d.chair.ID, newDist,
				); err != nil {
					return err
				}
			} else {
				if _, err := tx.ExecContext(
					ctx,
					`UPDATE chair_distance SET total_distance = ?, total_distance_updated_at = CURRENT_TIMESTAMP(6) WHERE chair_id = ?`,
					int64(distance.TotalDistance+newDist), d.chair.ID,
				); err != nil {
					return err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.New("failed to commit transaction: " + err.Error())
	}

	return nil
}
