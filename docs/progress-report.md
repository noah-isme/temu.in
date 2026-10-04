# Laporan Progres Temu.in

Tanggal pembaruan: 4 Oktober 2026 (hardening item 1-7; sebelumnya 9 Agustus 2026)

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
- [x] Docker image backend Go 1.27.1 (`distroless/static-debian12:nonroot`, user `nonroot:nonroot`) dan frontend `nginxinc/nginx-unprivileged` (uid 101, port container 8080).
- [x] Docker Compose untuk PostgreSQL 16, Redis 7, backend, frontend, dan Nginx.
- [x] Health check PostgreSQL/Redis serta dependency ordering.
- [x] Readiness `/api/health/ready`: PostgreSQL kritis (503 `unavailable`), Redis non-kritis (200 `degraded`), timeout 2 detik.
- [x] Structured logging (`log/slog`, JSON di produksi), header `X-Request-ID` pada setiap respons API, dan access log.
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
- [x] Redis cache dan invalidasi slot (version token per provider, TTL 5 menit, fallback ke database bila Redis tidak tersedia; invalidasi pada schedule/exception CRUD, perubahan booking, dan kegagalan payment).
- [x] SendGrid provider, notification outbox, retry/dead-letter worker, delivery audit, booking/provider notifications, reminders, dan payment receipt.
- [x] Swagger/OpenAPI: spesifikasi OpenAPI 3.0.3 di `backend/openapi/openapi.yaml`, disajikan di `GET /api/openapi.yaml`, dengan contract test route-walk dan validasi bentuk respons (kin-openapi); lint Redocly di CI.
- [x] Respons publik provider/review tidak lagi memuat email/role pengguna (DTO publik, hanya nama).

### Frontend

- [x] Routing, responsive navigation, protected admin route, dan auth state.
- [x] Provider/service list, booking flow, payment-result feedback, dan admin promote flow.
- [x] Axios authentication/refresh interceptor.
- [x] MSW untuk test dan development opsional.
- [x] Register/reset/verification UI lengkap (register, forgot password, reset password, verify email).
- [x] Booking history/detail (termasuk cancel dan reschedule) dan provider dashboard (daftar booking provider dengan confirm/complete, pengelolaan schedule dan exception). Calendar, analytics, dan export masih terbuka (lihat item di bawah).
- [x] Tipe API frontend di-generate dari spesifikasi OpenAPI (`openapi-typescript`, `src/api/schema.ts`).
- [ ] Midtrans hosted payment UI.
- [ ] Calendar penuh, analytics chart, export, dan notification center.

### QA dan dokumentasi

- [x] `go vet ./...` berhasil.
- [x] `go test -race ./...` berhasil (unit), dan suite integrasi PostgreSQL berhasil dengan `REQUIRE_INTEGRATION=1`.
- [x] Unit/regression test availability, payment provider/webhook, notification provider/templates/worker, provider authorization, dan registrasi route tersedia.
- [x] Authorization tests untuk customer, provider owner, admin, dan akses cross-tenant.
- [x] Frontend lint berhasil.
- [x] Frontend 12 file test/43 test berhasil (Node 26 lokal; CI Node 20). Catatan: di host dengan load sangat tinggi beberapa test dapat timeout (5 s); pengulangan pada host yang sama lulus.
- [x] Frontend production build berhasil.
- [x] GitHub Actions menjalankan backend gofmt/vet/race test (unit + integrasi)/coverage gate/migration smoke test, lint OpenAPI, frontend audit/gen:api diff/lint/test/build, serta `govulncheck`, `gitleaks`, dan `trivy`.
- [x] Docker smoke test berhasil untuk health, provider API, dan direct SPA route.
- [x] Docker smoke test 2026-10-04 (container non-root): readiness 200 `ok` → 200 `degraded` (Redis dihentikan) → 503 `unavailable` (PostgreSQL dihentikan); `/api/providers`, `/api/openapi.yaml`, dan route SPA merespons melalui proxy (`:80`) dan langsung (`:8080`, `:5173`); `X-Request-ID` ada di respons API; `/api/providers` tidak memuat `email`.
- [x] Target coverage minimal 60% untuk business logic kritis (diukur 2026-10-04 dengan suite integrasi, `scripts/coverage-check.sh`): booking 67.2%, payment 68.3%, availability 92.0%, auth 81.8%; token 94.1%, user 90.2%, review 87.5%, favorite 22.5%, middleware 98.0%, admin 82.4%, audit 83.3%, service 20.3%.
- [x] API integration test (PostgreSQL nyata, database terisolasi per package: booking, payment, auth, availability, token, user, review, admin, audit, serta authz router dan validasi respons OpenAPI).
- [ ] Browser E2E.
- [ ] Sandbox test Midtrans/SendGrid dijalankan dengan kredensial nyata (test tersedia, opt-in via `MIDTRANS_SANDBOX_SERVER_KEY`/`SENDGRID_SANDBOX_API_KEY`; belum dijalankan).

