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

## QA Pipeline Manual

1. `go test ./...` di direktori backend.
2. `npm run lint && npm test && npm run build` di direktori frontend.
3. Pastikan seluruh langkah lint → test → build berhasil sebelum membuat commit.
