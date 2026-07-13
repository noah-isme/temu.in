# Checklist Implementasi & Rencana Sprint

## Status aktual — 13 Juli 2026

Detail bukti QA, risiko, estimasi kesiapan, dan urutan implementasi terdapat di [progress-report.md](progress-report.md).

- [x] Monorepo backend/frontend, environment template, Dockerfile, Compose, dan Nginx.
- [x] PostgreSQL migrations tervalidasi pada PostgreSQL 16 serta constraint anti-overlap booking.
- [x] Authentication JWT/refresh token, provider, service, availability CRUD, booking, favorites, reviews, dan admin user management dasar.
- [x] Frontend routing, authentication, provider/service list, booking flow, dan admin management dasar.
- [x] Pipeline lokal lint → test → build dan workflow CI dengan migration smoke test.
- [ ] Integrasi Midtrans serta signature verification dan idempotency webhook.
- [ ] Integrasi SendGrid, template email, retry queue, reminder, dan receipt.
- [ ] Redis slot cache/invalidation dan pengujian concurrency end-to-end.
- [ ] Swagger/OpenAPI, E2E test, observability, deployment runbook, dan production hardening.

Catatan: checklist lama di bawah dipertahankan sebagai scope awal. Status aktual di atas mengikuti implementasi yang telah diverifikasi.

## Checklist Implementasi
- [ ] Inisialisasi repositori monorepo atau dua repo terpisah untuk backend dan frontend sesuai struktur yang ditentukan.
- [ ] Siapkan konfigurasi lingkungan (.env), Docker Compose, dan Nginx reverse proxy.
- [ ] Rancang database PostgreSQL sesuai skema dan indeks yang diwajibkan.
- [ ] Implementasikan seluruh endpoint backend (auth, providers, services, availability, bookings, payments, admin) dengan logika bisnis yang diberikan.
- [ ] Bangun frontend React lengkap dengan modul auth, booking, provider, admin dashboard, dan komponen UI yang disyaratkan.
- [ ] Integrasikan Midtrans Snap untuk pembayaran dan SendGrid untuk email notifikasi.
- [ ] Terapkan QA pipeline: lint, test, build untuk frontend dan backend; sertakan dokumentasi Swagger dan automasi CI/CD.

### Issue 1 – Fondasi Infrastruktur & Lingkungan
Fokus: struktur repo, environment, containerization, reverse proxy, tooling lint/test/build.

:::task-stub{title="Set up repo structure, environments, and CI foundation"}
1. Buat struktur `backend/` dan `frontend/` mengikuti hierarki di dokumentasi; pastikan file konfigurasi utama tersedia (Dockerfile, docker-compose, .env.example).
2. Konfigurasikan Docker Compose untuk PostgreSQL, Redis, backend API, frontend, dan Nginx sebagai reverse proxy.
3. Siapkan template environment (.env) untuk semua layanan, termasuk kredensial Midtrans dan SendGrid.
4. Konfigurasikan lint/test/build script (Go tools, npm scripts) dan pipeline CI menjalankan lint → test → build berurutan.
:::

### Issue 2 – Backend API & Layanan Inti
Fokus: database schema, repositories, services, handlers, auth, booking logic, payments, notifications.

:::task-stub{title="Implement backend modules and business logic in Go"}
1. Definisikan model GORM untuk semua tabel (users, service_providers, services, availability, bookings, notifications) beserta migrasi.
2. Implementasi repository & service layer untuk auth, provider, booking, availability, notification, payment mengikuti struktur `internal/`.
3. Bangun handler Gin untuk endpoint auth, providers, services, availability, bookings, payments, admin sesuai spesifikasi.
4. Integrasikan Redis untuk caching slot dan distributed locking; hubungkan SendGrid & Midtrans Snap dalam service terkait.
5. Tambahkan dokumentasi Swagger dan unit/integration test untuk layanan utama.
:::

### Issue 3 – Frontend Aplikasi Pemesanan
Fokus: struktur React, state management, halaman utama, dashboard, booking flow, komponen UI.

:::task-stub{title="Develop frontend SPA with booking, provider, and admin experiences"}
1. Bootstrap Vite + React + TS project dengan struktur `src/` sesuai dokumentasi (api, components, hooks, pages, store, types, utils).
2. Implementasikan modul auth (login/register, protected routes) terhubung ke backend JWT.
3. Bangun pengalaman pelanggan: browsing provider/service, memilih slot (FullCalendar), membuat & mengelola booking.
4. Sediakan dashboard provider & admin dengan statistik, tabel booking, dan manajemen pengguna.
5. Integrasikan pembayaran (Midtrans) dan notifikasi UI untuk status booking/pembayaran.
:::

### Issue 4 – Observability, QA, dan Peluncuran
Fokus: monitoring, email templates, worker queue, build & deploy pipeline.

:::task-stub{title="Finalize QA automation, notifications, and deployment readiness"}
1. Siapkan worker queue untuk email notifikasi (booking confirmation, reminder, cancellation, reschedule, payment receipt).
2. Implementasi logging, metrics, dan error tracking minimal (middleware logger, health checks, readiness).
3. Tambahkan end-to-end tests (Postman/Newman atau Cypress) mencakup alur auth → booking → pembayaran.
4. Harden konfigurasi produksi: Nginx, SSL, scaling guideline, backup strategi DB.
5. Dokumentasikan SOP deployment dan runbook incident response.
:::

## Rencana Sprint (3 Sprint, masing-masing 2 minggu)

1. **Sprint 1 – Fondasi & Backend Minimum**  
   - Deliverable: infrastruktur dasar, schema database, auth, provider CRUD, lint/test/build pipeline.

2. **Sprint 2 – Booking Workflow & Frontend Core**  
   - Deliverable: availability + booking logic, frontend auth/provider listing/booking flow, notifikasi awal.

3. **Sprint 3 – Payments, Admin, Observability**  
   - Deliverable: Midtrans integration, admin dashboard, analytics, QA hardening, deployment checklist.

## Developer notes — local dev & MSW

- The frontend includes MSW (Mock Service Worker) to allow frontend-first development without a running backend. Toggle MSW using the Vite env variable `VITE_USE_MSW` (set to `true` to use mocks, `false` to call the real backend).
- Backend menggunakan prefix canonical `/api` untuk endpoint aplikasi (misalnya `/api/providers`). Endpoint health tersedia pada `/health/live`, `/health/ready`, `/api/health/live`, dan `/api/health/ready`. Handler data mock pada root backend sudah dihapus agar tidak bertabrakan dengan route produksi.

If you run into 404s for frontend requests during local development, either enable MSW in the frontend or ensure the backend is running and reachable at the host/port configured in `VITE_API_URL`.