### Dependency audit

Diaudit 2026-10-04 (`frontend/`, `npm audit`).

- Group 1 (step 35, non-major: `npm audit fix` + axios 1.20.0, react-router-dom 6.30.6, vite 5.4.21, eslint 9.39.5, typescript-eslint 8.71.0): total 40 → 14 (sebelum: 1 critical, 31 high, 6 moderate, 2 low; sesudah: 1 critical, 7 high, 5 moderate, 1 low). Runtime (`--omit=dev`): 6 (5 high, 1 moderate) → 2 moderate.
- Group 2 (step 36, test tooling major): vitest 2 → 4.1.11 (fix `@vitest/mocker` GHSA-82fw-gwwq-j7x9 dan vitest UI GHSA-5xrq-8626-4rwp; vitest ≥4.1 membutuhkan vite 6), vite 5 → 6.4.3 (fix GHSA-fx2h-pf6j-xcff/GHSA-4w7w-66w2-5vf9/GHSA-v6wh-96g9-6wx3 dan esbuild GHSA-67mh-4wv8-2f99 via esbuild 0.25), jsdom 24 → 29.1.1 (jsdom 30 membutuhkan Node ≥22.22, CI memakai Node 20). Total 14 → 9 (0 critical, 6 high, 2 moderate, 1 low). Runtime: 2 moderate, 0 high/critical.
- CI gate: `npm audit --omit=dev --audit-level=high` di job frontend.

Sisa advisory (tidak di-upgrade pada pass ini):

| Package | Severity | Path (`via`) | Exploitability | Keputusan | Follow-up |
|---|---|---|---|---|---|
| `react-router` / `react-router-dom` 6.30.6 | moderate | langsung (GHSA-wrjc-x8rr-h8h6 open redirect via backslash di `<Link>`/`useNavigate`; GHSA-337j-9hxr-rhxg SSR `deserializeErrors`) | Runtime, rendah: SPA tanpa SSR (advisory SSR tidak berlaku); target navigasi hanya path literal dan `state.from.pathname` dari `RequireAuth` (history state same-origin, tidak dapat diisi lintas origin) | Diterima; fix hanya di v7 (major) | Upgrade ke react-router 7 sebagai satu grup dengan regression suite |
| `braces` (via `micromatch`, `fast-glob`, `chokidar` 3) | high | `tailwindcss` 3.4.19 → `chokidar`/`fast-glob`/`micromatch` → `braces`; `msw` 1.3.5 → `chokidar` → `braces` (GHSA-vfj7-8cjw-p6xm) | Dev/build-time saja: glob pattern berasal dari config repo, bukan input pengguna; tidak ada di bundle produksi | Diterima; fix menuntut tailwindcss 4 dan msw 2+ (major) | tailwindcss 4 migration; msw 1 → 2 (`rest` → `http`) |
| `chokidar`, `micromatch`, `fast-glob` | high (transitif dari `braces`) | sama seperti di atas | Dev/build-time saja | Diterima | Sama seperti `braces` |
| `tailwindcss` 3.4.19, `msw` 1.3.5 | high (carrier) | flagged hanya karena dependency transitif di atas; tidak ada advisory yang menyebut paket ini sendiri | Dev/build-time / test-time saja | Diterima (msw tetap v1 sesuai plan) | Lihat baris `braces` |
| `cookie` <0.7.0 | low | `msw` 1.3.5 → `cookie` (GHSA-pxg6-pf52-xh8x) | Test-time saja (mock server) | Diterima | msw 2 migration |

