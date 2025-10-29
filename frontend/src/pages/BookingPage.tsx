import BookingFlow from '../components/BookingFlow';
import PageTransition from '../components/PageTransition';

export default function BookingPage() {
  return (
    <PageTransition>
      <section className="mx-auto max-w-3xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Buat Booking</h2>
        <BookingFlow />
      </section>
    </PageTransition>
  );
}
