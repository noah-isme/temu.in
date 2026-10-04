import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import ForgotPasswordPage from '../ForgotPasswordPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';
import { server } from '../../mocks/server';
import { mockRoute } from '../../mocks/handlers';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage() {
  return renderRoutes(<Route path="/forgot-password" element={<ForgotPasswordPage />} />, '/forgot-password');
}

describe('ForgotPasswordPage', () => {
  afterEach(() => vi.clearAllMocks());

  it('sends the reset link and shows confirmation', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText('Email'), 'ani@user.test');
    await user.click(screen.getByRole('button', { name: 'Kirim tautan reset' }));

    expect(await screen.findByRole('status')).toHaveTextContent('tautan reset telah dikirim');
    expect(toast.success).toHaveBeenCalledWith('Jika email terdaftar, tautan reset telah dikirim', '');
  });

  it('validates the email inline', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole('button', { name: 'Kirim tautan reset' }));

    expect(await screen.findByText('Email wajib diisi')).toBeInTheDocument();
    expect(toast.success).not.toHaveBeenCalled();
  });

  it('shows a server error inline and as a toast', async () => {
    server.use(...mockRoute('post', '/auth/forgot-password', (_req, res, ctx) => res(ctx.status(400), ctx.json({ error: 'validation failed' }))));
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText('Email'), 'ani@user.test');
    await user.click(screen.getByRole('button', { name: 'Kirim tautan reset' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('validation failed');
    expect(toast.error).toHaveBeenCalledWith('validation failed', 'Reset password gagal');
  });
});
