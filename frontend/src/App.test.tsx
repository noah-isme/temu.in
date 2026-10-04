import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import { server } from './mocks/server';
import { MOCK_PROVIDER, mockRoute, mockState } from './mocks/handlers';
import { useAppStore } from './store/app-store';

// Same providers main.tsx renders App inside.
function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
    logger: { log: () => undefined, warn: () => undefined, error: () => undefined },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  );
}

describe('App', () => {
  afterEach(() => useAppStore.setState({ user: null, initialized: false }));

  it('renders the hero headline', () => {
    renderApp();
    expect(screen.getByText(/Solusi Smart Booking/i)).toBeInTheDocument();
  });

  it('shows the Dashboard link only for a signed-in provider owner', async () => {
    mockState.myProvider = MOCK_PROVIDER;
    server.use(
      ...mockRoute('get', '/me', (_req, res, ctx) =>
        res(ctx.status(200), ctx.json({ user: { id: 1, email: 'owner@user.test', role: 'user', name: 'Owner' } }))
      )
    );
    renderApp();

    expect(await screen.findByRole('link', { name: 'Dashboard' })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('link', { name: 'Booking saya' })).toBeInTheDocument();
  });

  it('does not call /providers/me or show Dashboard for anonymous visitors', async () => {
    let providerMeCalls = 0;
    server.use(
      ...mockRoute('get', '/me', (_req, res, ctx) => res(ctx.status(200), ctx.json({ user: null }))),
      ...mockRoute('get', '/providers/me', (_req, res, ctx) => {
        providerMeCalls += 1;
        return res(ctx.status(200), ctx.json({ provider: MOCK_PROVIDER }));
      })
    );
    renderApp();

    await waitFor(() => expect(useAppStore.getState().initialized).toBe(true));
    expect(screen.queryByRole('link', { name: 'Dashboard' })).not.toBeInTheDocument();
    expect(providerMeCalls).toBe(0);
  });
});
