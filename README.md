# Temu.in

Platform pemesanan janji temu dengan SPA React/TypeScript dan API Go/Gin. PostgreSQL menyimpan data utama, Redis menangani rate limit dan refresh-token state, sementara Nginx menjadi reverse proxy.

## Status implementasi

Fondasi aplikasi dan alur inti tersedia: authentication, provider/service CRUD, jadwal availability, booking, favorites, reviews, admin user management, Midtrans payment intent, serta durable email notifications. Stack Docker, migrasi PostgreSQL, lint, unit test, dan production build sudah dapat dijalankan.

Bagian berikut masih dalam pengembangan dan belum production-ready:

- sandbox integration test (tersedia, opt-in via `MIDTRANS_SANDBOX_SERVER_KEY`/`SENDGRID_SANDBOX_API_KEY`, belum dijalankan dengan kredensial nyata) dan konfigurasi kredensial/template produksi;
- analytics admin lengkap, browser E2E test, metrics/tracing, dan runbook deployment.

Sudah tersedia sejak hardening 2026-10-04: cache slot Redis dengan invalidasi per provider, spesifikasi OpenAPI 3.0.3 (`GET /api/openapi.yaml`, sumber: `backend/openapi/openapi.yaml`), structured logging dan `X-Request-ID`, readiness PostgreSQL/Redis, container non-root, serta job CI `govulncheck`, `gitleaks`, dan `trivy`.

Laporan progres, bukti verifikasi, risiko, dan urutan pengembangan berikutnya tersedia di [docs/progress-report.md](docs/progress-report.md). Spesifikasi awal tetap tersedia di [docs/AGENT.md](docs/AGENT.md), sedangkan status sprint berada di [docs/project-checklist-and-sprints.md](docs/project-checklist-and-sprints.md).

## Struktur

```text
backend/             API Go, migrasi, dan business logic
frontend/            SPA React, test Vitest, dan Nginx image
docs/                laporan progres, spesifikasi, dan rencana sprint
nginx/               reverse proxy utama
docker-compose.yml   PostgreSQL, Redis, API, SPA, dan proxy
```

## Menjalankan dengan Docker

Salin environment template bila ingin mengubah nilai default:

```bash
cp .env.example .env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
```

Bangun dan jalankan stack:

```bash
docker compose up -d --build
```

Endpoint lokal:

- aplikasi melalui reverse proxy: <http://localhost>
- frontend langsung: <http://localhost:5173> (container `nginx-unprivileged` mendengarkan port 8080)
- backend langsung: <http://localhost:8080>
- liveness: <http://localhost/api/health/live>
- readiness: <http://localhost/api/health/ready>

Readiness (`/health/ready` dan `/api/health/ready`) mem-ping dependensi dengan batas waktu 2 detik dan mengembalikan
`{"status":"ok"|"degraded"|"unavailable","checked_at":...,"checks":{"postgres":{"status":"ok","latency_ms":N,"critical":true},"redis":{...,"critical":false}}}`.
Postgres bersifat kritis: jika gagal, respons **503** dengan `status:"unavailable"`. Redis non-kritis (rate limiter,
token blacklist, dan slot cache tetap berjalan tanpa Redis): jika gagal, respons **200** dengan `status:"degraded"`.
Body hanya memuat status generik per dependensi; detail error dicatat di log backend (level Warn). Path health
dikecualikan dari rate limiter dan access log.

Seluruh API aplikasi menggunakan prefix `/api`, bukan `/api/v1`.

## Kontrak booking, payment, dan otorisasi

`POST /api/bookings` memerlukan JWT dan body berikut:

```json
{
  "providerId": "provider-uuid",
  "serviceId": "service-uuid",
  "customerName": "Nama pelanggan",
  "slot": {
    "date": "2026-08-10",
    "start_time": "10:00",
    "end_time": "10:45"
  }
}
```

Backend memvalidasi bahwa service aktif dan dimiliki provider yang dipilih, memastikan durasi slot sama dengan durasi service, lalu mengambil harga dan currency dari katalog server. Klien tidak dapat menetapkan harga sendiri. Endpoint availability publik juga memerlukan `serviceId` agar slot dihitung memakai durasi service, bukan interval tetap 30 menit.

`POST /api/payments/create-intent` dan `/api/payments/confirm` hanya dapat dipanggil user terautentikasi. Keduanya menggunakan provider Midtrans melalui interface provider; `POST /api/payments/webhook` adalah endpoint publik yang memverifikasi signature SHA-512, mencocokkan nominal, mengabaikan transisi status yang mundur, dan mencatat digest webhook untuk idempotency. Booking baru menjadi confirmed setelah payment succeeded, dan `payment_status` diperbarui bersama status booking.

Customer hanya dapat melihat atau membatalkan booking miliknya. Provider atau admin diperlukan untuk mengelola provider, service, schedule, exception, statistik, serta confirm/complete booking; semua tindakan provider di-scope dengan provider ID.

## Konfigurasi integrasi eksternal

Salin `backend/.env.example` lalu isi `MIDTRANS_SERVER_KEY`, `MIDTRANS_CLIENT_KEY`, dan `MIDTRANS_BASE_URL` (sandbox secara default). Daftarkan `POST /api/payments/webhook` pada dashboard Midtrans.

Notifikasi penting masuk ke tabel `notifications` melalui outbox setelah transaksi bisnis, lalu diproses worker dengan SendGrid. Isi `SENDGRID_API_KEY` dan `SENDGRID_FROM_EMAIL`; template ID bersifat opsional (`SENDGRID_BOOKING_TEMPLATE_ID`, `SENDGRID_CANCELLATION_TEMPLATE_ID`, `SENDGRID_REMINDER_TEMPLATE_ID`, `SENDGRID_RECEIPT_TEMPLATE_ID`, dan `SENDGRID_AUTH_TEMPLATE_ID`). Kegagalan akan dicoba ulang dengan backoff, dipindahkan ke `dead_letter` setelah batas percobaan, dan dicatat di `notification_deliveries`.

## Pengembangan lokal

Backend membutuhkan PostgreSQL dan Redis yang dapat dijangkau oleh nilai `DATABASE_URL` dan `REDIS_URL`.

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

Frontend menggunakan backend nyata secara default dari template environment. Untuk development berbasis mock, ubah `VITE_USE_MSW=true`.

```bash
cd frontend
cp .env.example .env
npm ci
npm run dev
```

## Migrasi

API menjalankan migrasi otomatis saat startup. Migrasi juga dapat dijalankan terpisah:

```bash
cd backend
go run ./cmd/migrate
```

Migrasi mencakup constraint database yang menolak booking provider dengan waktu tumpang tindih.

## QA

```bash
cd backend
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
# integration + coverage gate (role must be able to CREATE DATABASE)
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' REQUIRE_INTEGRATION=1 \
  go test -race -coverprofile=coverage.out -covermode=atomic ./... && ./scripts/coverage-check.sh coverage.out

cd ../frontend
npm run lint
npm test -- --run
npm run build
```

Workflow yang sama dijalankan oleh GitHub Actions, termasuk smoke test migrasi pada PostgreSQL 16, lint OpenAPI (Redocly), pemeriksaan `src/api/schema.ts` hasil generate, `npm audit --omit=dev --audit-level=high`, serta job keamanan `govulncheck`, `gitleaks` (`.gitleaks.toml`), dan `trivy` (`.trivyignore`).
