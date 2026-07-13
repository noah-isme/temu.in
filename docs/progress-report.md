# Laporan Progres Temu.in

Tanggal pembaruan: 13 Juli 2026

Baseline terverifikasi: commit `205b40d75d1d20baa0afd1973e15476ec8c5c18e`

## Ringkasan eksekutif

Temu.in sudah melewati tahap scaffold dan kini dapat dijalankan sebagai stack end-to-end untuk pengembangan. Fondasi MVP—authentication, katalog provider/service, jadwal, booking, frontend dasar, database, cache service, container, dan reverse proxy—sudah tersedia. Sistem belum production-ready karena payment dan email masih berupa implementasi internal, coverage backend masih terbatas, dan operasional produksi belum lengkap.

Estimasi kesiapan berdasarkan scope spesifikasi awal:

| Sasaran | Estimasi | Keterangan |
|---|---:|---|
| Fondasi MVP pengembangan | 70% | Stack berjalan, alur inti tersedia, QA dasar aktif |
| Spesifikasi produk penuh | 50% | Payment, notification worker, dashboard lengkap, dan E2E belum selesai |
| Kesiapan produksi | 30% | Security audit, observability, deployment, backup, dan runbook belum lengkap |

Persentase adalah estimasi engineering berdasarkan fitur dan risiko, bukan hasil coverage otomatis.

## Progres terverifikasi

### Infrastruktur dan database

- [x] Monorepo backend/frontend dan environment template.
- [x] Docker image backend Go 1.24 dan frontend Nginx.
- [x] Docker Compose untuk PostgreSQL 16, Redis 7, backend, frontend, dan Nginx.
- [x] Health check PostgreSQL/Redis serta dependency ordering.
- [x] Reverse proxy mempertahankan prefix `/api` dan SPA fallback bekerja.
- [x] Migrasi berjalan pada PostgreSQL 16 nyata.
- [x] Relasi ID konsisten: user integer; provider dan service UUID.
- [x] Exclusion constraint PostgreSQL menolak jadwal booking provider yang overlap.

### Backend

- [x] Register, login, refresh rotation, logout, forgot/reset password, dan email verification flow.
- [x] Provider dan service listing/search/CRUD dengan ownership check serta admin override.
- [x] Availability schedule dan exception CRUD.
- [x] Kalkulasi slot 30 menit dengan exception parsial dan full-day.
- [x] Booking create/list/detail/cancel/confirm/complete/reschedule.
- [x] Favorites, reviews, payment intent internal, admin promote/audit, rate limit, dan health endpoint.
- [ ] Redis cache dan invalidasi slot.
- [ ] Midtrans Snap/webhook nyata.
- [ ] SendGrid dan worker notifikasi asynchronous.
- [ ] Swagger/OpenAPI.

### Frontend

- [x] Routing, responsive navigation, protected admin route, dan auth state.
- [x] Provider/service list, booking flow, payment-result feedback, dan admin promote flow.
- [x] Axios authentication/refresh interceptor.
- [x] MSW untuk test dan development opsional.
- [ ] Register/reset/verification UI lengkap.
- [ ] Booking history/detail dan provider dashboard lengkap.
- [ ] Midtrans hosted payment UI.
- [ ] Calendar penuh, analytics chart, export, dan notification center.

### QA dan dokumentasi

- [x] `go vet ./...` berhasil.
- [x] `go test -race ./...` berhasil.
- [x] Unit/regression test availability, provider authorization, dan registrasi route tersedia.
- [x] Frontend lint berhasil.
- [x] Frontend 4 file test/5 test berhasil.
- [x] Frontend production build berhasil.
- [x] GitHub Actions menjalankan backend vet/race test/migration smoke test dan frontend lint/test/build.
- [x] Docker smoke test berhasil untuk health, provider API, dan direct SPA route.
- [ ] Target coverage minimal 60% untuk business logic kritis.
- [ ] API integration test dan browser E2E.

## Risiko dan utang teknis

