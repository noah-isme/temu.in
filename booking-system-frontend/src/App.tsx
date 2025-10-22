
import { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { useAppStore } from './store/app-store';
import PageTransition from './components/PageTransition';
import HomePage from './pages/HomePage';
import ProvidersPage from './pages/ProvidersPage';
import BookingPage from './pages/BookingPage';
import ServicesPage from './pages/ServicesPage';
import AdminPage from './pages/AdminPage';
import LoginPage from './pages/LoginPage';
import RequireAuth from './components/RequireAuth';
import { FEATURES, Feature } from './shared/features';

function AppNavLink({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `rounded-md px-3 py-2 text-sm font-medium transition ${isActive ? 'bg-primary text-white' : 'text-slate-700 hover:bg-slate-100'}`
      }
    >
      {label}
    </NavLink>
  );
}

export default function App(): JSX.Element {
  const initialize = useAppStore((state) => state.initialize);
  const user = useAppStore((s) => s.user);
  const logout = useAppStore((s) => s.logout);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    initialize();
  }, [initialize]);

  return (
    <BrowserRouter>
      <main className="min-h-screen bg-gradient-to-b from-white to-slate-50">
        <header className="border-b bg-white">
          <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-6 py-4">
            <div className="flex items-center gap-4">
              <h3 className="text-lg font-bold">Temu.in</h3>
              <nav className="hidden items-center gap-2 md:flex">
                  <AppNavLink to="/" label="Home" />
                  <AppNavLink to="/providers" label="Providers" />
                  <AppNavLink to="/booking" label="Booking" />
                  <AppNavLink to="/services" label="Services" />
                  {user?.role === 'admin' && <AppNavLink to="/admin" label="Admin" />}
                </nav>
            </div>

            <div className="flex items-center gap-2">
                {user ? (
                  <div className="hidden items-center gap-3 md:flex">
                    <span className="text-sm text-slate-700">{user.name ?? user.email}</span>
                    <button onClick={() => logout()} className="rounded bg-red-50 px-3 py-1 text-sm text-red-600">Logout</button>
                  </div>
                ) : (
                  <div className="hidden md:block">
                    <AppNavLink to="/login" label="Login" />
                  </div>
                )}
              <button
                aria-label="Toggle menu"
                onClick={() => setOpen((v) => !v)}
                className="inline-flex items-center rounded-md p-2 text-slate-700 hover:bg-slate-100 md:hidden"
              >
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              </button>
            </div>
          </div>

          {open && (
            <div className="md:hidden border-t bg-white">
              <div className="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-3">
                <NavLink to="/" onClick={() => setOpen(false)} className="py-2">Home</NavLink>
                <NavLink to="/providers" onClick={() => setOpen(false)} className="py-2">Providers</NavLink>
                <NavLink to="/booking" onClick={() => setOpen(false)} className="py-2">Booking</NavLink>
                <NavLink to="/services" onClick={() => setOpen(false)} className="py-2">Services</NavLink>
                {user?.role === 'admin' && <NavLink to="/admin" onClick={() => setOpen(false)} className="py-2">Admin</NavLink>}
                <NavLink to="/login" onClick={() => setOpen(false)} className="py-2">Login</NavLink>
              </div>
            </div>
          )}
        </header>

        <PageTransition>
          <div className="mx-auto max-w-6xl px-6 py-12">
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/providers" element={<ProvidersPage />} />
              <Route path="/booking" element={<BookingPage />} />
              <Route path="/services" element={<ServicesPage />} />
              <Route path="/admin" element={<RequireAuth><AdminPage /></RequireAuth>} />
              <Route path="/login" element={<LoginPage />} />
            </Routes>

            <section className="mt-12 grid gap-6 sm:grid-cols-2">
              {FEATURES.map((feature: Feature) => (
                <article key={feature.title} className="rounded-xl border border-border bg-card p-6 shadow-sm transition hover:shadow-md">
                  <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
                    <feature.icon className="h-6 w-6" aria-hidden />
                  </div>
                  <h3 className="mb-2 text-lg font-semibold text-card-foreground">{feature.title}</h3>
                  <p className="text-sm text-muted-foreground">{feature.description}</p>
                </article>
              ))}
            </section>
          </div>
        </PageTransition>
      </main>
    </BrowserRouter>
  );
}
