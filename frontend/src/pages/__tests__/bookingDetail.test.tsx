import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route } from 'react-router-dom';
import { vi } from 'vitest';
import BookingDetailPage from '../BookingDetailPage';
import { renderRoutes } from './testUtils';
import * as toast from '../../lib/toast';
import { server } from '../../mocks/server';
import { mockRoute, mockSlots, mockState } from '../../mocks/handlers';

vi.mock('../../lib/toast', () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warn: vi.fn(), setLocale: vi.fn() }));

function renderPage(id: number) {
  return renderRoutes(<Route path="/bookings/:id" element={<BookingDetailPage />} />, `/bookings/${id}`);
}

// Instant availability (the shared mock adds random latency).
function instantAvailability() {
  server.use(
    ...mockRoute('get', '/availability', (req, res, ctx) => {
      const date = req.url.searchParams.get('date') ?? '2025-07-10';
      const slots = mockSlots(date, 30);
      return res(ctx.status(200), ctx.json({ date, provider_id: 'p1', service_id: 's1', available_slots: slots, slots }));
    })
  );
}

describe('BookingDetailPage', () => {
  afterEach(() => vi.clearAllMocks());

  it('cancels a booking with a reason and shows the cancelled status', async () => {
    const user = userEvent.setup();
    renderPage(101);

    expect(await screen.findByTestId('booking-status')).toHaveTextContent('Menunggu');

    await user.click(screen.getByRole('button', { name: 'Batalkan booking' }));
    expect(screen.getByText('Alasan pembatalan wajib diisi')).toBeInTheDocument();

    await user.type(screen.getByLabelText('Alasan pembatalan'), 'Berhalangan');
    await user.click(screen.getByRole('button', { name: 'Batalkan booking' }));

    await waitFor(() => expect(screen.getByTestId('booking-status')).toHaveTextContent('Dibatalkan'));
    expect(toast.success).toHaveBeenCalledWith('Booking dibatalkan', '');
    expect(mockState.bookings.find((b) => b.id === 101)?.cancellation_reason).toBe('Berhalangan');
    expect(screen.getByRole('button', { name: 'Batalkan booking' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Jadwal ulang' })).toBeDisabled();
  });

  it('disables cancel and reschedule for a completed booking', async () => {
    renderPage(103);
    expect(await screen.findByTestId('booking-status')).toHaveTextContent('Selesai');
    expect(screen.getByRole('button', { name: 'Batalkan booking' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Jadwal ulang' })).toBeDisabled();
  });

  it('reschedules to a picked slot and shows success', async () => {
    instantAvailability();
    const user = userEvent.setup();
    renderPage(101);

    await user.click(await screen.findByRole('button', { name: 'Jadwal ulang' }));
    const dialog = screen.getByRole('dialog', { name: 'Jadwal ulang booking' });
    expect(dialog).toBeInTheDocument();

    const dateInput = screen.getByLabelText('Tanggal baru');
    await user.clear(dateInput);
    await user.type(dateInput, '2025-07-10');
    await user.click(await screen.findByRole('button', { name: '10:00–10:30' }));
    await user.click(screen.getByRole('button', { name: 'Simpan jadwal' }));

    await waitFor(() => expect(toast.success).toHaveBeenCalledWith('Booking dijadwalkan ulang', ''));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(await screen.findByText('10:00–10:30')).toBeInTheDocument();
    expect(mockState.bookings.find((b) => b.id === 101)).toMatchObject({ booking_date: '2025-07-10', start_time: '10:00' });
  });

  it('shows the conflict message on 409', async () => {
    instantAvailability();
    server.use(
      ...mockRoute('put', '/bookings/:bookingId', (_req, res, ctx) =>
        res(ctx.status(409), ctx.json({ error: 'requested time slot is not available' }))
      )
    );
    const user = userEvent.setup();
    renderPage(101);

    await user.click(await screen.findByRole('button', { name: 'Jadwal ulang' }));
    await user.click(await screen.findByRole('button', { name: '09:00–09:30' }));
    await user.click(screen.getByRole('button', { name: 'Simpan jadwal' }));

    const conflict = 'Slot waktu tersebut sudah tidak tersedia, silakan pilih yang lain';
    expect(await screen.findByRole('alert')).toHaveTextContent(conflict);
    expect(toast.error).toHaveBeenCalledWith(conflict, 'Booking gagal');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('shows not found for an unknown booking', async () => {
    renderPage(999);
    expect(await screen.findByRole('alert')).toHaveTextContent('Booking tidak ditemukan');
  });
});