| Prioritas | Risiko | Dampak |
|---|---|---|
| P0 | Payment masih intent/confirm internal tanpa Midtrans dan signature webhook | Tidak boleh digunakan untuk transaksi nyata |
| P0 | Email memakai console provider tanpa durable queue | Notifikasi tidak terkirim dan hilang saat proses gagal |
| P1 | Coverage backend masih berfokus pada tiga area | Regression business logic belum tertangkap luas |
| P1 | Dependency frontend memiliki 23 advisory pada audit terakhir | Perlu triase exploitability dan upgrade terkontrol |
| P1 | Readiness endpoint belum mewakili seluruh dependency secara mendalam | Orchestrator dapat menganggap instance siap terlalu cepat |
| P2 | Slot belum dicache dan belum memiliki invalidation strategy | Beban database meningkat saat traffic booking bertambah |
| P2 | Belum ada metrics, tracing, error tracking, backup SOP, atau incident runbook | Diagnosis dan pemulihan produksi lambat |

## Rekomendasi pengembangan berikutnya

### 1. Payment production path — P0

1. Integrasikan Midtrans Snap melalui interface provider agar sandbox dan mock dapat dipertukarkan.
2. Simpan order ID unik dan tambahkan idempotency key pada create payment.
3. Verifikasi signature webhook, petakan seluruh status transaksi, dan simpan audit callback.
4. Tambahkan integration test untuk success, pending, expired, deny, refund, duplicate callback, dan invalid signature.
5. Hubungkan frontend ke hosted Snap flow; jangan pernah menyimpan data kartu.

Kriteria selesai: callback terverifikasi, idempotent, dapat direplay dengan aman, dan booking/payment tetap konsisten.

### 2. Notification delivery — P0

1. Implementasikan SendGrid provider di belakang interface yang sudah tersedia.
2. Buat tabel/outbox job dan worker dengan retry exponential serta dead-letter state.
3. Tambahkan template verification, reset password, booking confirmation/cancellation/reminder, dan receipt.
4. Hindari goroutine fire-and-forget untuk event penting.

Kriteria selesai: event persisten, retry terukur, kegagalan dapat diaudit, dan provider dapat dimock pada test.

### 3. QA dan keamanan — P1

1. Tambahkan test auth, booking transaction/conflict, payment callback, refresh rotation, dan authorization per role.
2. Tambahkan E2E auth → provider → slot → booking → payment sandbox.
3. Ukur coverage dan capai minimal 60% pada package kritis.
4. Triase `npm audit`; upgrade dependency per kelompok dan jalankan regression suite setelah setiap kelompok.
5. Tambahkan secret scanning, dependency scanning, dan static analysis ke CI.

### 4. API contract dan operasional — P1

1. Publikasikan OpenAPI untuk prefix canonical `/api` dan generate contract test.
2. Ubah readiness agar memeriksa PostgreSQL dan Redis dengan timeout pendek.
3. Tambahkan structured logging, request ID, metrics HTTP/DB/cache/worker, dan error tracking.
4. Dokumentasikan deployment, rollback, backup/restore, rotasi secret, serta incident response.

### 5. Performance dan product completion — P2

1. Cache slot berdasarkan provider/tanggal/service dengan TTL pendek dan invalidasi pada perubahan schedule, exception, atau booking.
2. Lengkapi provider dashboard, booking history, reschedule UI, analytics, pagination server-side, dan export.
3. Tambahkan calendar UI, accessibility audit, responsive regression, dan performance budget.

## Urutan sprint yang disarankan

| Sprint | Fokus | Exit criteria |
|---|---|---|
| 1 | Midtrans, idempotency, webhook security | Payment sandbox end-to-end dan integration test lulus |
| 2 | SendGrid, outbox worker, template | Notifikasi durable dengan retry dan audit |
| 3 | Coverage, E2E, OpenAPI, dependency remediation | Coverage kritis ≥60%, contract/E2E hijau |
| 4 | Observability, readiness, deployment/runbook | Staging dapat dipantau, di-rollback, dan direstore |
| 5 | Cache dan penyelesaian dashboard/UI | Target performa serta alur pengguna utama tercapai |

## Perintah verifikasi

```bash
cd backend
go vet ./...
go test -race ./...

cd ../frontend
npm run lint
npm test -- --run
npm run build

cd ..
docker compose config --quiet
docker compose up -d --build
curl -fsS http://localhost/api/health/live
curl -fsS http://localhost/api/providers
curl -fsS http://localhost:5173/booking
```

Hasil verifikasi harus dicatat ulang pada laporan ini setiap kali risiko P0/P1 ditutup atau scope sprint berubah.
