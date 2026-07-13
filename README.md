# Temu.in

Platform pemesanan janji temu dengan SPA React/TypeScript dan API Go/Gin. PostgreSQL menyimpan data utama, Redis menangani rate limit dan refresh-token state, sementara Nginx menjadi reverse proxy.

## Status implementasi

Fondasi aplikasi dan alur inti tersedia: authentication, provider/service CRUD, jadwal availability, booking, favorites, reviews, admin user management, serta payment intent internal. Stack Docker, migrasi PostgreSQL, lint, unit test, dan production build sudah dapat dijalankan.

Bagian berikut masih dalam pengembangan dan belum production-ready:

- Midtrans dan verifikasi webhook pembayaran;
- SendGrid, template email, dan worker notifikasi;
- cache slot Redis dan invalidasinya;
- analytics admin lengkap, Swagger/OpenAPI, E2E test, metrics, dan deployment hardening.

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
- frontend langsung: <http://localhost:5173>
- backend langsung: <http://localhost:8080>
- liveness: <http://localhost/api/health/live>
- readiness: <http://localhost/api/health/ready>

Seluruh API aplikasi menggunakan prefix `/api`, bukan `/api/v1`.

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
go test ./...
go vet ./...

cd ../frontend
npm run lint
npm test -- --run
npm run build
```

Workflow yang sama dijalankan oleh GitHub Actions, termasuk smoke test migrasi pada PostgreSQL 16.
