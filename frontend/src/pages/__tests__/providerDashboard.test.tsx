import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import ProviderDashboardPage from '../ProviderDashboardPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';
import { MOCK_PROVIDER, mockState } from '../../mocks/handlers';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage() {
  return renderRoutes(<Route path="/dashboard" element={<ProviderDashboardPage />} />, '/dashboard');
}

async function openTab(name: string) {
  await userEvent.click(await screen.findByRole('tab', { name }));
}

describe('ProviderDashboardPage', () => {
  beforeEach(() => {
    mockState.myProvider = MOCK_PROVIDER;
  });
  afterEach(() => {
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

  it('shows an empty state when the user is not a provider', async () => {
    mockState.myProvider = null;
    renderPage();
    expect(await screen.findByText('Anda belum mendaftar sebagai provider.')).toBeInTheDocument();
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
  });

  it('lists, creates, edits and deletes schedules', async () => {
    const user = userEvent.setup();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    renderPage();

    const table = await screen.findByRole('table', { name: 'Jadwal mingguan' });
    expect(within(table).getByText('Senin')).toBeInTheDocument();
    expect(within(table).getByText('09:00')).toBeInTheDocument();

    const addForm = screen.getByRole('form', { name: 'Tambah jadwal' });
    await user.selectOptions(within(addForm).getByLabelText('Hari'), 'Rabu');
    fireEvent.change(within(addForm).getByLabelText('Jam mulai'), { target: { value: '10:00' } });
    fireEvent.change(within(addForm).getByLabelText('Jam selesai'), { target: { value: '15:30' } });
    await user.click(within(addForm).getByRole('button', { name: 'Tambah jadwal' }));

    expect(await within(table).findByText('Rabu')).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith('Jadwal disimpan', '');
    expect(mockState.schedules.find((s) => s.day_of_week === 3)).toMatchObject({ start_time: '10:00', end_time: '15:30', is_available: true });

    // Inline edit is pre-filled with the HH:MM DTO values and round-trips them.
    await user.click(screen.getByRole('button', { name: 'Ubah jadwal Senin 09:00' }));
    const editForm = screen.getByRole('form', { name: 'Ubah jadwal Senin' });
    expect(within(editForm).getByLabelText('Jam mulai')).toHaveValue('09:00');
    fireEvent.change(within(editForm).getByLabelText('Jam selesai'), { target: { value: '12:00' } });
    await user.click(within(editForm).getByRole('button', { name: 'Simpan' }));
    expect(await within(table).findByText('12:00')).toBeInTheDocument();
    expect(mockState.schedules.find((s) => s.id === 'sch1')).toMatchObject({ start_time: '09:00', end_time: '12:00' });

    await user.click(screen.getByRole('button', { name: 'Hapus jadwal Rabu 10:00' }));
    expect(window.confirm).toHaveBeenCalled();
    await waitFor(() => expect(within(table).queryByText('Rabu')).not.toBeInTheDocument());
    expect(toast.success).toHaveBeenCalledWith('Jadwal dihapus', '');
  });

  it('rejects a schedule whose end is not after its start', async () => {
    const user = userEvent.setup();
    renderPage();
    const addForm = await screen.findByRole('form', { name: 'Tambah jadwal' });
    fireEvent.change(within(addForm).getByLabelText('Jam selesai'), { target: { value: '08:00' } });
    await user.click(within(addForm).getByRole('button', { name: 'Tambah jadwal' }));
    expect(within(addForm).getByRole('alert')).toHaveTextContent('Jam selesai harus setelah jam mulai');
  });

  it('creates a full-day exception in the selected range', async () => {
    const user = userEvent.setup();
    renderPage();
    await openTab('Pengecualian');

    fireEvent.change(screen.getByLabelText('Dari'), { target: { value: '2025-07-01' } });
    fireEvent.change(screen.getByLabelText('Sampai'), { target: { value: '2025-07-31' } });
    expect(await screen.findByText('Tidak ada pengecualian pada rentang ini.')).toBeInTheDocument();

    const form = screen.getByRole('form', { name: 'Tambah pengecualian' });
    fireEvent.change(within(form).getByLabelText('Tanggal'), { target: { value: '2025-07-17' } });
    await user.type(within(form).getByLabelText('Alasan'), 'Libur');
    await user.click(within(form).getByRole('button', { name: 'Tambah pengecualian' }));

    const table = await screen.findByRole('table', { name: 'Pengecualian jadwal' });
    expect(within(table).getByText('Sepanjang hari')).toBeInTheDocument();
    expect(within(table).getByText('Libur')).toBeInTheDocument();
    expect(toast.success).toHaveBeenCalledWith('Pengecualian disimpan', '');
    expect(mockState.exceptions[0]).toMatchObject({ date: '2025-07-17', start_time: null, end_time: null, is_available: false });
  });

  it('confirms a pending booking', async () => {
    const user = userEvent.setup();
    renderPage();
    await openTab('Booking');

    const row = await screen.findByTestId('provider-booking-201');
    expect(row).toHaveTextContent('Budi');
    expect(row).toHaveTextContent('Menunggu');
    await user.click(within(row).getByRole('button', { name: 'Konfirmasi booking Budi' }));

    await waitFor(() => expect(screen.getByTestId('provider-booking-201')).toHaveTextContent('Dikonfirmasi'));
    expect(toast.success).toHaveBeenCalledWith('Status booking diperbarui', '');
  });

  it('surfaces the backend 400 message when completing fails', async () => {
    const user = userEvent.setup();
    renderPage();
    await openTab('Booking');

    const row = await screen.findByTestId('provider-booking-202');
    await user.click(within(row).getByRole('button', { name: 'Selesaikan booking Sari' }));

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith('payment must succeed before completion', 'Booking gagal'));
    expect(screen.getByTestId('provider-booking-202')).toHaveTextContent('Dikonfirmasi');
  });

  it('filters bookings by status', async () => {
    const user = userEvent.setup();
    renderPage();
    await openTab('Booking');
    await screen.findByTestId('provider-booking-201');

    await user.selectOptions(screen.getByLabelText('Filter status'), 'confirmed');
    await waitFor(() => expect(screen.queryByTestId('provider-booking-201')).not.toBeInTheDocument());
    expect(screen.getByTestId('provider-booking-202')).toBeInTheDocument();
  });
});
