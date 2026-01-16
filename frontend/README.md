# Frontend (development)

The frontend runs with MSW (Mock Service Worker) enabled by default in development so you can work without a running backend.

To disable MSW and have the frontend call the real backend instead, set the Vite environment variable `VITE_USE_MSW` to `false` before running the dev server. Example:

```bash
# disable MSW for this session and run dev server
VITE_USE_MSW=false npm run dev
```

Or add a `.env` file in the `frontend/` folder containing:

```
VITE_USE_MSW=false
VITE_API_URL=http://localhost:8080/api
```

Note: MSW is still used in tests (Vitest) via the test setup. Disabling MSW only affects the browser worker started in `src/main.tsx`.
