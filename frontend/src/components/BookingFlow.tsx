import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchServices, fetchProviders, fetchAvailabilityForDate, createBooking, createPaymentIntent, confirmPayment, Service } from '../api';
import toast from '../lib/toast';

export default function BookingFlow(): JSX.Element {
  const [services, setServices] = useState<Service[]>([]);
  const [providers, setProviders] = useState<any[]>([]);
  const [servicesLoading, setServicesLoading] = useState(false);
  const [providersLoading, setProvidersLoading] = useState(false);
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [slots, setSlots] = useState<string[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<string | null>(null);
  const [selectedDate, setSelectedDate] = useState<string>(() => {
    const d = new Date();
    return d.toISOString().slice(0, 10);
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [forceFailPayment, setForceFailPayment] = useState(false);

  const queryClient = useQueryClient();
  const { data: servicesData, isLoading: servicesLoadingQuery } = useQuery(['services'], fetchServices);
  const { data: providersData, isLoading: providersLoadingQuery } = useQuery(['providers'], fetchProviders);

  useEffect(() => {
    setServices((servicesData as Service[]) || []);
    setProviders((providersData as any[]) || []);
    setServicesLoading(!!servicesLoadingQuery);
    setProvidersLoading(!!providersLoadingQuery);
  }, [servicesData, providersData, servicesLoadingQuery, providersLoadingQuery]);

  useEffect(() => {
    async function loadSlots() {
      if (!selectedProvider) return setSlots([]);
      try {
        setSlots([]);
        const data = await fetchAvailabilityForDate(selectedProvider, selectedDate);
        setSlots(data.slots || []);
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error('Failed to fetch availability', err);
        setSlots([]);
      }
    }
    loadSlots();
  }, [selectedProvider, selectedDate]);

  async function handleConfirm() {
    if (!selectedProvider) return alert('Pilih provider');
    setLoading(true);
    setError(null);
    try {
      // 1) create booking (mock)
      const booking = await createBooking({ providerId: selectedProvider, customerName: 'Demo User', slot: selectedSlot });

      // 2) create payment intent for the booking (mock)
      const amount = 100000; // mock amount
      const intent = await createPaymentIntent(amount, 'IDR');

      // optionally force fail by altering client_secret
      const clientSecret = forceFailPayment ? `${intent.client_secret}_fail` : intent.client_secret;

      // 3) confirm payment using client_secret
      const paymentResult = await confirmPayment(clientSecret);

      if (paymentResult.status === 'succeeded') {
        toast.success(`Booking & payment successful: ${booking.id} (tx: ${paymentResult.transactionId})`);
        queryClient.invalidateQueries(['providers']);
      } else {
        setError(paymentResult.message || 'Payment failed');
        toast.error(paymentResult.message || 'Payment failed');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="max-w-2xl">
      <h2 className="text-2xl font-semibold mb-4">Booking Flow (mock)</h2>
      <div className="mb-4">
        <label className="block text-sm mb-1">Service</label>
        {servicesLoading ? (
          <div className="flex items-center gap-2"><div className="h-6 w-6"><svg className="animate-spin" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg></div><span>Loading services...</span></div>
        ) : (
          <select className="w-full rounded border p-2" value={selectedService ?? ''} onChange={(e) => setSelectedService(e.target.value || null)}>
            <option value="">-- pilih service --</option>
            {services.map((s) => (
              <option key={s.id} value={s.id}>{s.name}</option>
            ))}
          </select>
        )}
      </div>

      <div className="mb-4">
        <label className="block text-sm mb-1">Provider</label>
        {providersLoading ? (
          <div className="flex items-center gap-2"><svg className="animate-spin" viewBox="0 0 24 24" width="20" height="20"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg><span>Loading providers...</span></div>
        ) : (
          <select className="w-full rounded border p-2" value={selectedProvider ?? ''} onChange={(e) => setSelectedProvider(e.target.value || null)}>
            <option value="">-- pilih provider --</option>
            {providers.map((p) => (
              <option key={p.id} value={p.id}>{p.name} {p.available ? '' : '(unavailable)'}</option>
            ))}
          </select>
        )}
      </div>

      <div className="mb-4">
        <label className="block text-sm mb-1">Date</label>
        <input type="date" value={selectedDate} onChange={(e) => setSelectedDate(e.target.value)} className="rounded border p-2" />

        <label className="block text-sm mb-1 mt-4">Available slots</label>
        {slots.length === 0 && <p className="text-sm text-muted-foreground">Pilih provider & tanggal untuk melihat slots</p>}
        <ul className="grid gap-2">
          {slots.length === 0
            ? Array.from({ length: 3 }).map((_, i) => (
                <li key={i} className="animate-pulse">
                  <div className="h-4 w-48 bg-slate-200 rounded" />
                </li>
              ))
            : slots.map((s) => (
                <li key={s}>
                  <label className="flex items-center gap-2">
                    <input type="radio" checked={selectedSlot === s} onChange={() => setSelectedSlot(s)} />
                    <span>{new Date(s).toLocaleString()}</span>
                  </label>
                </li>
              ))}
        </ul>
      </div>

      <div className="mb-4">
        <label className="flex items-center gap-2">
          <input type="checkbox" checked={forceFailPayment} onChange={(e) => setForceFailPayment(e.target.checked)} />
          <span className="text-sm">Force payment fail (for testing)</span>
        </label>
      </div>

      {error && <p className="text-sm text-red-600 mb-2">{error}</p>}

      <div>
        <button className="rounded bg-primary px-3 py-1 text-white flex items-center gap-2" disabled={loading} onClick={handleConfirm}>
          {loading ? (
            <>
              <svg className="animate-spin" viewBox="0 0 24 24" width="16" height="16"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg>
              <span>Processing...</span>
            </>
          ) : (
            'Confirm Booking'
          )}
        </button>
      </div>
    </section>
  );
}
