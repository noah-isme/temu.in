import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import RegisterPage from '../RegisterPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';
import { useAppStore } from '../../store/app-store';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage() {
  return renderRoutes(<Route path="/register" element={<RegisterPage />} />, '/register');
}

describe('RegisterPage', () => {
  afterEach(() => {
    vi.clearAllMocks();
    useAppStore.setState({ user: null });
    localStorage.clear();
  });

  it('registers, stores the token and user, and redirects home', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText('Nama'), 'Ani');
    await user.type(screen.getByLabelText('Email'), 'ani@user.test');
    await user.type(screen.getByLabelText('Password'), 'secret1');
    await user.click(screen.getByRole('button', { name: 'Daftar' }));

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/));
    expect(toast.success).toHaveBeenCalledWith('Akun berhasil dibuat', '');
    expect(localStorage.getItem('auth_token')).toBe('mock-jwt-token');
    expect(useAppStore.getState().user?.email).toBe('ani@user.test');
  });

  it('shows inline validation errors without calling the API', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText('Email'), 'not-an-email');
    await user.type(screen.getByLabelText('Password'), '123');
    await user.click(screen.getByRole('button', { name: 'Daftar' }));

    expect(await screen.findByText('Format email tidak valid')).toBeInTheDocument();
    expect(screen.getByText('Password minimal 6 karakter')).toBeInTheDocument();
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.getByTestId('location')).toHaveTextContent('/register');
  });

  it('shows the server error inline and as a toast on 400', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText('Email'), 'taken@user.test');
    await user.type(screen.getByLabelText('Password'), 'secret1');
    await user.click(screen.getByRole('button', { name: 'Daftar' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('email exists');
    expect(toast.error).toHaveBeenCalledWith('email exists', 'Pendaftaran gagal');
    expect(screen.getByTestId('location')).toHaveTextContent('/register');
  });
});
