import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createSchedule, deleteSchedule, fetchSchedules, getApiErrorMessage, updateSchedule } from '../../api';
import type { AvailabilitySchedule, ScheduleInput } from '../../api';
import { ErrorState, LoadingState } from '../QueryFeedback';
import { formatClock } from '../../lib/datetime';
import * as toast from '../../lib/toast';
import { t } from '../../lib/i18n';

const DAY_NAMES = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];

type Draft = { day_of_week: string; start_time: string; end_time: string; is_available: boolean };

const EMPTY_DRAFT: Draft = { day_of_week: '1', start_time: '09:00', end_time: '17:00', is_available: true };

function toDraft(s: AvailabilitySchedule): Draft {
  // DTO times are HH:MM already; formatClock also tolerates HH:MM:SS so the
  // round-trip always satisfies the backend `len=5` binding.
  return { day_of_week: String(s.day_of_week), start_time: formatClock(s.start_time), end_time: formatClock(s.end_time), is_available: s.is_available };
}

/** Validates a draft; returns the API payload or an error message. */
function toInput(d: Draft): ScheduleInput | string {
  const start = formatClock(d.start_time);
  const end = formatClock(d.end_time);
  if (!/^\d{2}:\d{2}$/.test(start) || !/^\d{2}:\d{2}$/.test(end)) return 'Jam mulai dan jam selesai wajib diisi';
  if (start >= end) return 'Jam selesai harus setelah jam mulai';
  return { day_of_week: Number(d.day_of_week), start_time: start, end_time: end, is_available: d.is_available };
}

type FieldsProps = { idPrefix: string; draft: Draft; onChange: (d: Draft) => void };

function ScheduleFields({ idPrefix, draft, onChange }: FieldsProps): JSX.Element {
  return (
    <>
      <div>
        <label htmlFor={`${idPrefix}-day`} className="block text-xs text-slate-600">Hari</label>
        <select id={`${idPrefix}-day`} value={draft.day_of_week} onChange={(e) => onChange({ ...draft, day_of_week: e.target.value })} className="rounded border p-1 text-sm">
          {DAY_NAMES.map((name, i) => <option key={name} value={String(i)}>{name}</option>)}
        </select>
      </div>
      <div>
        <label htmlFor={`${idPrefix}-start`} className="block text-xs text-slate-600">Jam mulai</label>
        <input id={`${idPrefix}-start`} type="time" value={draft.start_time} onChange={(e) => onChange({ ...draft, start_time: e.target.value })} className="rounded border p-1 text-sm" />
      </div>
      <div>
        <label htmlFor={`${idPrefix}-end`} className="block text-xs text-slate-600">Jam selesai</label>
        <input id={`${idPrefix}-end`} type="time" value={draft.end_time} onChange={(e) => onChange({ ...draft, end_time: e.target.value })} className="rounded border p-1 text-sm" />
      </div>
      <div className="flex items-center gap-1 self-end pb-1">
        <input id={`${idPrefix}-available`} type="checkbox" checked={draft.is_available} onChange={(e) => onChange({ ...draft, is_available: e.target.checked })} />
        <label htmlFor={`${idPrefix}-available`} className="text-sm">Tersedia</label>
      </div>
    </>
  );
}

