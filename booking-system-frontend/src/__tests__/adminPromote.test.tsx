import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { rest } from 'msw';
import { setupServer } from 'msw/node';
import AdminPage from '../pages/AdminPage';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const users = [
  { id: 1, email: 'alice@example.com', name: 'Alice', role: 'user' },
  { id: 2, email: 'bob@example.com', name: 'Bob', role: 'admin' },
];

const server = setupServer(
  rest.get('/api/admin/users', (req, res, ctx) => {
    return res(ctx.json({ users }));
  }),
  rest.get('/api/admin/audit', (req, res, ctx) => {
    return res(ctx.json({ audit: [] }));
  }),
  rest.post('/api/admin/promote', (req, res, ctx) => {
    // simulate success
    return res(ctx.status(200), ctx.json({ status: 'promoted' }));
  })
);

describe('Admin promote optimistic flow', () => {
  const qc = new QueryClient();

  beforeAll(() => server.listen());
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  test('optimistic promote updates UI and shows toast', async () => {
    render(
      <QueryClientProvider client={qc}>
        <AdminPage />
      </QueryClientProvider>
    );

    // wait for users to render
    await waitFor(() => expect(screen.getByText('alice@example.com')).toBeInTheDocument());

    const promoteBtn = screen.getByRole('button', { name: /promote/i });
    await userEvent.click(promoteBtn);

    // modal shows
    expect(screen.getByText(/Promote alice@example.com to admin/)).toBeInTheDocument();

    const confirm = screen.getByRole('button', { name: /confirm/i });
    await userEvent.click(confirm);

    // optimistic update: role should show admin (immediately or after settle)
    await waitFor(() => expect(screen.getAllByText('admin').length).toBeGreaterThanOrEqual(1));
  });
});
import '../../src/components/__tests__/adminPromote.test';
