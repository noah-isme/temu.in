import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { fetchMyBookings, getApiErrorMessage } from '../api';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import { BookingStatusBadge, PaymentStatusBadge } from '../components/BookingStatusBadge';
import { formatClock, formatDate } from '../lib/datetime';

export default function MyBookingsPage(): JSX.Element {
  const { data, isLoading, isError, error, refetch, isFetching } = useQuery(['bookings'], () => fetchMyBookings());

  return (
    <PageTransition>
      <section className="max-w-3xl">
        <h2 className="mb-6 text-2xl font-semibold">Booking saya</h2>

        {isLoading && (
          <p role="status" className="flex items-center gap-2 text-sm text-slate-700"><Spinner size={16} /> Memuat booking...</p>
        )}

        {isError && (
          <div role="alert" className="rounded border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <p>Gagal memuat booking: {getApiErrorMessage(error, 'terjadi kesalahan')}</p>
            <button type="button" onClick={() => refetch()} disabled={isFetching} className="mt-2 rounded border border-red-300 px-3 py-1">
              Coba lagi
            </button>
          </div>
        )}

        {data && data.length === 0 && (
          <div className="rounded border bg-white p-6 text-sm text-slate-600">
            <p>Belum ada booking.</p>
            <Link to="/booking" className="text-primary underline">Buat booking</Link>
          </div>
        )}

        {data && data.length > 0 && (
          <ul className="grid gap-3" aria-label="Daftar booking">
            {data.map((b) => (
              <li key={b.id}>
                <Link to={`/bookings/${b.id}`} className="block rounded border bg-white p-4 transition hover:border-primary">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <p className="font-medium">{b.service_name || 'Layanan'} · {b.provider_name || b.provider_id}</p>
                      <p className="text-sm text-slate-600">
                        {formatDate(b.booking_date)} · {formatClock(b.start_time)}–{formatClock(b.end_time)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <BookingStatusBadge status={b.status} />
                      <PaymentStatusBadge status={b.payment_status} />
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </PageTransition>
  );
}
