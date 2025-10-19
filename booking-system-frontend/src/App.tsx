
import { useEffect } from 'react';
import { Calendar, Clock, Mail, ShieldCheck } from 'lucide-react';
import { useAppStore } from './store/app-store';

const FEATURES = [
  {
    icon: Calendar,
    title: 'Smart Scheduling',
    description: 'Kelola ketersediaan provider dan booking pelanggan dalam satu kalender terpadu.'
  },
  {
    icon: Clock,
    title: 'Real-time Availability',
    description: 'Slot booking otomatis ter-update untuk mencegah double booking dan konflik jadwal.'
  },
  {
    icon: Mail,
    title: 'Automated Notifications',
    description: 'Kirim email konfirmasi, pengingat, dan reschedule menggunakan SendGrid.'
  },
  {
    icon: ShieldCheck,
    title: 'Secure Payments',
    description: 'Integrasi Midtrans Snap memastikan transaksi aman dan mudah.'
  }
];

function FeatureCard({ icon: Icon, title, description }: (typeof FEATURES)[number]) {
  return (
    <article className="rounded-xl border border-border bg-card p-6 shadow-sm transition hover:shadow-md">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Icon className="h-6 w-6" aria-hidden />
      </div>
      <h3 className="mb-2 text-lg font-semibold text-card-foreground">{title}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </article>
  );
}

export default function App(): JSX.Element {
  const initialize = useAppStore((state) => state.initialize);

  useEffect(() => {
    initialize();
  }, [initialize]);

  return (
    <main className="min-h-screen bg-gradient-to-b from-white to-slate-50">
      <section className="mx-auto flex max-w-5xl flex-col gap-12 px-6 py-16 md:py-24">
        <header className="text-center">
          <p className="text-sm font-medium uppercase tracking-[0.3em] text-primary">Temu.in Platform</p>
          <h1 className="mt-4 text-4xl font-bold text-slate-900 sm:text-5xl">
            Inisialisasi Monorepo Booking System
          </h1>
          <p className="mt-4 text-lg text-slate-600">
            Fondasi frontend React + TypeScript siap dikembangkan dengan Tailwind, Zustand, dan shadcn/ui.
          </p>
        </header>

        <section className="grid gap-6 sm:grid-cols-2">
          {FEATURES.map((feature) => (
            <FeatureCard key={feature.title} {...feature} />
          ))}
        </section>
      </section>
    </main>
  );
}
