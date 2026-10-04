import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import ResetPasswordPage from '../ResetPasswordPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage(entry: string) {
  return renderRoutes(<Route path="/reset-password" element={<ResetPasswordPage />} />, entry);
}

describe('ResetPasswordPage', () => {
  afterEach(() => vi.clearAllMocks());

  it('resets the password with the token from the URL and redirects to login', async () => {
    const user = userEvent.setup();
    renderPage('/reset-password?token=good-token');

    await user.type(screen.getByLabelText('Password baru'), 'newpass1');
    await user.type(screen.getByLabelText('Ulangi password'), 'newpass1');
    await user.click(screen.getByRole('button', { name: 'Simpan password' }));

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/login$/));
    expect(toast.success).toHaveBeenCalledWith('Password diperbarui, silakan masuk', '');
  });

  it('validates password length and confirmation inline', async () => {
    const user = userEvent.setup();
    renderPage('/reset-password?token=good-token');

    await user.type(screen.getByLabelText('Password baru'), '123');
    await user.type(screen.getByLabelText('Ulangi password'), '1234');
    await user.click(screen.getByRole('button', { name: 'Simpan password' }));

    expect(await screen.findByText('Password minimal 6 karakter')).toBeInTheDocument();
    expect(screen.getByText('Konfirmasi password tidak sama')).toBeInTheDocument();
    expect(toast.success).not.toHaveBeenCalled();
  });

  it('shows the server error for an expired token', async () => {
    const user = userEvent.setup();
    renderPage('/reset-password?token=expired');

    await user.type(screen.getByLabelText('Password baru'), 'newpass1');
    await user.type(screen.getByLabelText('Ulangi password'), 'newpass1');
    await user.click(screen.getByRole('button', { name: 'Simpan password' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid or expired token');
    expect(toast.error).toHaveBeenCalledWith('invalid or expired token', 'Reset password gagal');
    expect(screen.getByTestId('location')).toHaveTextContent('/reset-password');
  });

  it('shows an error state when the token is missing', () => {
    renderPage('/reset-password');
    expect(screen.getByRole('alert')).toHaveTextContent('token tidak ditemukan');
    expect(screen.queryByLabelText('Password baru')).not.toBeInTheDocument();
  });
});
