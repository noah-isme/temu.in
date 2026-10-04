const messages: Record<string, Record<string, string>> = {
  en: {
    login_success: 'Login successful',
    booking_success: 'Booking confirmed',
    booking_failed: 'Booking failed',
    payment_failed: 'Payment failed',
    register_success: 'Account created',
    register_failed: 'Registration failed',
    reset_link_sent: 'If the email is registered, a reset link has been sent',
    password_reset_success: 'Password updated, please log in',
    password_reset_failed: 'Password reset failed',
    email_verified: 'Email verified',
    email_verify_failed: 'Email verification failed',
    booking_conflict: 'That time slot is no longer available, please pick another',
    booking_cancelled: 'Booking cancelled',
    booking_rescheduled: 'Booking rescheduled',
    booking_status_updated: 'Booking status updated',
    schedule_saved: 'Schedule saved',
    schedule_deleted: 'Schedule deleted',
    exception_saved: 'Exception saved',
    exception_deleted: 'Exception deleted',
    save_failed: 'Save failed'
  },
  id: {
    login_success: 'Login berhasil',
    booking_success: 'Booking dikonfirmasi',
    booking_failed: 'Booking gagal',
    payment_failed: 'Pembayaran gagal',
    register_success: 'Akun berhasil dibuat',
    register_failed: 'Pendaftaran gagal',
    reset_link_sent: 'Jika email terdaftar, tautan reset telah dikirim',
    password_reset_success: 'Password diperbarui, silakan masuk',
    password_reset_failed: 'Reset password gagal',
    email_verified: 'Email terverifikasi',
    email_verify_failed: 'Verifikasi email gagal',
    booking_conflict: 'Slot waktu tersebut sudah tidak tersedia, silakan pilih yang lain',
    booking_cancelled: 'Booking dibatalkan',
    booking_rescheduled: 'Booking dijadwalkan ulang',
    booking_status_updated: 'Status booking diperbarui',
    schedule_saved: 'Jadwal disimpan',
    schedule_deleted: 'Jadwal dihapus',
    exception_saved: 'Pengecualian disimpan',
    exception_deleted: 'Pengecualian dihapus',
    save_failed: 'Gagal menyimpan'
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
