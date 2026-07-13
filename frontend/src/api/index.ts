import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? ''
});
api.defaults.withCredentials = true;

// Attach Authorization header if token present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token && config.headers) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }
  return config;
});

// Attempt automatic refresh on 401 once
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value?: unknown) => void;
  reject: (err: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((p) => {
    if (error) p.reject(error);
    else p.resolve(token);
  });
  failedQueue = [];
};

api.interceptors.response.use(undefined, async (error) => {
  const originalRequest = error.config;
  if (error.response?.status === 401 && !originalRequest._retry) {
    if (isRefreshing) {
      return new Promise(function (resolve, reject) {
        failedQueue.push({ resolve, reject });
      })
        .then(() => api(originalRequest))
        .catch((err) => Promise.reject(err));
    }

    originalRequest._retry = true;
    isRefreshing = true;
    try {
      await refresh();
      processQueue(null, localStorage.getItem('auth_token'));
      return api(originalRequest);
    } catch (err) {
      processQueue(err, null);
      return Promise.reject(err);
    } finally {
      isRefreshing = false;
    }
  }
  return Promise.reject(error);
});

export function setAuthToken(token: string | null) {
  if (token) {
    localStorage.setItem('auth_token', token);
  } else {
    localStorage.removeItem('auth_token');
  }
}

export function getAuthToken(): string | null {
  return localStorage.getItem('auth_token');
}

export function clearAuthToken() {
  localStorage.removeItem('auth_token');
}

export type Provider = {
  id: string;
  name: string;
  business_name?: string;
  available: boolean;
};

type ProviderResponse = Provider & {
  is_active?: boolean;
};

export async function fetchProviders(): Promise<Provider[]> {
  const res = await api.get('/providers');
  const providers = Array.isArray(res.data) ? res.data : (res.data?.providers ?? []);
  return (providers as ProviderResponse[]).map((provider) => ({
    ...provider,
    name: provider.name ?? provider.business_name,
    available: provider.available ?? provider.is_active ?? false,
  }));
}

export type BookingInput = {
  providerId: string;
  customerName: string;
  slot?: string | null;
  client_secret?: string;
  intent_id?: string;
  amount?: number;
  currency?: string;
};

export type PaymentIntent = {
  id: number;
  client_secret: string;
  amount: number;
  currency: string;
  status: string;
};

export type BookingResponse = {
  booking: { id: number; status: string };
  payment?: PaymentIntent;
};

export async function createBooking(input: BookingInput): Promise<BookingResponse> {
  const res = await api.post('/bookings', input);
  return res.data;
}

export type Service = {
  id: string;
  name: string;
  durationMinutes: number;
  duration?: number;
  description?: string;
  price: number;
};

type ServiceResponse = Omit<Service, 'durationMinutes'> & {
  durationMinutes?: number;
};

export async function fetchServices(): Promise<Service[]> {
  const res = await api.get('/services');
  const services = Array.isArray(res.data) ? res.data : (res.data?.services ?? []);
  return (services as ServiceResponse[]).map((service) => ({
    ...service,
    durationMinutes: service.durationMinutes ?? service.duration ?? 0,
  }));
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

export async function refresh() {
  const res = await api.post('/auth/refresh');
  const token = res.data?.token;
  if (token) setAuthToken(token);
  return res.data;
}

export async function logout() {
  const res = await api.post('/auth/logout');
  clearAuthToken();
  return res.data;
}

export async function me() {
  const res = await api.get('/me');
  return res.data;
}

export async function adminListUsers() {
  const res = await api.get('/admin/users');
  return res.data;
}

export type AdminUser = {
  id: number;
  email: string;
  name?: string;
  role: string;
};

export type AdminAudit = {
  id: number;
  action: string;
  created_at: string;
  target?: string;
  details?: string;
};

export async function adminListAudit() {
  const res = await api.get('/admin/audit');
  return res.data;
}

export async function adminPromoteUser(email: string) {
  const res = await api.post('/admin/promote', { email });
  return res.data;
}

export default api;
