import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Calendar, Clock, CheckCircle2, Sparkles, ArrowRight, ShieldCheck } from 'lucide-react';

interface ServiceOption {
  id: string;
  name: string;
  duration: string;
  price: number;
}

const SERVICES: ServiceOption[] = [
  { id: 'consultation', name: 'Konsultasi Layanan', duration: '30m', price: 150000 },
  { id: 'full-service', name: 'Paket Full Service', duration: '60m', price: 300000 },
  { id: 'express', name: 'Express Booking', duration: '15m', price: 90000 }
];

const DATES = [
  { day: 'Senin', date: '30 Jul', value: '2026-07-30' },
  { day: 'Selasa', date: '31 Jul', value: '2026-07-31' },
  { day: 'Rabu', date: '01 Agu', value: '2026-08-01' }
];

const TIME_SLOTS = ['09:00', '11:00', '14:00', '16:00'];

export default function InteractiveBookingSimulator() {
  const [selectedService, setSelectedService] = useState<ServiceOption>(SERVICES[0]);
  const [selectedDate, setSelectedDate] = useState(DATES[0].value);
  const [selectedTime, setSelectedTime] = useState(TIME_SLOTS[1]);
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSimulate = () => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      setIsSubmitted(true);
    }, 600);
  };

  const handleReset = () => {
    setIsSubmitted(false);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.2 }}
      className="relative overflow-hidden rounded-2xl border border-slate-200/80 bg-white p-6 shadow-xl shadow-slate-200/50 backdrop-blur-xl md:p-8"
    >
      {/* Background Subtle Gradient Glow */}
      <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/10 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-20 -left-20 h-64 w-64 rounded-full bg-blue-400/10 blur-3xl" />

      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-100 pb-4">
        <div className="flex items-center gap-2">
          <span className="flex h-3 w-3 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Live Simulator</span>
        </div>
        <div className="flex items-center gap-1 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
          <Sparkles className="h-3.5 w-3.5" />
          <span>Micro-Interaction Demo</span>
        </div>
      </div>

      <AnimatePresence mode="wait">
        {!isSubmitted ? (
          <motion.div
            key="booking-form"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.2 }}
            className="mt-6 space-y-6"
          >
            {/* Step 1: Service Selection */}
            <div>
              <label className="mb-2 block text-xs font-bold uppercase tracking-wider text-slate-500">
                1. Pilih Layanan
              </label>
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {SERVICES.map((service) => {
                  const isSelected = selectedService.id === service.id;
                  return (
                    <button
                      key={service.id}
                      type="button"
                      onClick={() => setSelectedService(service)}
                      className={`relative flex flex-col justify-between rounded-xl p-3.5 text-left transition-all ${
                        isSelected
                          ? 'text-primary font-semibold'
                          : 'text-slate-600 hover:bg-slate-50'
                      }`}
                    >
                      {isSelected && (
                        <motion.div
                          layoutId="selected-service-pill"
                          className="absolute inset-0 rounded-xl border-2 border-primary bg-primary/5 shadow-sm"
                          transition={{ type: 'spring', stiffness: 380, damping: 30 }}
                        />
                      )}
                      <span className="relative z-10 text-sm font-medium">{service.name}</span>
                      <div className="relative z-10 mt-2 flex items-center justify-between text-xs text-slate-500">
                        <span>{service.duration}</span>
                        <span className="font-semibold text-primary">
                          Rp {service.price.toLocaleString('id-ID')}
                        </span>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Step 2: Date & Time Picker */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-2 block text-xs font-bold uppercase tracking-wider text-slate-500">
                  2. Tanggal Booking
                </label>
                <div className="flex gap-2">
                  {DATES.map((d) => {
                    const isSelected = selectedDate === d.value;
                    return (
                      <motion.button
                        key={d.value}
                        whileHover={{ scale: 1.02 }}
                        whileTap={{ scale: 0.98 }}
                        type="button"
                        onClick={() => setSelectedDate(d.value)}
                        className={`flex flex-1 flex-col items-center justify-center rounded-lg border py-2 text-xs transition ${
                          isSelected
                            ? 'border-primary bg-primary text-white font-medium shadow-md shadow-primary/20'
                            : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300'
                        }`}
                      >
                        <Calendar className="mb-1 h-3.5 w-3.5 opacity-80" />
                        <span>{d.day}</span>
                        <span className="text-[10px] opacity-90">{d.date}</span>
                      </motion.button>
                    );
                  })}
                </div>
              </div>

              <div>
                <label className="mb-2 block text-xs font-bold uppercase tracking-wider text-slate-500">
                  3. Slot Waktu
                </label>
                <div className="grid grid-cols-2 gap-2">
                  {TIME_SLOTS.map((slot) => {
                    const isSelected = selectedTime === slot;
                    return (
                      <motion.button
                        key={slot}
                        whileHover={{ scale: 1.03 }}
                        whileTap={{ scale: 0.97 }}
                        type="button"
                        onClick={() => setSelectedTime(slot)}
                        className={`flex items-center justify-center gap-1.5 rounded-lg border py-2 text-xs font-medium transition ${
                          isSelected
                            ? 'border-slate-900 bg-slate-900 text-white shadow-sm'
                            : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300'
                        }`}
                      >
                        <Clock className="h-3.5 w-3.5" />
                        <span>{slot} WIB</span>
                      </motion.button>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* Price & Action */}
            <div className="flex flex-col items-stretch justify-between gap-4 rounded-xl border border-slate-100 bg-slate-50/80 p-4 sm:flex-row sm:items-center">
              <div>
                <span className="text-xs text-slate-500">Total Estimasi</span>
                <div className="flex items-baseline gap-2">
                  <span className="text-2xl font-bold text-slate-900">
                    Rp {selectedService.price.toLocaleString('id-ID')}
                  </span>
                  <span className="text-xs text-emerald-600 font-medium flex items-center gap-0.5">
                    <ShieldCheck className="h-3.5 w-3.5" /> Midtrans Ready
                  </span>
                </div>
              </div>

              <motion.button
                whileHover={{ scale: 1.04 }}
                whileTap={{ scale: 0.96 }}
                type="button"
                onClick={handleSimulate}
                disabled={loading}
                className="group relative inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-6 py-3 font-semibold text-white shadow-lg shadow-primary/30 transition-all hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? (
                  <motion.div
                    animate={{ rotate: 360 }}
                    transition={{ repeat: Infinity, duration: 1, ease: 'linear' }}
                    className="h-5 w-5 border-2 border-white border-t-transparent rounded-full"
                  />
                ) : (
                  <>
                    <span>Tes Simulasi Booking</span>
                    <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                  </>
                )}
              </motion.button>
            </div>
          </motion.div>
        ) : (
          <motion.div
            key="success-state"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0 }}
            transition={{ type: 'spring', stiffness: 300, damping: 25 }}
            className="my-8 flex flex-col items-center justify-center text-center"
          >
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ type: 'spring', stiffness: 400, damping: 18 }}
              className="flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600"
            >
              <CheckCircle2 className="h-10 w-10" />
            </motion.div>
            <h4 className="mt-4 text-xl font-bold text-slate-900">Simulasi Booking Berhasil!</h4>
            <p className="mt-1 max-w-md text-sm text-slate-600">
              Sistem telah memproses slot <span className="font-semibold text-slate-900">{selectedTime}</span> untuk <span className="font-semibold text-slate-900">{selectedService.name}</span>.
            </p>
            <motion.button
              whileHover={{ scale: 1.03 }}
              whileTap={{ scale: 0.97 }}
              onClick={handleReset}
              className="mt-6 rounded-lg bg-slate-100 px-5 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-200"
            >
              Ulangi Simulasi
            </motion.button>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
