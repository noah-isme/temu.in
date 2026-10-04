import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { getApiErrorMessage, resetPassword } from '../api';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

// Mirrors the backend `resetPasswordReq` binding: password min 6.
const schema = z
  .object({
    password: z.string().min(6, 'Password minimal 6 karakter'),
    confirm: z.string(),
  })
  .refine((v) => v.password === v.confirm, { message: 'Konfirmasi password tidak sama', path: ['confirm'] });

type FormValues = z.infer<typeof schema>;

export default function ResetPasswordPage(): JSX.Element {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const navigate = useNavigate();
  const [serverError, setServerError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { password: '', confirm: '' } });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      await resetPassword(token, values.password);
      toast.success(t('password_reset_success'), '');
      navigate('/login', { replace: true });
    } catch (err) {
      const msg = getApiErrorMessage(err, t('password_reset_failed'));
      setServerError(msg);
      toast.error(msg, t('password_reset_failed'));
    }
  }

  return (
    <PageTransition>
      <section className="mx-auto max-w-md px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Reset password</h2>
        {!token ? (
          <div role="alert" className="text-sm text-red-600">
            <p>Tautan reset tidak valid: token tidak ditemukan.</p>
            <Link to="/forgot-password" className="text-primary underline">Minta tautan baru</Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit(onSubmit)} noValidate className="max-w-sm">
            <div className="mb-2">
              <label htmlFor="reset-password" className="block text-sm">Password baru</label>
              <input id="reset-password" type="password" autoComplete="new-password" {...register('password')} className="w-full rounded border p-2" aria-invalid={!!errors.password} aria-describedby={errors.password ? 'reset-password-error' : undefined} />
              {errors.password && <p id="reset-password-error" className="mt-1 text-sm text-red-600">{errors.password.message}</p>}
            </div>
            <div className="mb-2">
              <label htmlFor="reset-confirm" className="block text-sm">Ulangi password</label>
              <input id="reset-confirm" type="password" autoComplete="new-password" {...register('confirm')} className="w-full rounded border p-2" aria-invalid={!!errors.confirm} aria-describedby={errors.confirm ? 'reset-confirm-error' : undefined} />
              {errors.confirm && <p id="reset-confirm-error" className="mt-1 text-sm text-red-600">{errors.confirm.message}</p>}
            </div>
            <button type="submit" disabled={isSubmitting} className="flex items-center gap-2 rounded bg-primary px-3 py-1 text-white disabled:opacity-60">
              {isSubmitting ? (<><Spinner size={16} /><span>Menyimpan...</span></>) : 'Simpan password'}
            </button>
            {serverError && <p role="alert" className="mt-2 text-sm text-red-600">{serverError}</p>}
          </form>
        )}
      </section>
    </PageTransition>
  );
}
