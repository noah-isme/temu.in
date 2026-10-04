import Spinner from './Spinner';
import { getApiErrorMessage } from '../api';

export function LoadingState({ label }: { label: string }): JSX.Element {
  return (
    <p role="status" className="flex items-center gap-2 text-sm text-slate-700">
      <Spinner size={16} /> {label}
    </p>
  );
}

export function ErrorState({ title, error, onRetry, retrying }: { title: string; error: unknown; onRetry: () => void; retrying?: boolean }): JSX.Element {
  return (
    <div role="alert" className="rounded border border-red-200 bg-red-50 p-4 text-sm text-red-700">
      <p>{title}: {getApiErrorMessage(error, 'terjadi kesalahan')}</p>
      <button type="button" onClick={onRetry} disabled={retrying} className="mt-2 rounded border border-red-300 px-3 py-1">
        Coba lagi
      </button>
    </div>
  );
}
