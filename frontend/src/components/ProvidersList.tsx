import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { fetchProviders, Provider } from '../api';
import Spinner from './Spinner';

export default function ProvidersList(): JSX.Element {
  const { data: providers = [], isLoading, isError } = useQuery(['providers'], fetchProviders);
  const error = isError ? 'Failed to load providers' : null;

  return (
    <section>
      <h2 className="text-2xl font-semibold mb-4">Providers</h2>
      {isLoading && (
        <div className="mb-4 flex items-center gap-2">
          <Spinner />
          <p>Loading providers...</p>
        </div>
      )}
      {error && <p className="text-sm text-red-600">{error}</p>}
      <ul className="grid gap-4 sm:grid-cols-2">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => (
              <li key={i} className="rounded border p-4 animate-pulse">
                <div className="h-4 w-32 bg-slate-200 mb-2 rounded" />
                <div className="h-3 w-24 bg-slate-200 rounded" />
              </li>
            ))
          : providers.map((p: Provider) => (
              <li key={p.id} className="rounded border p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium">{p.name}</p>
                    <p className="text-sm text-muted-foreground">{p.available ? 'Available' : 'Unavailable'}</p>
                  </div>
                  <div>
                    <Link
                      to={`/booking?providerId=${encodeURIComponent(p.id)}`}
                      className="rounded bg-primary px-3 py-1 text-white disabled:opacity-50"
                      aria-disabled={!p.available}
                      onClick={(event) => { if (!p.available) event.preventDefault(); }}
                    >
                      Book
                    </Link>
                  </div>
                </div>
              </li>
            ))}
      </ul>
    </section>
  );
}
