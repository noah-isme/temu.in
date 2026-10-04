import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { completeBooking, confirmBooking, fetchProviderBookings, getApiErrorMessage } from '../../api';
import type { BookingStatus } from '../../api';
import { ErrorState, LoadingState } from '../QueryFeedback';
import { BookingStatusBadge, PaymentStatusBadge } from '../BookingStatusBadge';
import { formatClock, formatDate } from '../../lib/datetime';
import * as toast from '../../lib/toast';
import { t } from '../../lib/i18n';

const STATUS_OPTIONS: Array<{ value: '' | BookingStatus; label: string }> = [
  { value: '', label: 'Semua' },
  { value: 'pending', label: 'Menunggu' },
  { value: 'confirmed', label: 'Dikonfirmasi' },
  { value: 'completed', label: 'Selesai' },
  { value: 'cancelled', label: 'Dibatalkan' },
  { value: 'payment_failed', label: 'Pembayaran gagal' },
];

type Action = { id: number; kind: 'confirm' | 'complete' };

export default function ProviderBookingsTable({ providerId }: { providerId: string }): JSX.Element {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<'' | BookingStatus>('');
  const baseKey = ['provider', providerId, 'bookings'];

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery([...baseKey, status], () =>
    fetchProviderBookings(providerId, status || undefined)
  );

  const mutation = useMutation(({ id, kind }: Action) => (kind === 'confirm' ? confirmBooking(id) : completeBooking(id)), {
    onSuccess: () => {
      toast.success(t('booking_status_updated'), '');
      queryClient.invalidateQueries(baseKey);
    },
    // Backend 400s carry actionable text, e.g. "payment must succeed before confirming booking".
    onError: (err: unknown) => toast.error(getApiErrorMessage(err, t('booking_failed')), t('booking_failed')),
  });

  return (
    <div>
      <div className="mb-4">
        <label htmlFor="provider-bookings-status" className="block text-xs text-slate-600">Filter status</label>
        <select
          id="provider-bookings-status"
          value={status}
          onChange={(e) => setStatus(e.target.value as '' | BookingStatus)}
          className="rounded border p-1 text-sm"
        >
          {STATUS_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>

      {isLoading && <LoadingState label="Memuat booking..." />}
      {isError && <ErrorState title="Gagal memuat booking" error={error} onRetry={() => refetch()} retrying={isFetching} />}
      {data && data.length === 0 && <p className="text-sm text-slate-600">Tidak ada booking.</p>}

      {data && data.length > 0 && (
        <table className="w-full text-left text-sm" aria-label="Booking provider">
          <thead>
            <tr className="border-b text-slate-500">
              <th className="py-2">Pelanggan</th>
              <th>Layanan</th>
              <th>Tanggal</th>
              <th>Waktu</th>
              <th>Status</th>
              <th><span className="sr-only">Aksi</span></th>
            </tr>
          </thead>
          <tbody>
            {data.map((b) => {
              const busy = mutation.isLoading && mutation.variables?.id === b.id;
              return (
                <tr key={b.id} className="border-b" data-testid={`provider-booking-${b.id}`}>
                  <td className="py-2">{b.customer_name}</td>
                  <td>{b.service_name || '-'}</td>
                  <td>{formatDate(b.booking_date)}</td>
                  <td>{formatClock(b.start_time)}–{formatClock(b.end_time)}</td>
                  <td>
                    <div className="flex flex-wrap gap-1">
                      <BookingStatusBadge status={b.status} />
                      <PaymentStatusBadge status={b.payment_status} />
                    </div>
                  </td>
                  <td className="py-2 text-right">
                    {b.status === 'pending' && (
                      <button type="button" disabled={busy} onClick={() => mutation.mutate({ id: b.id, kind: 'confirm' })} className="rounded bg-primary px-2 py-0.5 text-white disabled:opacity-60" aria-label={`Konfirmasi booking ${b.customer_name}`}>
                        Konfirmasi
                      </button>
                    )}
                    {b.status === 'confirmed' && (
                      <button type="button" disabled={busy} onClick={() => mutation.mutate({ id: b.id, kind: 'complete' })} className="rounded border px-2 py-0.5 disabled:opacity-60" aria-label={`Selesaikan booking ${b.customer_name}`}>
                        Selesaikan
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}
