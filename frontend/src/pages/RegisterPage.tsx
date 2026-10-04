import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link, useNavigate } from 'react-router-dom';
import { getApiErrorMessage, register as registerAccount, setAuthToken } from '../api';
import { useAppStore } from '../store/app-store';
import PageTransition from '../components/PageTransition';
import Spinner from '../components/Spinner';
import * as toast from '../lib/toast';
import { t } from '../lib/i18n';

// Mirrors the backend `registerReq` binding: email required + valid, password min 6.
const schema = z.object({
  name: z.string().trim().max(255, 'Nama terlalu panjang'),
  email: z.string().trim().min(1, 'Email wajib diisi').email('Format email tidak valid'),
  password: z.string().min(6, 'Password minimal 6 karakter'),
});

type FormValues = z.infer<typeof schema>;

export default function RegisterPage(): JSX.Element {
  const navigate = useNavigate();
  const setUser = useAppStore((s) => s.setUser);
  const [serverError, setServerError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { name: '', email: '', password: '' } });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      const data = await registerAccount(values.email, values.password, values.name);
      setAuthToken(data.token);
      setUser({ ...data.user, name: values.name || undefined });
      toast.success(t('register_success'), '');
      navigate('/', { replace: true });
    } catch (err) {
      const msg = getApiErrorMessage(err, t('register_failed'));
      setServerError(msg);
      toast.error(msg, t('register_failed'));
    }
  }

  return (
    <PageTransition>
      <section className="mx-auto max-w-md px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Daftar</h2>
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="max-w-sm">
          <div className="mb-2">
            <label htmlFor="register-name" className="block text-sm">Nama</label>
            <input id="register-name" autoComplete="name" {...register('name')} className="w-full rounded border p-2" aria-invalid={!!errors.name} aria-describedby={errors.name ? 'register-name-error' : undefined} />
            {errors.name && <p id="register-name-error" className="mt-1 text-sm text-red-600">{errors.name.message}</p>}
          </div>
          <div className="mb-2">
            <label htmlFor="register-email" className="block text-sm">Email</label>
            <input id="register-email" type="email" autoComplete="email" {...register('email')} className="w-full rounded border p-2" aria-invalid={!!errors.email} aria-describedby={errors.email ? 'register-email-error' : undefined} />
            {errors.email && <p id="register-email-error" className="mt-1 text-sm text-red-600">{errors.email.message}</p>}
          </div>
          <div className="mb-2">
            <label htmlFor="register-password" className="block text-sm">Password</label>
            <input id="register-password" type="password" autoComplete="new-password" {...register('password')} className="w-full rounded border p-2" aria-invalid={!!errors.password} aria-describedby={errors.password ? 'register-password-error' : undefined} />
            {errors.password && <p id="register-password-error" className="mt-1 text-sm text-red-600">{errors.password.message}</p>}
          </div>
          <button type="submit" disabled={isSubmitting} className="flex items-center gap-2 rounded bg-primary px-3 py-1 text-white disabled:opacity-60">
            {isSubmitting ? (<><Spinner size={16} /><span>Mendaftar...</span></>) : 'Daftar'}
          </button>
          {serverError && <p role="alert" className="mt-2 text-sm text-red-600">{serverError}</p>}
        </form>
        <p className="mt-4 text-sm text-slate-600">
          Sudah punya akun? <Link to="/login" className="text-primary underline">Masuk</Link>
        </p>
      </section>
    </PageTransition>
  );
}