export default function ScheduleManager({ providerId }: { providerId: string }): JSX.Element {
  const queryClient = useQueryClient();
  const queryKey = ['provider', providerId, 'schedules'];
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [formError, setFormError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState<Draft>(EMPTY_DRAFT);
  const [editError, setEditError] = useState<string | null>(null);

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery(queryKey, () => fetchSchedules(providerId));

  const onMutationError = (setter: (m: string) => void) => (err: unknown) => {
    const msg = getApiErrorMessage(err, t('save_failed'));
    setter(msg);
    toast.error(msg, t('save_failed'));
  };

  const createMutation = useMutation((input: ScheduleInput) => createSchedule(providerId, input), {
    onSuccess: () => {
      setDraft(EMPTY_DRAFT);
      toast.success(t('schedule_saved'), '');
      queryClient.invalidateQueries(queryKey);
    },
    onError: onMutationError(setFormError),
  });

  const updateMutation = useMutation(({ id, input }: { id: string; input: ScheduleInput }) => updateSchedule(providerId, id, input), {
    onSuccess: () => {
      setEditingId(null);
      toast.success(t('schedule_saved'), '');
      queryClient.invalidateQueries(queryKey);
    },
    onError: onMutationError(setEditError),
  });

  const deleteMutation = useMutation((id: string) => deleteSchedule(providerId, id), {
    onSuccess: () => {
      toast.success(t('schedule_deleted'), '');
      queryClient.invalidateQueries(queryKey);
    },
    onError: (err: unknown) => toast.error(getApiErrorMessage(err, t('save_failed')), t('save_failed')),
  });

  const schedules = [...(data ?? [])].sort((a, b) => a.day_of_week - b.day_of_week || a.start_time.localeCompare(b.start_time));

  return (
    <div>
      {isLoading && <LoadingState label="Memuat jadwal..." />}
      {isError && <ErrorState title="Gagal memuat jadwal" error={error} onRetry={() => refetch()} retrying={isFetching} />}

      {data && schedules.length === 0 && <p className="text-sm text-slate-600">Belum ada jadwal.</p>}

      {schedules.length > 0 && (
        <table className="w-full text-left text-sm" aria-label="Jadwal mingguan">
          <thead>
            <tr className="border-b text-slate-500">
              <th className="py-2">Hari</th>
              <th>Mulai</th>
              <th>Selesai</th>
              <th>Status</th>
              <th><span className="sr-only">Aksi</span></th>
            </tr>
          </thead>
          <tbody>
            {schedules.map((s) =>
              editingId === s.id ? (
                <tr key={s.id} className="border-b">
                  <td colSpan={5} className="py-2">
                    <form
                      aria-label={`Ubah jadwal ${DAY_NAMES[s.day_of_week] ?? s.day_of_week}`}
                      className="flex flex-wrap items-end gap-3"
                      onSubmit={(e) => {
                        e.preventDefault();
                        const input = toInput(editDraft);
                        if (typeof input === 'string') {
                          setEditError(input);
                          return;
                        }
                        setEditError(null);
                        updateMutation.mutate({ id: s.id, input });
                      }}
                    >
                      <ScheduleFields idPrefix={`schedule-edit-${s.id}`} draft={editDraft} onChange={setEditDraft} />
                      <button type="submit" disabled={updateMutation.isLoading} className="rounded bg-primary px-3 py-1 text-white disabled:opacity-60">Simpan</button>
                      <button type="button" onClick={() => setEditingId(null)} className="rounded border px-3 py-1">Batal</button>
                      {editError && <p role="alert" className="w-full text-sm text-red-600">{editError}</p>}
                    </form>
                  </td>
                </tr>
              ) : (
                <tr key={s.id} className="border-b">
                  <td className="py-2">{DAY_NAMES[s.day_of_week] ?? s.day_of_week}</td>
                  <td>{formatClock(s.start_time)}</td>
                  <td>{formatClock(s.end_time)}</td>
                  <td>{s.is_available ? 'Tersedia' : 'Tutup'}</td>
                  <td className="flex justify-end gap-2 py-2">
                    <button
                      type="button"
                      className="rounded border px-2 py-0.5"
                      aria-label={`Ubah jadwal ${DAY_NAMES[s.day_of_week]} ${formatClock(s.start_time)}`}
                      onClick={() => {
                        setEditDraft(toDraft(s));
                        setEditError(null);
                        setEditingId(s.id);
                      }}
                    >
                      Ubah
                    </button>
                    <button
                      type="button"
                      className="rounded border border-red-200 px-2 py-0.5 text-red-600 disabled:opacity-50"
                      aria-label={`Hapus jadwal ${DAY_NAMES[s.day_of_week]} ${formatClock(s.start_time)}`}
                      disabled={deleteMutation.isLoading}
                      onClick={() => {
                        if (window.confirm(`Hapus jadwal ${DAY_NAMES[s.day_of_week]} ${formatClock(s.start_time)}–${formatClock(s.end_time)}?`)) {
                          deleteMutation.mutate(s.id);
                        }
                      }}
                    >
                      Hapus
                    </button>
                  </td>
                </tr>
              )
            )}
          </tbody>
        </table>
      )}

      <form
        aria-label="Tambah jadwal"
        className="mt-6 flex flex-wrap items-end gap-3 rounded border bg-white p-4"
        onSubmit={(e) => {
          e.preventDefault();
          const input = toInput(draft);
          if (typeof input === 'string') {
            setFormError(input);
            return;
          }
          setFormError(null);
          createMutation.mutate(input);
        }}
      >
        <ScheduleFields idPrefix="schedule-new" draft={draft} onChange={setDraft} />
        <button type="submit" disabled={createMutation.isLoading} className="rounded bg-primary px-3 py-1 text-white disabled:opacity-60">
          {createMutation.isLoading ? 'Menyimpan...' : 'Tambah jadwal'}
        </button>
        {formError && <p role="alert" className="w-full text-sm text-red-600">{formError}</p>}
      </form>
    </div>
  );
}
