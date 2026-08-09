import { motion } from 'framer-motion';
import { ShieldCheck, Zap, Users, ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';

const STATS = [
  { icon: Zap, value: '< 50ms', label: 'Kecepatan Sync Slot' },
  { icon: ShieldCheck, value: '100%', label: 'Aman & Terenkripsi' },
  { icon: Users, value: '10.000+', label: 'Booking Terproses' }
];

export default function StatsSection() {
  return (
    <section className="mt-20 space-y-12">
      {/* Stats Counter Strip */}
      <div className="grid gap-6 rounded-2xl border border-slate-200/80 bg-gradient-to-r from-slate-900 to-slate-800 p-8 text-white shadow-xl sm:grid-cols-3">
        {STATS.map((stat, idx) => {
          const IconComponent = stat.icon;
          return (
            <motion.div
              key={stat.label}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: idx * 0.1, duration: 0.4 }}
              className="flex items-center gap-4 border-b border-slate-700/50 pb-4 sm:border-b-0 sm:border-r sm:pb-0 sm:pr-4 last:border-none"
            >
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/10 text-primary-300 backdrop-blur-md">
                <IconComponent className="h-6 w-6 text-indigo-400" />
              </div>
              <div>
                <div className="text-2xl font-black tracking-tight text-white sm:text-3xl">{stat.value}</div>
                <div className="text-xs text-slate-400 font-medium">{stat.label}</div>
              </div>
            </motion.div>
          );
        })}
      </div>

      {/* CTA Banner */}
      <motion.div
        initial={{ opacity: 0, scale: 0.98 }}
        whileInView={{ opacity: 1, scale: 1 }}
        viewport={{ once: true }}
        className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary via-indigo-600 to-indigo-800 p-8 text-white shadow-2xl md:p-12"
      >
        <div className="pointer-events-none absolute -right-16 -top-16 h-72 w-72 rounded-full bg-white/10 blur-2xl" />

        <div className="relative z-10 flex flex-col items-start justify-between gap-6 md:flex-row md:items-center">
          <div>
            <div className="inline-flex items-center gap-2 rounded-full bg-white/10 px-3 py-1 text-xs font-semibold backdrop-blur-md">
              <span className="h-2 w-2 rounded-full bg-emerald-400 animate-ping" />
              <span>Siap Digunakan Sekarang</span>
            </div>
            <h3 className="mt-3 text-2xl font-black sm:text-3xl">Mulai Kelola Jadwal & Booking Sekarang</h3>
            <p className="mt-2 max-w-xl text-sm text-indigo-100">
              Jelajahi daftar penyedia layanan atau lakukan simulasi booking interaktif secara langsung.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Link to="/booking">
              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                className="group flex items-center gap-2 rounded-xl bg-white px-6 py-3 font-bold text-slate-900 shadow-lg hover:bg-slate-50 transition"
              >
                <span>Buka Form Booking</span>
                <ArrowUpRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
              </motion.button>
            </Link>

            <Link to="/providers">
              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                className="rounded-xl border border-white/30 bg-white/10 px-6 py-3 font-semibold text-white backdrop-blur-md hover:bg-white/20 transition"
              >
                Lihat Provider
              </motion.button>
            </Link>
          </div>
        </div>
      </motion.div>
    </section>
  );
}
