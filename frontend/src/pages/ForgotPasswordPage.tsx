import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link } from 'react-router-dom';
import { forgotPassword, getApiErrorMessage } from '../api';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

// Mirrors the backend `forgotPasswordReq` binding: email required + valid.
const schema = z.object({
  email: z.string().trim().min(1, 'Email wajib diisi').email('Format email tidak valid'),
});

type FormValues = z.infer<typeof schema>;

export default function ForgotPasswordPage(): JSX.Element {
  const [serverError, setServerError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { email: '' } });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      await forgotPassword(values.email);
      setSent(true);
      toast.success(t('reset_link_sent'), '');
    } catch (err) {
      const msg = getApiErrorMessage(err, t('password_reset_failed'));
      setServerError(msg);
      toast.error(msg, t('password_reset_failed'));
    }
  }

  return (
    <PageTransition>
      <section className="mx-auto max-w-md px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Lupa password</h2>
        {sent ? (
          <p role="status" className="text-sm text-green-700">{t('reset_link_sent')}</p>
        ) : (
          <form onSubmit={handleSubmit(onSubmit)} noValidate className="max-w-sm">
            <div className="mb-2">
              <label htmlFor="forgot-email" className="block text-sm">Email</label>
              <input id="forgot-email" type="email" autoComplete="email" {...register('email')} className="w-full rounded border p-2" aria-invalid={!!errors.email} aria-describedby={errors.email ? 'forgot-email-error' : undefined} />
              {errors.email && <p id="forgot-email-error" className="mt-1 text-sm text-red-600">{errors.email.message}</p>}
            </div>
            <button type="submit" disabled={isSubmitting} className="flex items-center gap-2 rounded bg-primary px-3 py-1 text-white disabled:opacity-60">
              {isSubmitting ? (<><Spinner size={16} /><span>Mengirim...</span></>) : 'Kirim tautan reset'}
            </button>
            {serverError && <p role="alert" className="mt-2 text-sm text-red-600">{serverError}</p>}
          </form>
        )}
        <p className="mt-4 text-sm text-slate-600">
          <Link to="/login" className="text-primary underline">Kembali ke halaman masuk</Link>
        </p>
      </section>
    </PageTransition>
  );
}
