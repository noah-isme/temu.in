import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useParams } from 'react-router-dom';
import { cancelBooking, fetchBooking, getApiErrorMessage, getApiErrorStatus } from '../api';
import type { BookingStatus } from '../api';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import RescheduleDialog from '../components/RescheduleDialog';
import { BookingStatusBadge, PaymentStatusBadge } from '../components/BookingStatusBadge';
import { formatClock, formatDate } from '../lib/datetime';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

// Mirrors backend booking/service.go: completed bookings cannot be cancelled
// (cancelled ones are already final); neither can be rescheduled.
const FINAL_STATUSES: BookingStatus[] = ['completed', 'cancelled'];

export default function BookingDetailPage(): JSX.Element {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState<string | null>(null);
  const [cancelError, setCancelError] = useState<string | null>(null);
  const [rescheduleOpen, setRescheduleOpen] = useState(false);

  const { data: booking, isLoading, isError, error, refetch, isFetching } = useQuery(['bookings', id], () => fetchBooking(id), {
    enabled: !!id,
  });

  const cancelMutation = useMutation((why: string) => cancelBooking(id, why), {
    onSuccess: () => {
      setReason('');
      toast.success(t('booking_cancelled'), '');
      queryClient.invalidateQueries(['bookings']);
    },
    onError: (err: unknown) => {
      const msg = getApiErrorMessage(err, t('booking_failed'));
      setCancelError(msg);
      toast.error(msg, t('booking_failed'));
    },
  });

  const isFinal = booking ? FINAL_STATUSES.includes(booking.status) : true;

  return (
    <PageTransition>
      <section className="max-w-2xl">
        <p className="mb-4 text-sm"><Link to="/bookings" className="text-primary underline">← Booking saya</Link></p>
        <h2 className="mb-6 text-2xl font-semibold">Detail booking</h2>

        {isLoading && (
          <p role="status" className="flex items-center gap-2 text-sm text-slate-700"><Spinner size={16} /> Memuat booking...</p>
        )}

        {isError && (
          <div role="alert" className="rounded border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <p>
              {getApiErrorStatus(error) === 404 ? 'Booking tidak ditemukan.' : `Gagal memuat booking: ${getApiErrorMessage(error, 'terjadi kesalahan')}`}
            </p>
            {getApiErrorStatus(error) !== 404 && (
              <button type="button" onClick={() => refetch()} disabled={isFetching} className="mt-2 rounded border border-red-300 px-3 py-1">
                Coba lagi
              </button>
            )}
          </div>
        )}

        {booking && (
          <>
            <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-2 rounded border bg-white p-4 text-sm">
              <dt className="text-slate-500">Layanan</dt>
              <dd>{booking.service_name || '-'}</dd>
              <dt className="text-slate-500">Provider</dt>
              <dd>{booking.provider_name || booking.provider_id}</dd>
              <dt className="text-slate-500">Tanggal</dt>
              <dd>{formatDate(booking.booking_date)}</dd>
              <dt className="text-slate-500">Waktu</dt>
              <dd>{formatClock(booking.start_time)}–{formatClock(booking.end_time)}</dd>
              <dt className="text-slate-500">Status</dt>
              <dd className="flex flex-wrap gap-2" data-testid="booking-status">
                <BookingStatusBadge status={booking.status} />
                <PaymentStatusBadge status={booking.payment_status} />
              </dd>
              {booking.cancellation_reason && (
                <>
                  <dt className="text-slate-500">Alasan batal</dt>
                  <dd>{booking.cancellation_reason}</dd>
                </>
              )}
            </dl>

            <div className="mt-6">
              <button
                type="button"
                onClick={() => setRescheduleOpen(true)}
                disabled={isFinal}
                className="rounded border px-3 py-1 text-sm disabled:opacity-50"
              >
                Jadwal ulang
              </button>
            </div>

            <form
              className="mt-6 rounded border bg-white p-4"
              noValidate
              onSubmit={(e) => {
                e.preventDefault();
                setCancelError(null);
                if (!reason.trim()) {
                  setReasonError('Alasan pembatalan wajib diisi');
                  return;
                }
                setReasonError(null);
                cancelMutation.mutate(reason.trim());
              }}
            >
              <label htmlFor="cancel-reason" className="mb-1 block text-sm">Alasan pembatalan</label>
              <textarea
                id="cancel-reason"
                required
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                disabled={isFinal}
                aria-invalid={!!reasonError}
                aria-describedby={reasonError ? 'cancel-reason-error' : undefined}
                className="w-full rounded border p-2 text-sm"
                rows={3}
              />
              {reasonError && <p id="cancel-reason-error" className="mt-1 text-sm text-red-600">{reasonError}</p>}
              {cancelError && <p role="alert" className="mt-1 text-sm text-red-600">{cancelError}</p>}
              <button
                type="submit"
                disabled={isFinal || cancelMutation.isLoading}
                className="mt-2 flex items-center gap-2 rounded bg-red-600 px-3 py-1 text-sm text-white disabled:opacity-50"
              >
                {cancelMutation.isLoading ? (<><Spinner size={16} /><span>Membatalkan...</span></>) : 'Batalkan booking'}
              </button>
            </form>

            {rescheduleOpen && (
              <RescheduleDialog booking={booking} open={rescheduleOpen} onClose={() => setRescheduleOpen(false)} />
            )}
          </>
        )}
      </section>
    </PageTransition>
  );
}
