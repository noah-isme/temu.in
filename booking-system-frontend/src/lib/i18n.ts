const messages: Record<string, Record<string, string>> = {
  en: {
    login_success: 'Login successful',
    booking_success: 'Booking confirmed',
    booking_failed: 'Booking failed',
    payment_failed: 'Payment failed'
  },
  id: {
    login_success: 'Login berhasil',
    booking_success: 'Booking dikonfirmasi',
    booking_failed: 'Booking gagal',
    payment_failed: 'Pembayaran gagal'
  }
};

let locale = 'id';

export function t(key: string) {
  return messages[locale]?.[key] ?? messages['en'][key] ?? key;
}

export function setLocale(l: string) {
  locale = l;
}

export default { t, setLocale };
