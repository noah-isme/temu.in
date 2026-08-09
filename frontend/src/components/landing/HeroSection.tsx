import { motion } from 'framer-motion';
import { Sparkles } from 'lucide-react';
import InteractiveBookingSimulator from './InteractiveBookingSimulator';

export default function HeroSection() {
  return (
    <header className="relative flex flex-col gap-12 pt-4 pb-8">
      {/* Top Tagline & Headline */}
      <div className="mx-auto max-w-3xl text-center">
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
          className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/5 px-4 py-1.5 text-xs font-semibold text-primary shadow-sm backdrop-blur-md"
        >
          <Sparkles className="h-3.5 w-3.5" />
          <span>Platform Booking Terpadu & Real-time</span>
        </motion.div>

        <motion.h1
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="mt-6 text-4xl font-black tracking-tight text-slate-900 sm:text-5xl lg:text-6xl"
        >
          Solusi Smart Booking untuk <span className="bg-gradient-to-r from-primary via-indigo-600 to-purple-600 bg-clip-text text-transparent">Penyedia Layanan</span>
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mt-6 text-lg leading-relaxed text-slate-600 sm:text-xl"
        >
          Kelola ketersediaan provider, slot jadwal real-time, dan pembayaran Midtrans otomatis dengan proteksi bentrok jadwal.
        </motion.p>
      </div>

      {/* Interactive Hero Widget */}
      <div className="mx-auto w-full max-w-4xl">
        <InteractiveBookingSimulator />
      </div>
    </header>
  );
}
