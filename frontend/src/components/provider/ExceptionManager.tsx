import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createException, deleteException, fetchExceptions, getApiErrorMessage, updateException } from '../../api';
import type { AvailabilityException, ExceptionInput } from '../../api';
import { ErrorState, LoadingState } from '../QueryFeedback';
import { formatClock, formatDate, localISODate } from '../../lib/datetime';
import * as toast from '../../lib/toast';
import { t } from '../../lib/i18n';

type Draft = { date: string; start_time: string; end_time: string; is_available: boolean; reason: string };

function emptyDraft(): Draft {
  return { date: localISODate(), start_time: '', end_time: '', is_available: false, reason: '' };
}

function toDraft(e: AvailabilityException): Draft {
  return {
    date: e.date,
    start_time: e.start_time ? formatClock(e.start_time) : '',
    end_time: e.end_time ? formatClock(e.end_time) : '',
    is_available: e.is_available,
    reason: e.reason,
  };
}

/** Validates a draft; returns the API payload or an error message. Empty times mean a full-day exception. */
function toInput(d: Draft): ExceptionInput | string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(d.date)) return 'Tanggal wajib diisi';
  const start = d.start_time ? formatClock(d.start_time) : '';
  const end = d.end_time ? formatClock(d.end_time) : '';
  if (!!start !== !!end) return 'Isi jam mulai dan jam selesai, atau kosongkan keduanya untuk sepanjang hari';
  if (start && start >= end) return 'Jam selesai harus setelah jam mulai';
  const input: ExceptionInput = { date: d.date, is_available: d.is_available, reason: d.reason.trim() };
  if (start) {
    input.start_time = start;
    input.end_time = end;
  }
  return input;
}