Trivy (`fs`, default tanpa dev dependency) hanya akan melihat `react-router` (moderate) dari daftar ini.

Pemindaian keamanan 2026-10-04 (step 33/37):

- `trivy fs --scanners vuln,misconfig --severity CRITICAL,HIGH --ignore-unfixed`: 0 temuan (go.mod, package-lock.json, kedua Dockerfile). Trivy melaporkan advisory react-router sebagai CVE-2026-53666 dan CVE-2026-53669 (MEDIUM, fix di 7.18.0); keduanya dicatat di `.trivyignore` bersama alias GHSA dan GO-2026-5932.
- `gitleaks detect` (riwayat git penuh, 12 commit) dan `gitleaks dir` (working tree): 0 temuan setelah allowlist `graphify-out/cache/` di `.gitleaks.toml` (6 temuan `generic-api-key` adalah hash SHA-256 isi file pada cache graphify, bukan secret).
- `govulncheck ./...`: 0 vulnerability yang terjangkau; 1 advisory di module yang di-require tetapi tidak dipanggil (GO-2026-5932, `x/crypto/openpgp`, tanpa fix upstream).
- `npm audit --omit=dev --audit-level=high`: lulus (runtime 2 moderate, react-router).

## Risiko dan utang teknis

| Prioritas | Risiko | Dampak |
|---|---|---|
| P1 | Payment/SendGrid belum memiliki integration test sandbox dan tetap membutuhkan konfigurasi kredensial produksi | Integrasi eksternal belum terbukti end-to-end pada environment produksi |
| P2 | Coverage package kritis ≥60% dan suite integrasi berjalan di CI, tetapi `notification` (48%), `provider` (41%), `favorite`, `service`, dan `server` masih rendah; browser E2E belum ada | Regression di area tersebut dan alur UI end-to-end belum tertangkap otomatis |
| P2 | Sisa advisory frontend: 2 moderate runtime (react-router 6) dan 7 dev-only; CI kini menolak advisory runtime high/critical (`npm audit --omit=dev --audit-level=high`) dan menjalankan Trivy; lihat "Dependency audit" | Upgrade major (react-router 7, tailwindcss 4, msw 2) masih tertunda |
| P2 | Readiness kini memeriksa PostgreSQL (kritis, 503) dan Redis (non-kritis, 200 `degraded`); saat Redis turun, rate limiter dan token blacklist fail-open | Instance tetap menerima traffic tanpa rate limit/blacklist selama Redis tidak tersedia (dicatat di log Warn) |
| P3 | Slot dicache per provider dengan invalidasi version token; granularitas per provider (semua tanggal) dan slot belum mengurangi booking yang ada | Recompute lebih sering dari minimum; semantik slot tidak berubah dari sebelumnya |
| P2 | Structured logging dan request ID tersedia, tetapi belum ada metrics, tracing, error tracking, backup SOP, atau incident runbook | Diagnosis dan pemulihan produksi lambat |

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
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' REQUIRE_INTEGRATION=1 \
  go test -race -coverprofile=coverage.out -covermode=atomic ./... && ./scripts/coverage-check.sh coverage.out
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

cd ../frontend
npm run lint
npm test -- --run
npm run build
npm audit --omit=dev --audit-level=high
npm run gen:api && git diff --exit-code -- src/api/schema.ts

cd ..
npx --yes @redocly/cli@1 lint --config backend/openapi/redocly.yaml backend/openapi/openapi.yaml
docker compose config --quiet
docker compose up -d --build
curl -fsS http://localhost/api/health/live
curl -fsS http://localhost/api/providers
curl -fsS http://localhost:5173/booking
curl -i http://localhost/api/health/ready
curl -fsS http://localhost/api/openapi.yaml
docker inspect --format '{{.Config.User}}' $(docker compose ps -q backend)   # nonroot:nonroot
docker compose exec frontend id -u                                           # 101
```

Hasil verifikasi harus dicatat ulang pada laporan ini setiap kali risiko P0/P1 ditutup atau scope sprint berubah.
