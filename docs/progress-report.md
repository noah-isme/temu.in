# Laporan Progres Temu.in

Tanggal pembaruan: 9 Agustus 2026

Baseline historis laporan: commit `205b40d75d1d20baa0afd1973e15476ec8c5c18e`.
Verifikasi terbaru pada tanggal di atas mencakup perubahan payment, booking, authorization, availability, dan notification yang akan dicatat dalam commit berikutnya.

## Ringkasan eksekutif

Temu.in sudah melewati tahap scaffold dan kini dapat dijalankan sebagai stack end-to-end untuk pengembangan. Fondasi MVP—authentication, katalog provider/service, jadwal, booking, frontend dasar, database, cache service, container, dan reverse proxy—sudah tersedia. Payment sekarang menggunakan Midtrans melalui provider interface dengan webhook bertanda tangan, sedangkan email penting menggunakan durable outbox dan worker SendGrid. Sistem masih memerlukan kredensial/integration test sandbox, coverage yang lebih luas, observability, dan runbook produksi.

Estimasi kesiapan berdasarkan scope spesifikasi awal:

| Sasaran | Estimasi | Keterangan |
|---|---:|---|
| Fondasi MVP pengembangan | 80% | Stack berjalan, alur inti, payment gateway, dan outbox notifikasi tersedia |
| Spesifikasi produk penuh | 65% | Dashboard, cache, OpenAPI, dan E2E masih belum selesai |
| Kesiapan produksi | 45% | Kredensial produksi, observability, deployment, backup, dan runbook belum lengkap |

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
- [x] Availability schedule dan exception CRUD dengan policy provider/admin serta query update/delete yang ter-scope provider.
- [x] Kalkulasi slot berdasarkan durasi service dengan exception parsial dan full-day.
- [x] Booking create/list/detail/cancel/confirm/complete/reschedule.
- [x] Booking mewajibkan service aktif milik provider, mengambil durasi/harga dari server, dan memakai kontrak `slot` bertipe `{date,start_time,end_time}`.
- [x] Authorization policy terpusat untuk provider/admin, termasuk ownership booking dan cross-tenant tests.
- [x] Payment provider interface Midtrans, order ID acak, amount server-side, authenticated intent/confirm, signed webhook, state transition, dan webhook idempotency.
- [x] Favorites, reviews, admin promote/audit, rate limit, dan health endpoint.
- [ ] Redis cache dan invalidasi slot.
- [x] SendGrid provider, notification outbox, retry/dead-letter worker, delivery audit, booking/provider notifications, reminders, dan payment receipt.
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
- [x] `go test ./...` berhasil.
- [x] Unit/regression test availability, payment provider/webhook, notification provider/templates/worker, provider authorization, dan registrasi route tersedia.
- [x] Authorization tests untuk customer, provider owner, admin, dan akses cross-tenant.
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
| P1 | Payment/SendGrid belum memiliki integration test sandbox dan tetap membutuhkan konfigurasi kredensial produksi | Integrasi eksternal belum terbukti end-to-end pada environment produksi |
| P1 | Coverage backend masih berfokus pada beberapa area | Regression business logic belum tertangkap luas |
| P1 | Dependency frontend memiliki 23 advisory pada audit terakhir | Perlu triase exploitability dan upgrade terkontrol |
| P1 | Readiness endpoint belum mewakili seluruh dependency secara mendalam | Orchestrator dapat menganggap instance siap terlalu cepat |
| P2 | Slot belum dicache dan belum memiliki invalidation strategy | Beban database meningkat saat traffic booking bertambah |
| P2 | Belum ada metrics, tracing, error tracking, backup SOP, atau incident runbook | Diagnosis dan pemulihan produksi lambat |

## Rekomendasi pengembangan berikutnya

### 1. Payment production path — selesai untuk jalur aplikasi

1. Midtrans Snap diisolasi melalui provider interface; order ID dibuat memakai random bytes.
2. Amount dan currency ditentukan dari service aktif di server, bukan dari payload klien.
3. Intent dan confirm memerlukan JWT; webhook memverifikasi signature SHA-512, nominal, state transition, dan digest idempotency.
4. Payment dan booking memiliki state machine; status payment sukses memperbarui `booking.status` dan `booking.payment_status` secara konsisten.

Pekerjaan lanjutan: integration test sandbox untuk success, pending, expired, deny, refund, duplicate callback, dan invalid signature; jangan pernah menyimpan data kartu.

### 2. Notification delivery — selesai untuk outbox dan worker

1. SendGrid provider menerima dynamic template ID atau rendered HTML/text fallback.
2. Tabel `notifications` menjadi durable outbox dengan deduplication key, status processing/retry/sent/dead-letter, dan claim berbasis `FOR UPDATE SKIP LOCKED`.
3. Worker memakai exponential backoff, memulihkan claim yang stale, dan mencatat setiap attempt pada `notification_deliveries`.
4. Event booking confirmation/cancellation, reminder customer/provider, payment receipt, email verification, dan password reset masuk ke outbox tanpa goroutine fire-and-forget.

Pekerjaan lanjutan: verifikasi template SendGrid di staging, alert untuk dead-letter, dan integration test provider.

### 3. QA dan keamanan — P1

1. Tambahkan integration test auth, booking transaction/conflict, payment callback, refresh rotation, dan authorization per role.
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
| 1 | Midtrans, idempotency, webhook security | Implementasi aplikasi selesai; sandbox integration test masih tersisa |
| 2 | SendGrid, outbox worker, template | Outbox, retry, dead-letter, reminder, receipt, dan audit tersedia |
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
