import '@testing-library/jest-dom/vitest';
import { server } from './src/mocks/server';
import { resetMockState } from './src/mocks/handlers';

// Node >= 22 ships an experimental global `localStorage`/`sessionStorage`
// (enabled by default from Node 25). Without `--localstorage-file` its getter
// yields undefined and shadows jsdom's Storage, so `localStorage.getItem`
// throws. Point both globals back at the jsdom window's own Storage objects.
// No-op on Node 20 (CI) where jsdom's Storage is already installed.
type JsdomHandle = { window: Pick<Window, 'localStorage' | 'sessionStorage'> };
const dom = (globalThis as typeof globalThis & { jsdom?: JsdomHandle }).jsdom;
if (dom) {
  for (const key of ['localStorage', 'sessionStorage'] as const) {
    const storage = dom.window[key];
    if (globalThis[key] !== storage) {
      Object.defineProperty(globalThis, key, { value: storage, configurable: true, writable: true });
    }
  }
}

// Start MSW server for tests
beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  resetMockState();
});
afterAll(() => server.close());
