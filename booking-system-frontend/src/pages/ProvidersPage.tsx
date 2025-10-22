import ProvidersList from '../components/ProvidersList';
import PageTransition from '../components/PageTransition';

export default function ProvidersPage() {
  return (
    <PageTransition>
      <section className="mx-auto max-w-5xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Providers</h2>
        <ProvidersList />
      </section>
    </PageTransition>
  );
}
