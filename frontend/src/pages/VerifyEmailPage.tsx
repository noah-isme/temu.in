import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { getApiErrorMessage, verifyEmail } from '../api';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

type State =
  | { kind: 'loading' }
  | { kind: 'success' }
  | { kind: 'error'; message: string };

export default function VerifyEmailPage(): JSX.Element {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const [state, setState] = useState<State>(() =>
    token ? { kind: 'loading' } : { kind: 'error', message: 'Token verifikasi tidak ditemukan.' }
  );
  // Guard against StrictMode's double effect run submitting a one-time token
  // twice. No cleanup flag: the first run's request must still update state.
  const submitted = useRef<string | null>(null);

  useEffect(() => {
    if (!token || submitted.current === token) return;
    submitted.current = token;
    verifyEmail(token)
      .then(() => {
        setState({ kind: 'success' });
        toast.success(t('email_verified'), '');
      })
      .catch((err: unknown) => {
        const message = getApiErrorMessage(err, t('email_verify_failed'));
        setState({ kind: 'error', message });
        toast.error(message, t('email_verify_failed'));
      });
  }, [token]);

  return (
    <PageTransition>
      <section className="mx-auto max-w-md px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Verifikasi email</h2>
        {state.kind === 'loading' && (
          <p role="status" className="flex items-center gap-2 text-sm text-slate-700"><Spinner size={16} /> Memverifikasi email...</p>
        )}
        {state.kind === 'success' && (
          <p role="status" className="text-sm text-green-700">{t('email_verified')}</p>
        )}
        {state.kind === 'error' && (
          <div role="alert" className="text-sm text-red-600">
            <p>{t('email_verify_failed')}: {state.message}</p>
          </div>
        )}
        {state.kind !== 'loading' && (
          <p className="mt-4 text-sm">
            <Link to="/login" className="text-primary underline">Ke halaman masuk</Link>
          </p>
        )}
      </section>
    </PageTransition>
  );
}
