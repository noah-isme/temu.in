import PageTransition from '../components/PageTransition';

export default function AdminPage() {
  return (
    <PageTransition>
      <section className="mx-auto max-w-5xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Admin Dashboard</h2>
        <p className="text-sm text-muted-foreground">Placeholder untuk halaman admin — statistik, manajemen bookings, dsb.</p>
      </section>
    </PageTransition>
  );
}
