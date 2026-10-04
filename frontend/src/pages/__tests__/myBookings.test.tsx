import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import MyBookingsPage from '../MyBookingsPage';
import RequireAuth from '../../components/RequireAuth';
import { renderRoutes } from './testUtils';
import { server } from '../../mocks/server';
import { mockRoute, mockState } from '../../mocks/handlers';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage() {
  return renderRoutes(
    <>
      <Route path="/bookings" element={<RequireAuth><MyBookingsPage /></RequireAuth>} />
      <Route path="/login" element={<p>login page</p>} />
    </>,
    '/bookings'
  );
}

describe('MyBookingsPage', () => {
  beforeEach(() => localStorage.setItem('auth_token', 'mock-jwt-token'));
  afterEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('lists the mocked bookings with formatted date, time and status', async () => {
    renderPage();

    expect(screen.getByRole('status')).toHaveTextContent('Memuat booking');
    const list = await screen.findByRole('list', { name: 'Daftar booking' });
    const items = within(list).getAllByRole('listitem');
    expect(items).toHaveLength(3);

    const first = items[0];
    expect(first).toHaveTextContent('Haircut · Provider A');
    expect(first).toHaveTextContent('09:00–09:30');
    expect(first).toHaveTextContent('2025');
    expect(first).toHaveTextContent('Menunggu');
    expect(within(first).getByRole('link')).toHaveAttribute('href', '/bookings/101');
    expect(items[2]).toHaveTextContent('Selesai');
  });

  it('shows an empty state', async () => {
    mockState.bookings = [];
    renderPage();
    expect(await screen.findByText('Belum ada booking.')).toBeInTheDocument();
  });

  it('shows an error with a working retry', async () => {
    let calls = 0;
    server.use(
      ...mockRoute('get', '/bookings', (_req, res, ctx) => {
        calls += 1;
        if (calls === 1) return res(ctx.status(500), ctx.json({ error: 'failed to list bookings' }));
        return res(ctx.status(200), ctx.json({ bookings: [] }));
      })
    );
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByRole('alert')).toHaveTextContent('failed to list bookings');
    await user.click(screen.getByRole('button', { name: 'Coba lagi' }));
    expect(await screen.findByText('Belum ada booking.')).toBeInTheDocument();
  });

  it('redirects anonymous visitors to login', () => {
    localStorage.clear();
    renderPage();
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/login$/);
    expect(screen.getByText('login page')).toBeInTheDocument();
  });
});
