import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import Modal from './Modal';
import Spinner from './Spinner';
import { fetchAvailabilityForDate, getApiErrorMessage, getApiErrorStatus, rescheduleBooking } from '../api';
import type { Booking, TimeSlot } from '../api';
import { formatClock } from '../lib/datetime';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

type Props = {
  booking: Booking;
  open: boolean;
  onClose: () => void;
};

export default function RescheduleDialog({ booking, open, onClose }: Props): JSX.Element | null {
  const [date, setDate] = useState(booking.booking_date);
  const [selected, setSelected] = useState<TimeSlot | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const serviceId = booking.service_id ?? '';

  const slotsQuery = useQuery(
    ['availability', booking.provider_id, serviceId, date],
    () => fetchAvailabilityForDate(booking.provider_id, date, serviceId),
    { enabled: open && !!serviceId && !!date }
  );

  const mutation = useMutation((slot: TimeSlot) => rescheduleBooking(booking.id, slot), {
    onSuccess: () => {
      toast.success(t('booking_rescheduled'), '');
      queryClient.invalidateQueries(['bookings']);
      onClose();
    },
    onError: (err: unknown) => {
      const msg = getApiErrorStatus(err) === 409 ? t('booking_conflict') : getApiErrorMessage(err, t('booking_failed'));
      setSubmitError(msg);
      toast.error(msg, t('booking_failed'));
    },
  });

  const slots = slotsQuery.data?.slots ?? [];

  return (
    <Modal title="Jadwal ulang booking" open={open} onClose={onClose}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          setSubmitError(null);
          if (selected) mutation.mutate(selected);
        }}
      >
        <label htmlFor="reschedule-date" className="mb-1 block text-sm">Tanggal baru</label>
        <input
          id="reschedule-date"
          type="date"
          value={date}
          onChange={(e) => {
            setDate(e.target.value);
            setSelected(null);
          }}
          className="mb-3 rounded border p-2"
        />

        <fieldset>
          <legend className="mb-1 text-sm">Slot tersedia</legend>
          {!serviceId && <p className="text-sm text-slate-600">Layanan booking tidak diketahui; slot tidak dapat dimuat.</p>}
          {slotsQuery.isFetching && (
            <p role="status" className="flex items-center gap-2 text-sm text-slate-700"><Spinner size={16} /> Memuat slot...</p>
          )}
          {slotsQuery.isError && (
            <p role="alert" className="text-sm text-red-600">
              Gagal memuat slot.{' '}
              <button type="button" className="underline" onClick={() => slotsQuery.refetch()}>Coba lagi</button>
            </p>
          )}
          {slotsQuery.isSuccess && !slotsQuery.isFetching && slots.length === 0 && (
            <p className="text-sm text-slate-600">Tidak ada slot pada tanggal ini.</p>
          )}
          <div className="flex flex-wrap gap-2">
            {slots.map((slot) => {
              const active = selected?.start_time === slot.start_time && selected?.end_time === slot.end_time;
              return (
                <button
                  key={`${slot.date}-${slot.start_time}`}
                  type="button"
                  aria-pressed={active}
                  onClick={() => setSelected(slot)}
                  className={`rounded border px-3 py-1 text-sm ${active ? 'border-primary bg-primary text-white' : 'bg-white'}`}
                >
                  {formatClock(slot.start_time)}–{formatClock(slot.end_time)}
                </button>
              );
            })}
          </div>
        </fieldset>

        {submitError && <p role="alert" className="mt-3 text-sm text-red-600">{submitError}</p>}

        <button
          type="submit"
          disabled={!selected || mutation.isLoading}
          className="mt-4 flex items-center gap-2 rounded bg-primary px-3 py-1 text-white disabled:opacity-60"
        >
          {mutation.isLoading ? (<><Spinner size={16} /><span>Menyimpan...</span></>) : 'Simpan jadwal'}
        </button>
      </form>
    </Modal>
  );
}
