# Frontend Temu.in

SPA React 18 + TypeScript yang dibangun dengan Vite, React Router, TanStack Query, Zustand, Tailwind CSS, Axios, dan Vitest/MSW.

## Menjalankan frontend

```bash
cp .env.example .env
npm ci
npm run dev
```

Template environment menggunakan backend nyata secara default:

```env
VITE_API_URL=http://localhost:8080/api
VITE_USE_MSW=false
```

Aktifkan MSW bila ingin mengembangkan UI tanpa backend:

```bash
VITE_USE_MSW=true npm run dev
```

MSW selalu digunakan oleh test melalui `.env.test` dan `vitest.setup.ts`.

## Route aplikasi

- `/` — beranda;
- `/providers` — daftar provider;
- `/services` — daftar layanan;
- `/booking` — alur pemilihan provider, service, tanggal, slot berdurasi service, booking, dan payment intent Midtrans;
- `/login` — login;
- `/admin` — user management dan audit untuk user terautentikasi.

## QA

```bash
npm run lint
npm test -- --run
npm run build
```

Per 9 Agustus 2026, lint, 4 file test/5 test, dan production build berhasil. Warning data browser yang usang tidak menggagalkan build. Audit dependency terakhir melaporkan 23 advisory (2 low, 6 moderate, 14 high, 1 critical); lakukan triase sebelum upgrade dan hindari `npm audit fix --force` tanpa regression test.

## Container

Image production membangun aset statis lalu menyajikannya melalui Nginx pada port 80. Konfigurasi `frontend/nginx.conf` menyediakan SPA fallback dan meneruskan `/api/*` ke service backend pada jaringan Docker Compose.
