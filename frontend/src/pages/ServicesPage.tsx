import { useEffect, useState } from 'react';
import PageTransition from '../components/PageTransition';
import { fetchServices } from '../api';
import Spinner from '../components/Spinner';

export default function ServicesPage() {
  const [services, setServices] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetchServices().then((s) => setServices(s)).catch(() => {}).finally(() => setLoading(false));
  }, []);

  return (
    <PageTransition>
      <section className="mx-auto max-w-4xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Layanan</h2>
        {loading ? (
          <div className="flex items-center justify-center">
            <Spinner />
          </div>
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2">
            {services.map((s) => (
              <li key={s.id} className="rounded-md border p-4">
                <h3 className="font-medium">{s.name}</h3>
                <p className="text-sm text-muted-foreground">{s.description}</p>
                <div className="mt-2 text-sm font-semibold">Rp {s.price}</div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </PageTransition>
  );
}
