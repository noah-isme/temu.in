
import '@testing-library/jest-dom/vitest';
import { server } from './src/mocks/server';

// Start MSW server for tests
beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
