import http from 'k6/http';
import { check, sleep } from 'k6';

// Mengizinkan konfigurasi alamat server melalui environment variable BASE_URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Tahap pemanasan menuju 50 VU
    { duration: '30s', target: 150 }, // Peningkatan beban menuju 150 VU
    { duration: '40s', target: 250 }, // Beban puncak pada 250 VU
    { duration: '20s', target: 0 },   // Penurunan beban kembali ke 0 VU
  ],
  thresholds: {
    // 95 persen permintaan harus selesai di bawah 1 detik (1000ms)
    http_req_duration: ['p(95)<1000'],
    // Tingkat kegagalan permintaan (error rate) harus di bawah 1 persen (0.01)
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  // Skenario 1: GET /items/{id} dengan ID acak 1-20 (kombinasi cache hit dan cache miss)
  const randomId = Math.floor(Math.random() * 20) + 1;
  const resItem = http.get(`${BASE_URL}/items/${randomId}`);
  check(resItem, {
    'GET /items/{id} status 200': (r) => r.status === 200,
    'GET /items/{id} memiliki header X-Cache': (r) => r.headers['X-Cache'] !== undefined,
  });

  // Skenario 2 (probabilitas 30%): GET /items via worker pool
  if (Math.random() < 0.3) {
    const resList = http.get(`${BASE_URL}/items?limit=5`);
    check(resList, {
      'GET /items status 200': (r) => r.status === 200,
    });
  }

  // Skenario 3 (probabilitas 10%): POST /jobs untuk distribusi komputasi
  if (Math.random() < 0.1) {
    const payload = JSON.stringify({ jobs: 5 });
    const params = {
      headers: { 'Content-Type': 'application/json' },
    };
    const resJobs = http.post(`${BASE_URL}/jobs`, payload, params);
    check(resJobs, {
      'POST /jobs status 200': (r) => r.status === 200,
    });
  }

  // Jeda singkat antar iterasi untuk simulasi perilaku pengguna nyata
  sleep(0.05);
}
