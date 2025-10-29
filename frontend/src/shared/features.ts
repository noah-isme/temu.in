import { Calendar, Clock, Mail, ShieldCheck } from 'lucide-react';

export const FEATURES = [
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

export type Feature = (typeof FEATURES)[number];
