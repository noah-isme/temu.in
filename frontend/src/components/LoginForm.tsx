import { useState, useEffect } from 'react';
import axios from 'axios';
import { login, setAuthToken, getAuthToken } from '../api';
import { useAppStore } from '../store/app-store';
import toast from '../lib/toast';

export default function LoginForm({ onLogin }: { onLogin?: (token: string) => void }) {
  const [email, setEmail] = useState('demo@user.test');
  const [password, setPassword] = useState('password');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    setToken(getAuthToken());
  }, []);

  const initialize = useAppStore((s) => s.initialize);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const data = await login(email, password);
      setAuthToken(data.token);
      setToken(data.token);
  if (onLogin) onLogin(data.token);
    // refresh user state
      await initialize();
      toast.success('Login successful');
    } catch (err: unknown) {
      const responseData = axios.isAxiosError(err) ? err.response?.data as { message?: string; error?: string } : undefined;
      const msg = responseData?.message ?? responseData?.error ?? 'Login failed';
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="max-w-sm">
      <h2 className="text-xl font-semibold mb-4">Login</h2>
      <div className="mb-2">
        <label className="block text-sm">Email</label>
        <input value={email} onChange={(e) => setEmail(e.target.value)} className="w-full rounded border p-2" />
      </div>
      <div className="mb-2">
        <label className="block text-sm">Password</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full rounded border p-2" />
      </div>
      <button className="rounded bg-primary px-3 py-1 text-white flex items-center gap-2" disabled={loading} type="submit">
        {loading ? (
          <>
            <svg className="animate-spin" viewBox="0 0 24 24" width="16" height="16"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg>
            <span>Logging in...</span>
          </>
        ) : (
          'Login'
        )}
      </button>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
      {token && <p className="mt-2 text-sm text-green-600">Logged in</p>}
    </form>
  );
}
