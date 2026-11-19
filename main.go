package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"syscall"
	"time"
)

// writeJSON adalah fungsi pembantu untuk menulis respons JSON.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// jsonResponseMiddleware memastikan respons error HTTP seperti 405 diformat sebagai JSON.
func jsonResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		interceptor := &statusInterceptor{ResponseWriter: w}
		next.ServeHTTP(interceptor, r)
	})
}

type statusInterceptor struct {
	http.ResponseWriter
	status int
}

func (i *statusInterceptor) WriteHeader(status int) {
	i.status = status
	if status == http.StatusMethodNotAllowed {
		i.Header().Set("Content-Type", "application/json")
		i.ResponseWriter.WriteHeader(status)
		_ = json.NewEncoder(i.ResponseWriter).Encode(map[string]string{
			"error": "metode HTTP tidak diizinkan",
		})
		return
	}
	i.ResponseWriter.WriteHeader(status)
}

func (i *statusInterceptor) Write(b []byte) (int, error) {
	if i.status == http.StatusMethodNotAllowed {
		return len(b), nil
	}
	return i.ResponseWriter.Write(b)
}

// handleHealth menangani endpoint GET /health.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// handleGetItemByID menangani endpoint GET /items/{id} dengan in-memory cache dan simulasi miss.
func handleGetItemByID(cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "parameter id harus berupa bilangan bulat",
			})
			return
		}

		cacheKey := fmt.Sprintf("item:%d", id)
		if cachedItem, ok := cache.Get(cacheKey); ok {
			w.Header().Set("X-Cache", "HIT")
			writeJSON(w, http.StatusOK, map[string]any{
				"id":     id,
				"name":   fmt.Sprintf("Item-%d", id),
				"cached": true,
			})
			_ = cachedItem
			return
		}

		// Simulasi latensi pemrosesan backend sebesar 50ms saat terjadi cache miss
		select {
		case <-r.Context().Done():
			return
		case <-time.After(50 * time.Millisecond):
		}

		itemData := map[string]any{
			"id":   id,
			"name": fmt.Sprintf("Item-%d", id),
		}
		cache.Set(cacheKey, itemData)

		w.Header().Set("X-Cache", "MISS")
		writeJSON(w, http.StatusOK, map[string]any{
			"id":     id,
			"name":   fmt.Sprintf("Item-%d", id),
			"cached": false,
		})
	}
}

// handleGetItems menangani endpoint GET /items dengan batasan query limit via worker pool.
func handleGetItems(workers int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil || parsed <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "parameter limit harus berupa bilangan bulat positif",
				})
				return
			}
			if parsed > 100 {
				limit = 100
			} else {
				limit = parsed
			}
		}

		jobs := make([]Job, limit)
		for i := 0; i < limit; i++ {
			jobs[i] = Job{ID: i + 1}
		}

		pool := NewPool(workers, limit)
		results := pool.Process(r.Context(), jobs)

		items := make([]map[string]any, len(results))
		for i, res := range results {
			items[i] = map[string]any{
				"id":   res.JobID,
				"name": fmt.Sprintf("Item-%d", res.JobID),
			}
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i]["id"].(int) < items[j]["id"].(int)
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	}
}

type postJobsRequest struct {
	Jobs int `json:"jobs"`
}

type postJobsResponse struct {
	TotalJobs       int     `json:"total_jobs"`
	SuccessCount    int     `json:"success_count"`
	TotalDurationMs int64   `json:"total_duration_ms"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
}

// handlePostJobs menangani endpoint POST /jobs untuk mendistribusikan beban ke worker pool.
func handlePostJobs(workers int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postJobsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "format payload JSON tidak valid",
			})
			return
		}

		if req.Jobs < 1 || req.Jobs > 1000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "parameter jobs harus bernilai antara 1 sampai 1000",
			})
			return
		}

		jobs := make([]Job, req.Jobs)
		for i := 0; i < req.Jobs; i++ {
			jobs[i] = Job{ID: i + 1}
		}

		start := time.Now()
		pool := NewPool(workers, req.Jobs)
		results := pool.Process(r.Context(), jobs)
		elapsed := time.Since(start)

		var successCount int
		var sumDuration time.Duration
		for _, res := range results {
			if res.Err == nil {
				successCount++
			}
			sumDuration += res.Duration
		}

		var avgDurationMs float64
		if successCount > 0 {
			avg := float64(sumDuration.Milliseconds()) / float64(successCount)
			avgDurationMs = math.Round(avg*100) / 100
		}

		writeJSON(w, http.StatusOK, postJobsResponse{
			TotalJobs:       req.Jobs,
			SuccessCount:    successCount,
			TotalDurationMs: elapsed.Milliseconds(),
			AvgDurationMs:   avgDurationMs,
		})
	}
}

// setupRouter menginisialisasi router net/http ServeMux dengan Go 1.22+ pattern routing.
func setupRouter(cache *Cache, workers int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /items/{id}", handleGetItemByID(cache))
	mux.HandleFunc("GET /items", handleGetItems(workers))
	mux.HandleFunc("POST /jobs", handlePostJobs(workers))

	return jsonResponseMiddleware(mux)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cache := NewCache(30 * time.Second)
	workers := runtime.NumCPU()

	handler := setupRouter(cache, workers)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server HTTP berjalan di port %s dengan %d worker goroutine...", port, workers)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gagal menjalankan server HTTP: %v", err)
		}
	}()

	<-stop
	log.Println("Menerima sinyal penghentian, mematikan server secara graceful...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Peringatan saat shutdown server: %v", err)
	}
	log.Println("Server berhasil dihentikan.")
}
