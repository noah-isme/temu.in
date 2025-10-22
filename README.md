# Temu.in Monorepo

Platform pemesanan janji temu end-to-end dengan frontend React dan backend Go.

## Struktur Proyek

```
/booking-system-backend   # API Go + Gin
/booking-system-frontend  # SPA React + Vite
/docs                     # Dokumentasi proyek
/docker-compose.yml       # Orkestrasi layanan pendukung
/nginx/nginx.conf         # Reverse proxy konfigurasi
```

## Memulai

1. Salin file lingkungan contoh dan sesuaikan variabelnya.

   ```bash
   cp .env.example .env
   cp booking-system-backend/.env.example booking-system-backend/.env
   cp booking-system-frontend/.env.example booking-system-frontend/.env
   ```

2. Jalankan layanan menggunakan Docker Compose.

   ```bash
   docker compose up --build
   ```

3. Akses layanan:
   - Backend API: <http://localhost:8080>
   - Frontend web: <http://localhost:5173>
   - Reverse proxy: <http://localhost>

## Pengembangan Lokal

### Backend

```bash
cd booking-system-backend
cp .env.example .env
go run ./cmd/api
```

### Frontend

```bash
cd booking-system-frontend
cp .env.example .env
npm install
npm run dev
```

### Frontend: Menggunakan MSW (Mock Service Worker)

Jika kamu ingin mengembangkan frontend tanpa menunggu backend, MSW sudah disiapkan untuk development. Berikut langkah singkat:

1. Dari direktori `booking-system-frontend`, pasang dependensi dan jalankan dev server:

```bash
cd booking-system-frontend
npm install
npm run dev

Note: The frontend now uses TanStack React Query and React Toastify for improved UX. After pulling changes, install new dependencies with:

```bash
cd booking-system-frontend
npm install
```

Additionally, the project uses Framer Motion for toast and component animations. If you want these animations, ensure `framer-motion` is installed (it is listed in optionalDependencies):

```bash
cd booking-system-frontend
npm install framer-motion
```

If you want to run tests (Vitest + MSW):

```bash
npm test

### Navigation (React Router)

The frontend now uses React Router for client-side navigation. Routes available:

- `/` → Home
- `/providers` → Providers list
- `/booking` → Booking flow
- `/services` → Services list
- `/admin` → Admin dashboard
- `/login` → Login page

The top navigation includes a mobile toggle (hamburger) that opens the mobile menu. Links are standard anchor-style client routes using React Router's `Link` component.

If you pull the repository and see navigation-related TypeScript errors, ensure `react-router-dom` is installed in the frontend:

```bash
cd booking-system-frontend
npm install react-router-dom
```
```

### Toast examples & i18n

The frontend exposes a small toast helper in `src/lib/toast.tsx`. Examples:

```ts
import toast, { setLocale } from './src/lib/toast';

// simple message
toast.success('Booking created successfully');

// use i18n key (prefix with msg:)
toast.success('msg:booking_success');

// change locale to English
setLocale('en');
```

To ensure animations work, install `framer-motion` as dependency.

```

2. Inisialisasi file service worker ke folder `public/` (hanya perlu dilakukan sekali):

```bash
npx msw init public/ --save
```

3. MSW akan otomatis dimulai ketika `import.meta.env.DEV` bernilai true. Mock tersedia di `src/mocks/handlers.ts` dan `src/mocks/browser.ts`. Contoh endpoint yang dimock: `/health` dan `/providers`.

Catatan: bila menjalankan lewat Docker Compose, frontend dijalankan sebagai build/production container dan MSW development worker tidak akan aktif. Untuk workflow MSW gunakan `npm run dev` lokal.


## QA Pipeline Manual

1. `go test ./...` di direktori backend.
2. `npm run lint && npm test && npm run build` di direktori frontend.
3. Pastikan seluruh langkah lint → test → build berhasil sebelum membuat commit.
