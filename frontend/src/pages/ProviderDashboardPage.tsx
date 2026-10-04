import { useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { Link } from 'react-router-dom';
import PageTransition from '../components/PageTransition';
import { ErrorState, LoadingState } from '../components/QueryFeedback';
import ScheduleManager from '../components/provider/ScheduleManager';
import ExceptionManager from '../components/provider/ExceptionManager';
import ProviderBookingsTable from '../components/provider/ProviderBookingsTable';
import { useMyProvider } from '../hooks/useMyProvider';

const TABS = [
  { id: 'schedules', label: 'Jadwal' },
  { id: 'exceptions', label: 'Pengecualian' },
  { id: 'bookings', label: 'Booking' },
] as const;

type TabId = (typeof TABS)[number]['id'];

export default function ProviderDashboardPage(): JSX.Element {
  // The route sits behind RequireAuth, so fetch regardless of whether the
  // user profile in the store has loaded yet.
  const { data: provider, isLoading, isError, error, refetch, isFetching } = useMyProvider(true);
  const [tab, setTab] = useState<TabId>('schedules');
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);

  function onTabKeyDown(e: KeyboardEvent<HTMLButtonElement>, index: number) {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return;
    e.preventDefault();
    const next = (index + (e.key === 'ArrowRight' ? 1 : TABS.length - 1)) % TABS.length;
    setTab(TABS[next].id);
    tabRefs.current[next]?.focus();
  }

  return (
    <PageTransition>
      <section>
        <h2 className="mb-2 text-2xl font-semibold">Dashboard provider</h2>

        {isLoading && <LoadingState label="Memuat profil provider..." />}
        {isError && <ErrorState title="Gagal memuat profil provider" error={error} onRetry={() => refetch()} retrying={isFetching} />}

        {provider === null && (
          <div className="rounded border bg-white p-6 text-sm text-slate-600">
            <p className="mb-2">Anda belum mendaftar sebagai provider.</p>
            <Link to="/providers" className="text-primary underline">Lihat daftar provider</Link>
          </div>
        )}

        {provider && (
          <>
            <p className="mb-6 text-slate-600">{provider.business_name}</p>
            <div role="tablist" aria-label="Bagian dashboard" className="mb-4 flex gap-2 border-b">
              {TABS.map((item, i) => (
                <button
                  key={item.id}
                  ref={(el) => {
                    tabRefs.current[i] = el;
                  }}
                  id={`dashboard-tab-${item.id}`}
                  type="button"
                  role="tab"
                  aria-selected={tab === item.id}
                  aria-controls={`dashboard-panel-${item.id}`}
                  tabIndex={tab === item.id ? 0 : -1}
                  onClick={() => setTab(item.id)}
                  onKeyDown={(e) => onTabKeyDown(e, i)}
                  className={`-mb-px border-b-2 px-3 py-2 text-sm font-medium ${tab === item.id ? 'border-primary text-primary' : 'border-transparent text-slate-600'}`}
                >
                  {item.label}
                </button>
              ))}
            </div>
            <div role="tabpanel" id={`dashboard-panel-${tab}`} aria-labelledby={`dashboard-tab-${tab}`}>
              {tab === 'schedules' && <ScheduleManager providerId={provider.id} />}
              {tab === 'exceptions' && <ExceptionManager providerId={provider.id} />}
              {tab === 'bookings' && <ProviderBookingsTable providerId={provider.id} />}
            </div>
          </>
        )}
      </section>
    </PageTransition>
  );
}
