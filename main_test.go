package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCacheTTL memverifikasi mekanisme kedaluwarsa in-memory cache berdasarkan durasi TTL.
func TestCacheTTL(t *testing.T) {
	ttl := 50 * time.Millisecond
	cache := NewCache(ttl)

	cache.Set("test_key", "test_value")

	val, ok := cache.Get("test_key")
	if !ok || val != "test_value" {
		t.Fatalf("Ekspektasi nilai 'test_value', didapat: %v", val)
	}

	time.Sleep(60 * time.Millisecond)

	_, ok = cache.Get("test_key")
	if ok {
		t.Fatal("Ekspektasi cache kedaluwarsa (miss), tetapi nilai masih ditemukan")
	}
}

// BenchmarkCacheHit mengukur performa handler saat data tersedia di dalam in-memory cache.
func BenchmarkCacheHit(b *testing.B) {
	cache := NewCache(30 * time.Second)
	router := setupRouter(cache, 4)

	// Inisialisasi awal agar request pertama memuat data ke dalam cache
	initReq := httptest.NewRequest("GET", "/items/1", nil)
	initW := httptest.NewRecorder()
	router.ServeHTTP(initW, initReq)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/items/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Cache") != "HIT" {
			b.Fatalf("Ekspektasi header X-Cache: HIT, didapat: %s", w.Header().Get("X-Cache"))
		}
	}
}

// BenchmarkCacheMiss mengukur performa handler saat data belum ada di cache (membutuhkan simulasi I/O backend).
func BenchmarkCacheMiss(b *testing.B) {
	cache := NewCache(30 * time.Second)
	router := setupRouter(cache, 4)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Menggunakan ID unik pada setiap iterasi untuk menjamin terjadinya cache miss
		req := httptest.NewRequest("GET", fmt.Sprintf("/items/%d", i+1000000), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkWorkerPool mengukur throughput eksekusi tugas konkuren pada worker pool.
func BenchmarkWorkerPool(b *testing.B) {
	ctx := context.Background()
	jobs := []Job{
		{ID: 1},
		{ID: 2},
		{ID: 3},
		{ID: 4},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pool := NewPool(4, len(jobs))
		results := pool.Process(ctx, jobs)
		if len(results) != len(jobs) {
			b.Fatalf("Ekspektasi %d hasil, didapat %d", len(jobs), len(results))
		}
	}
}
