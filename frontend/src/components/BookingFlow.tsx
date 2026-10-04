import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { confirmPayment, createBooking, fetchAvailabilityForDate, fetchProviders, fetchServices, Provider, Service, TimeSlot } from '../api';
import toast from '../lib/toast';

export default function BookingFlow(): JSX.Element {
  const [services, setServices] = useState<Service[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [servicesLoading, setServicesLoading] = useState(false);
  const [providersLoading, setProvidersLoading] = useState(false);
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [slots, setSlots] = useState<TimeSlot[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<TimeSlot | null>(null);
  const [customerName, setCustomerName] = useState('');
  const [selectedDate, setSelectedDate] = useState<string>(() => {
    const d = new Date();
    return d.toISOString().slice(0, 10);
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [paymentLink, setPaymentLink] = useState<string | null>(null);

  const queryClient = useQueryClient();
  const { data: servicesData, isLoading: servicesLoadingQuery } = useQuery(['services'], fetchServices);
  const { data: providersData, isLoading: providersLoadingQuery } = useQuery(['providers'], fetchProviders);

  useEffect(() => {
    setServices((servicesData as Service[]) || []);
    setProviders(providersData || []);
    setServicesLoading(!!servicesLoadingQuery);
    setProvidersLoading(!!providersLoadingQuery);
  }, [servicesData, providersData, servicesLoadingQuery, providersLoadingQuery]);

  useEffect(() => {
    let cancelled = false;

    async function loadSlots() {
      if (!selectedProvider || !selectedService) {
        setSlots([]);
        setSelectedSlot(null);
        return;
      }
      try {
        setSlots([]);
        setSelectedSlot(null);
        const data = await fetchAvailabilityForDate(selectedProvider, selectedDate, selectedService);
        if (!cancelled) setSlots((data.slots || []) as TimeSlot[]);
      } catch (err) {
        if (!cancelled) {
          console.error('Failed to fetch availability', err);
          setSlots([]);
        }
      }
    }

    loadSlots();
    return () => {
      cancelled = true;
    };
  }, [selectedProvider, selectedService, selectedDate]);

  async function handleConfirm() {
    if (!selectedService) {
      setError('Pilih service');
      return;
    }
    if (!selectedProvider) {
      setError('Pilih provider');
      return;
    }
    if (!customerName.trim()) {
      setError('Masukkan nama pelanggan');
      return;
    }
    if (!selectedSlot) {
      setError('Pilih waktu booking');
      return;
    }

    setLoading(true);
    setError(null);
    setSuccessMessage(null);
    setPaymentLink(null);
    try {
      const res = await createBooking({
        providerId: selectedProvider,
        serviceId: selectedService,
        customerName: customerName.trim(),
        slot: selectedSlot,
      });

      const booking = res.booking;
      const intent = res.payment;
      queryClient.invalidateQueries(['bookings']);

      if (!intent) {
        const message = `Booking confirmed: ${booking.id}`;
        setSuccessMessage(message);
        toast.success(message);
        return;
      }

      // Midtrans returns a signed checkout URL/token. The browser must finish
      // payment there; the backend will trust only the signed webhook/status.
      if (intent.redirect_url) {
        const message = `Booking created: ${booking.id}. Continue to payment.`;
        setSuccessMessage(message);
        setPaymentLink(intent.redirect_url);
        toast.success(message);
        return;
      }

      // Kept for providers/test adapters that expose a server-side status
      // endpoint without a redirect URL. This is still verified by the backend.
      const paymentResult = await confirmPayment(intent.client_secret, intent.order_id);
      if (paymentResult.status === 'succeeded') {
        const message = `Booking & payment successful: ${booking.id} (tx: ${paymentResult.transactionId})`;
        setSuccessMessage(message);
        toast.success(message);
        queryClient.invalidateQueries(['providers']);
      } else {
        const message = `Payment status: ${paymentResult.status}`;
        setError(message);
        toast.error(message);
      }
    } catch {
      setError('Booking or payment failed');
      toast.error('Booking or payment failed');
    } finally {
      setLoading(false);
    }
  }

  const service = services.find((item) => item.id === selectedService);

  return (
    <section className="max-w-2xl">
      <h2 className="text-2xl font-semibold mb-4">Booking Flow</h2>

      <div className="mb-4">
        <label htmlFor="booking-service" className="block text-sm mb-1">Service</label>
        {servicesLoading ? (
          <div className="flex items-center gap-2"><div className="h-6 w-6"><svg className="animate-spin" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg></div><span>Loading services...</span></div>
        ) : (
          <>
            <select id="booking-service" className="w-full rounded border p-2" value={selectedService ?? ''} onChange={(e) => setSelectedService(e.target.value || null)}>
              <option value="">-- pilih service --</option>
              {services.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
            {service && <p className="mt-1 text-sm text-muted-foreground">{service.durationMinutes} minutes · IDR {service.price.toLocaleString()}</p>}
          </>
        )}
      </div>

      <div className="mb-4">
        <label htmlFor="booking-provider" className="block text-sm mb-1">Provider</label>
        {providersLoading ? (
          <div className="flex items-center gap-2"><svg className="animate-spin" viewBox="0 0 24 24" width="20" height="20"><circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.2" /><path d="M22 12a10 10 0 00-10-10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" /></svg><span>Loading providers...</span></div>
        ) : (
          <select id="booking-provider" className="w-full rounded border p-2" value={selectedProvider ?? ''} onChange={(e) => setSelectedProvider(e.target.value || null)}>
            <option value="">-- pilih provider --</option>
            {providers.map((p) => (
              <option key={p.id} value={p.id}>{p.name} {p.available ? '' : '(unavailable)'}</option>
            ))}
          </select>
        )}
      </div>

      <div className="mb-4">
        <label htmlFor="booking-customer-name" className="block text-sm mb-1">Customer name</label>
        <input id="booking-customer-name" value={customerName} onChange={(e) => setCustomerName(e.target.value)} className="w-full rounded border p-2" required />
      </div>

      <div className="mb-4">
        <label htmlFor="booking-date" className="block text-sm mb-1">Date</label>
        <input id="booking-date" type="date" value={selectedDate} onChange={(e) => setSelectedDate(e.target.value)} className="rounded border p-2" />

        <label className="block text-sm mb-1 mt-4">Available slots</label>
        {slots.length === 0 && <p className="text-sm text-muted-foreground">Pilih service, provider & tanggal untuk melihat slots</p>}
        <ul className="grid gap-2">
          {slots.length === 0
            ? Array.from({ length: 3 }).map((_, i) => (
                <li key={i} className="animate-pulse">
                  <div className="h-4 w-48 bg-slate-200 rounded" />
                </li>
              ))
            : slots.map((slot) => {
                const key = `${slot.date}-${slot.start_time}-${slot.end_time}`;
                return (
                  <li key={key}>
                    <label className="flex items-center gap-2">
                      <input type="radio" checked={selectedSlot?.date === slot.date && selectedSlot?.start_time === slot.start_time && selectedSlot?.end_time === slot.end_time} onChange={() => setSelectedSlot(slot)} />
                      <span>{slot.start_time}–{slot.end_time}</span>
                    </label>
                  </li>
                );
              })}
        </ul>
      </div>

      {error && <p className="text-sm text-red-600 mb-2">{error}</p>}
      {successMessage && <p className="text-sm text-green-700 mb-2">{successMessage}</p>}
      {paymentLink && <a className="inline-block mb-3 text-primary underline" href={paymentLink} target="_blank" rel="noreferrer">Continue to payment</a>}

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
