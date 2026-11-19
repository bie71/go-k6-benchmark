# Go High-Concurrency and k6 Benchmark

Proyek demonstrasi backend Go dengan arsitektur high-concurrency, mencakup in-memory caching thread-safe, worker pool berbasis goroutine dan channel, serta pengujian beban (load testing) menggunakan k6. Proyek ini dibangun untuk mendemonstrasikan keandalan penanganan 250+ Virtual Users (VU) dengan latensi respons API di bawah 1 detik tanpa menggunakan framework atau dependensi eksternal.

## Arsitektur Singkat

- Standar Library Go Murni: Menggunakan `net/http` ServeMux dengan pattern routing Go 1.22+ (`GET /items/{id}`).
- In-Memory Caching: Menggunakan `sync.RWMutex` dengan evaluasi kedaluwarsa berbasis TTL saat akses (lazy eviction).
- Worker Pool: Menggunakan goroutine, channel, dan context cancellation untuk mendistribusikan beban kerja secara efisien.

## Prasyarat

- Go 1.25 atau lebih baru: https://go.dev/dl/
- k6: https://k6.io/docs/get-started/installation/

## Menjalankan Server

Jalankan perintah berikut di dalam direktori proyek:

```bash
go run .
```

Server akan aktif dan mendengarkan permintaan di port `:8080`.

Daftar endpoint:
- `GET /health` : Pemeriksaan status kesehatan server (`{"status":"ok"}`).
- `GET /items/{id}` : Mengambil data item dari cache (header `X-Cache: HIT`) atau simulasi backend 50ms saat miss (header `X-Cache: MISS`).
- `GET /items?limit=10` : Mengambil daftar item yang diproses melalui worker pool (maksimal 100).
- `POST /jobs` : Menerima payload `{"jobs": N}` untuk dieksekusi secara konkuren oleh worker pool.

## Menjalankan Pengujian Beban (k6)

Jalankan skenario beban penuh (ramping bertahap hingga 250 VU selama 2 menit):

```bash
k6 run k6/load.js
```

Untuk smoke test singkat (misalnya 5 VU selama 10 detik):

```bash
k6 run --vus 5 --duration 10s k6/load.js
```

Jika server berjalan di alamat selain default, gunakan environment variable:

```bash
BASE_URL=http://localhost:8080 k6 run k6/load.js
```

### Cara Membaca Output Metrik k6

- `http_req_duration p(95)`: Waktu respons di mana 95% dari seluruh permintaan selesai di bawah nilai tersebut (target: < 1000ms).
- `http_req_failed`: Persentase permintaan yang gagal atau menghasilkan status error (target: < 1.00%).
- `vus`: Jumlah Virtual Users yang sedang aktif mengeksekusi skenario secara simultan.
- `http_reqs`: Total permintaan HTTP yang diproses beserta rata-rata throughput (req/s).

## Menjalankan Benchmark Internal Go

Gunakan Go testing tool untuk mengukur performa komponen internal:

```bash
go test -bench=. -benchmem ./...
```

Perintah di atas akan menjalankan:
- `BenchmarkCacheHit`: Mengukur alokasi memori dan kecepatan pembacaan data cache.
- `BenchmarkCacheMiss`: Mengukur penanganan saat data belum tersedia di cache.
- `BenchmarkWorkerPool`: Mengukur throughput distribusi tugas pada worker pool.

## Tabel Hasil Pengujian

Berikut template tabel ringkasan benchmark untuk dicatat pada portofolio atau CV:

| Skenario | VU | p95 (ms) | Throughput (req/s) | Error rate |
| :--- | :---: | :---: | :---: | :---: |
| Smoke Test (10s, laptop i5-1135G7) | 5 | 50,67 | 97,45 | 0,00% |
| Peak Load Test (1 menit, laptop i5-1135G7) | 250 | 50,12 | 5.047 | 0,00% |

Referensi peak test terukur (250 VU, 1 menit, k6 v1.x di i5-1135G7):
- p(95) = 50,12 ms, throughput 5.047 req/s, error rate 0,00%.
- Total 303.588 request berhasil diproses dalam 60 detik.
- Hasil benchmark internal Go (`go test -bench=. -benchmem`): CacheHit ≈ 4,4 µs/op, CacheMiss ≈ 50 ms/op.

### Hasil Aktual

Untuk menyimpan hasil metrik pengujian ke dalam file ringkasan terstruktur:

```bash
k6 run --summary-export=hasil.json k6/load.js
```

Catatan: Angka performa aktual sangat dipengaruhi oleh spesifikasi perangkat keras (CPU, RAM, sistem operasi) dan lingkungan jaringan lokal saat pengujian dijalankan. Jangan mencantumkan atau mengklaim angka performa yang belum diukur secara langsung pada mesin target.

## Inisialisasi Git dan Push ke GitHub

Untuk mempublikasikan proyek ini ke repositori Git:

```bash
git init
git add .
git commit -m "feat: inisialisasi go-k6-benchmark backend dan load testing"
git branch -M main
git remote add origin <URL_REPOSITORI_ANDA>
git push -u origin main
```