function currentMonthRange(): { start: string; end: string } {
  const now = new Date();
  return {
    start: localISODate(new Date(now.getFullYear(), now.getMonth(), 1)),
    end: localISODate(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
  };
}

export default function ExceptionManager({ providerId }: { providerId: string }): JSX.Element {
  const queryClient = useQueryClient();
  const [range, setRange] = useState(currentMonthRange);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  const baseKey = ['provider', providerId, 'exceptions'];
  const { data, isLoading, isError, error, refetch, isFetching } = useQuery(
    [...baseKey, range.start, range.end],
    () => fetchExceptions(providerId, range.start, range.end),
    { enabled: !!range.start && !!range.end }
  );

  function resetForm() {
    setDraft(emptyDraft());
    setEditingId(null);
    setFormError(null);
  }

  const onError = (err: unknown) => {
    const msg = getApiErrorMessage(err, t('save_failed'));
    setFormError(msg);
    toast.error(msg, t('save_failed'));
  };

  const saveMutation = useMutation(
    ({ id, input }: { id: string | null; input: ExceptionInput }) =>
      id ? updateException(providerId, id, input) : createException(providerId, input),
    {
      onSuccess: () => {
        resetForm();
        toast.success(t('exception_saved'), '');
        queryClient.invalidateQueries(baseKey);
      },
      onError,
    }
  );

  const deleteMutation = useMutation((id: string) => deleteException(providerId, id), {
    onSuccess: () => {
      toast.success(t('exception_deleted'), '');
      queryClient.invalidateQueries(baseKey);
    },
    onError: (err: unknown) => toast.error(getApiErrorMessage(err, t('save_failed')), t('save_failed')),
  });

  const exceptions = [...(data ?? [])].sort((a, b) => a.date.localeCompare(b.date));

  return (
    <div>
      <fieldset className="mb-4 flex flex-wrap items-end gap-3">
        <legend className="mb-1 text-sm font-medium">Rentang tanggal</legend>
        <div>
          <label htmlFor="exceptions-from" className="block text-xs text-slate-600">Dari</label>
          <input id="exceptions-from" type="date" value={range.start} onChange={(e) => setRange((r) => ({ ...r, start: e.target.value }))} className="rounded border p-1 text-sm" />
        </div>
        <div>
          <label htmlFor="exceptions-to" className="block text-xs text-slate-600">Sampai</label>
          <input id="exceptions-to" type="date" value={range.end} onChange={(e) => setRange((r) => ({ ...r, end: e.target.value }))} className="rounded border p-1 text-sm" />
        </div>
      </fieldset>

      {isLoading && <LoadingState label="Memuat pengecualian..." />}
      {isError && <ErrorState title="Gagal memuat pengecualian" error={error} onRetry={() => refetch()} retrying={isFetching} />}
      {data && exceptions.length === 0 && <p className="text-sm text-slate-600">Tidak ada pengecualian pada rentang ini.</p>}

      {exceptions.length > 0 && (
        <table className="w-full text-left text-sm" aria-label="Pengecualian jadwal">
          <thead>
            <tr className="border-b text-slate-500">
              <th className="py-2">Tanggal</th>
              <th>Waktu</th>
              <th>Status</th>
              <th>Alasan</th>
              <th><span className="sr-only">Aksi</span></th>
            </tr>
          </thead>
          <tbody>
            {exceptions.map((e) => (
              <tr key={e.id} className="border-b">
                <td className="py-2">{formatDate(e.date)}</td>
                <td>{e.start_time && e.end_time ? `${formatClock(e.start_time)}–${formatClock(e.end_time)}` : 'Sepanjang hari'}</td>
                <td>{e.is_available ? 'Buka' : 'Tutup'}</td>
                <td>{e.reason || '-'}</td>
                <td className="flex justify-end gap-2 py-2">
                  <button
                    type="button"
                    className="rounded border px-2 py-0.5"
                    aria-label={`Ubah pengecualian ${e.date}`}
                    onClick={() => {
                      setDraft(toDraft(e));
                      setEditingId(e.id);
                      setFormError(null);
                    }}
                  >
                    Ubah
                  </button>
                  <button
                    type="button"
                    className="rounded border border-red-200 px-2 py-0.5 text-red-600 disabled:opacity-50"
                    aria-label={`Hapus pengecualian ${e.date}`}
                    disabled={deleteMutation.isLoading}
                    onClick={() => {
                      if (window.confirm(`Hapus pengecualian ${formatDate(e.date)}?`)) deleteMutation.mutate(e.id);
                    }}
                  >
                    Hapus
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <form
        aria-label={editingId ? 'Ubah pengecualian' : 'Tambah pengecualian'}
        className="mt-6 grid gap-3 rounded border bg-white p-4 sm:grid-cols-2"
        onSubmit={(ev) => {
          ev.preventDefault();
          const input = toInput(draft);
          if (typeof input === 'string') {
            setFormError(input);
            return;
          }
          setFormError(null);
          saveMutation.mutate({ id: editingId, input });
        }}
      >
        <h3 className="font-medium sm:col-span-2">{editingId ? 'Ubah pengecualian' : 'Tambah pengecualian'}</h3>
        <div>
          <label htmlFor="exception-date" className="block text-xs text-slate-600">Tanggal</label>
          <input id="exception-date" type="date" value={draft.date} onChange={(e) => setDraft({ ...draft, date: e.target.value })} className="rounded border p-1 text-sm" />
        </div>
        <div className="flex items-center gap-1 self-end pb-1">
          <input id="exception-available" type="checkbox" checked={draft.is_available} onChange={(e) => setDraft({ ...draft, is_available: e.target.checked })} />
          <label htmlFor="exception-available" className="text-sm">Buka (tambahan jam tersedia)</label>
        </div>
        <div>
          <label htmlFor="exception-start" className="block text-xs text-slate-600">Jam mulai (opsional)</label>
          <input id="exception-start" type="time" value={draft.start_time} onChange={(e) => setDraft({ ...draft, start_time: e.target.value })} className="rounded border p-1 text-sm" />
        </div>
        <div>
          <label htmlFor="exception-end" className="block text-xs text-slate-600">Jam selesai (opsional)</label>
          <input id="exception-end" type="time" value={draft.end_time} onChange={(e) => setDraft({ ...draft, end_time: e.target.value })} className="rounded border p-1 text-sm" />
        </div>
        <div className="sm:col-span-2">
          <label htmlFor="exception-reason" className="block text-xs text-slate-600">Alasan</label>
          <input id="exception-reason" value={draft.reason} onChange={(e) => setDraft({ ...draft, reason: e.target.value })} className="w-full rounded border p-1 text-sm" />
        </div>
        {formError && <p role="alert" className="text-sm text-red-600 sm:col-span-2">{formError}</p>}
        <div className="flex gap-2 sm:col-span-2">
          <button type="submit" disabled={saveMutation.isLoading} className="rounded bg-primary px-3 py-1 text-white disabled:opacity-60">
            {saveMutation.isLoading ? 'Menyimpan...' : editingId ? 'Simpan perubahan' : 'Tambah pengecualian'}
          </button>
          {editingId && <button type="button" onClick={resetForm} className="rounded border px-3 py-1">Batal</button>}
        </div>
      </form>
    </div>
  );
}
