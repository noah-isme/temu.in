import type { BookingPaymentStatus, BookingStatus } from '../api';

const STATUS_LABELS: Record<BookingStatus, string> = {
  pending: 'Menunggu',
  confirmed: 'Dikonfirmasi',
  completed: 'Selesai',
  cancelled: 'Dibatalkan',
  payment_failed: 'Pembayaran gagal',
};

const STATUS_CLASSES: Record<BookingStatus, string> = {
  pending: 'bg-amber-50 text-amber-800 border-amber-200',
  confirmed: 'bg-green-50 text-green-800 border-green-200',
  completed: 'bg-slate-100 text-slate-700 border-slate-200',
  cancelled: 'bg-red-50 text-red-700 border-red-200',
  payment_failed: 'bg-red-50 text-red-700 border-red-200',
};

const PAYMENT_LABELS: Record<BookingPaymentStatus, string> = {
  unpaid: 'belum dibayar',
  pending: 'menunggu',
  paid: 'lunas',
  failed: 'gagal',
  not_required: 'tidak diperlukan',
};

export function BookingStatusBadge({ status }: { status: BookingStatus }): JSX.Element {
  const label = STATUS_LABELS[status] ?? status;
  const cls = STATUS_CLASSES[status] ?? 'bg-slate-100 text-slate-700 border-slate-200';
  return <span className={`inline-block rounded border px-2 py-0.5 text-xs font-medium ${cls}`}>{label}</span>;
}

export function PaymentStatusBadge({ status }: { status: BookingPaymentStatus }): JSX.Element {
  const ok = status === 'paid';
  return (
    <span className={`inline-block rounded border px-2 py-0.5 text-xs ${ok ? 'border-green-200 text-green-800' : 'border-slate-200 text-slate-600'}`}>
      Pembayaran: {PAYMENT_LABELS[status] ?? status}
    </span>
  );
}
