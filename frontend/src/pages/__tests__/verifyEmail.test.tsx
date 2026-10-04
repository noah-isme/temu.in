import { screen } from '@testing-library/react';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import VerifyEmailPage from '../VerifyEmailPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage(entry: string) {
  return renderRoutes(<Route path="/verify-email" element={<VerifyEmailPage />} />, entry);
}

describe('VerifyEmailPage', () => {
  afterEach(() => vi.clearAllMocks());

  it('verifies the token on mount and links to login', async () => {
    renderPage('/verify-email?token=good-token');

    expect(screen.getByRole('status')).toHaveTextContent('Memverifikasi email');
    expect(await screen.findByText('Email terverifikasi')).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith('Email terverifikasi', '');
    expect(screen.getByRole('link', { name: 'Ke halaman masuk' })).toHaveAttribute('href', '/login');
  });

  it('shows the server error for an invalid token', async () => {
    renderPage('/verify-email?token=expired');

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid or expired token');
    expect(toast.error).toHaveBeenCalledWith('invalid or expired token', 'Verifikasi email gagal');
  });

  it('shows an error state without a token and makes no request', () => {
    renderPage('/verify-email');

    expect(screen.getByRole('alert')).toHaveTextContent('Token verifikasi tidak ditemukan');
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.getByRole('link', { name: 'Ke halaman masuk' })).toBeInTheDocument();
  });
});
