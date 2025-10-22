import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? ''
});

// Attach Authorization header if token present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('mock_token');
  if (token && config.headers) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }
  return config;
});

export function setAuthToken(token: string | null) {
  if (token) {
    localStorage.setItem('mock_token', token);
  } else {
    localStorage.removeItem('mock_token');
  }
}

export function getAuthToken(): string | null {
  return localStorage.getItem('mock_token');
}

export function clearAuthToken() {
  localStorage.removeItem('mock_token');
}

export type Provider = {
  id: string;
  name: string;
  available: boolean;
};

export async function fetchProviders(): Promise<Provider[]> {
  const res = await api.get('/providers');
  return res.data;
}

export type BookingInput = {
  providerId: string;
  customerName: string;
  slot?: string | null;
};

export async function createBooking(input: BookingInput) {
  const res = await api.post('/bookings', input);
  return res.data;
}

export type Service = {
  id: string;
  name: string;
  durationMinutes: number;
  price: number;
};

export async function fetchServices(): Promise<Service[]> {
  const res = await api.get('/services');
  return res.data;
}

export async function fetchAvailability(providerId: string) {
  const res = await api.get('/availability', { params: { providerId } });
  return res.data;
}

export async function fetchAvailabilityForDate(providerId: string, date: string) {
  const res = await api.get('/availability', { params: { providerId, date } });
  return res.data;
}

export async function createPaymentIntent(amount: number, currency = 'IDR') {
  const res = await api.post('/payments/create-intent', { amount, currency });
  return res.data;
}

export async function confirmPayment(clientSecret: string) {
  const res = await api.post('/payments/confirm', { client_secret: clientSecret });
  return res.data;
}

export async function login(email: string, password: string) {
  const res = await api.post('/auth/login', { email, password });
  return res.data;
}

export default api;
