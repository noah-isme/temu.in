/* eslint-disable react-refresh/only-export-components -- test-only helper module, never hot-reloaded */
import type { ReactElement, ReactNode } from 'react';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, useLocation } from 'react-router-dom';

function LocationProbe(): JSX.Element {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    logger: { log: () => undefined, warn: () => undefined, error: () => undefined },
  });
}

/** Renders `routes` inside a MemoryRouter + fresh QueryClient; `location` testid shows the current URL. */
export function renderRoutes(routes: ReactNode, initialEntry: string): ReturnType<typeof render> & { queryClient: QueryClient } {
  const queryClient = createTestQueryClient();
  const ui: ReactElement = (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>{routes}</Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>
  );
  return { queryClient, ...render(ui) };
}
