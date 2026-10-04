import axios from 'axios';
import type { components, paths } from './schema';

// ---------------------------------------------------------------------------
// Generated-schema helpers (src/api/schema.ts is produced by `npm run gen:api`
// from backend/openapi/openapi.yaml; never edit it by hand).
// ---------------------------------------------------------------------------

type Schema<K extends keyof components['schemas']> = components['schemas'][K];
export type HttpMethod = 'get' | 'post' | 'put' | 'delete';
type JsonContent<T> = T extends { content: { 'application/json': infer B } } ? B : never;
/** JSON body of the `S` response of `M P` in the spec. */
export type ApiResponse<P extends keyof paths, M extends HttpMethod, S extends number = 200> = paths[P][M] extends {
  responses: infer R;
}
  ? S extends keyof R
    ? JsonContent<R[S]>
    : never
  : never;

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

/** UI view model of a provider, mapped from the backend `PublicProvider`. */
export type Provider = {
  id: string;
  name: string;
  business_name?: string;
  available: boolean;
};

type ProviderListResponse = paths['/api/providers']['get']['responses']['200']['content']['application/json'];

export async function fetchProviders(): Promise<Provider[]> {
  const res = await api.get<ProviderListResponse>('/providers');
  return (res.data.providers ?? []).map((provider) => ({
    id: provider.id,
    name: provider.business_name,
    business_name: provider.business_name,
    available: provider.is_active,
  }));
}

export type BookingInput = paths['/api/bookings']['post']['requestBody']['content']['application/json'];

export type TimeSlot = Schema<'TimeSlot'>;

export type PaymentIntent = Schema<'Payment'>;

export type BookingResponse = paths['/api/bookings']['post']['responses']['201']['content']['application/json'];

export async function createBooking(input: BookingInput): Promise<BookingResponse> {
  const res = await api.post<BookingResponse>('/bookings', input);
  return res.data;
}

/** UI view model of a service, mapped from the backend `Service` (`duration` is minutes). */
export type Service = {
  id: string;
  name: string;
  durationMinutes: number;
  duration?: number;
  description?: string;
  price: number;
};

type ServiceListResponse = ApiResponse<'/api/services', 'get'>;

export async function fetchServices(): Promise<Service[]> {
  const res = await api.get<ServiceListResponse>('/services');
  return (res.data.services ?? []).map((service) => ({
    id: service.id,
    name: service.name,
    description: service.description,
    price: service.price,
    duration: service.duration,
    durationMinutes: service.duration,
  }));
}

/** Response of `GET /availability`: `date` is `YYYY-MM-DD`, slot times are `HH:MM`. */
export type AvailabilityResponse = Schema<'AvailabilitySlotsResponse'>;

export async function fetchAvailability(providerId: string, serviceId: string): Promise<AvailabilityResponse> {
  const res = await api.get<AvailabilityResponse>('/availability', { params: { providerId, serviceId } });
  return res.data;
}

export async function fetchAvailabilityForDate(providerId: string, date: string, serviceId: string): Promise<AvailabilityResponse> {
  const res = await api.get<AvailabilityResponse>('/availability', { params: { providerId, date, serviceId } });
  return res.data;
}

export type PaymentConfirmResponse = Schema<'PaymentConfirmResponse'>;

export async function createPaymentIntent(providerId: string, serviceId: string, currency = 'IDR'): Promise<PaymentIntent> {
  const res = await api.post<ApiResponse<'/api/payments/create-intent', 'post', 201>>('/payments/create-intent', {
    providerId,
    serviceId,
    currency,
  });
  return res.data;
}

export async function confirmPayment(clientSecret: string, orderId?: string): Promise<PaymentConfirmResponse> {
  const res = await api.post<ApiResponse<'/api/payments/confirm', 'post'>>('/payments/confirm', {
    client_secret: clientSecret,
    order_id: orderId,
  });
  return res.data;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const res = await api.post<ApiResponse<'/api/auth/login', 'post'>>('/auth/login', { email, password });
  return res.data;
}

export async function refresh(): Promise<Schema<'TokenResponse'>> {
  const res = await api.post<ApiResponse<'/api/auth/refresh', 'post'>>('/auth/refresh');
  const token = res.data?.token;
  if (token) setAuthToken(token);
  return res.data;
}

export async function logout(): Promise<Schema<'StatusResponse'>> {
  const res = await api.post<ApiResponse<'/api/auth/logout', 'post'>>('/auth/logout');
  clearAuthToken();
  return res.data;
}

export async function me(): Promise<ApiResponse<'/api/me', 'get'>> {
  const res = await api.get<ApiResponse<'/api/me', 'get'>>('/me');
  return res.data;
}

export async function adminListUsers(): Promise<ApiResponse<'/api/admin/users', 'get'>> {
  const res = await api.get<ApiResponse<'/api/admin/users', 'get'>>('/admin/users');
  return res.data;
}

export type AdminUser = Schema<'User'>;

export type AdminAudit = Schema<'AdminAudit'>;

export async function adminListAudit(): Promise<ApiResponse<'/api/admin/audit', 'get'>> {
  const res = await api.get<ApiResponse<'/api/admin/audit', 'get'>>('/admin/audit');
  return res.data;
}

export async function adminPromoteUser(email: string): Promise<Schema<'StatusResponse'>> {
  const res = await api.post<ApiResponse<'/api/admin/promote', 'post'>>('/admin/promote', { email });
  return res.data;
}

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

type ApiErrorBody = { error?: string; message?: string };

/** HTTP status of a failed API call, or undefined for network/non-HTTP errors. */
export function getApiErrorStatus(err: unknown): number | undefined {
  return axios.isAxiosError(err) ? err.response?.status : undefined;
}

/** The backend's `error` (or `message`) text for a failed API call, else `fallback`. */
export function getApiErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined;
    return body?.error ?? body?.message ?? fallback;
  }
  return fallback;
}

// ---------------------------------------------------------------------------
// Auth flows
// ---------------------------------------------------------------------------

export type AuthUser = Schema<'AuthUser'>;

export type AuthResponse = Schema<'AuthResponse'>;

export type MessageResponse = Schema<'MessageResponse'>;

export async function register(email: string, password: string, name: string): Promise<AuthResponse> {
  const res = await api.post<ApiResponse<'/api/auth/register', 'post'>>('/auth/register', { email, password, name });
  return res.data;
}

export async function forgotPassword(email: string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/auth/forgot-password', 'post'>>('/auth/forgot-password', { email });
  return res.data;
}

export async function resetPassword(token: string, password: string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/auth/reset-password', 'post'>>('/auth/reset-password', { token, password });
  return res.data;
}

export async function verifyEmail(token: string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/auth/verify-email', 'post'>>('/auth/verify-email', { token });
  return res.data;
}

// ---------------------------------------------------------------------------
// Bookings
// ---------------------------------------------------------------------------

export type BookingStatus = Schema<'BookingStatus'>;

/** Payment state as tracked on the booking (not the payment row's own status). */
export type BookingPaymentStatus = Schema<'BookingPaymentStatus'>;

/** Customer view of a booking (backend `CustomerBookingDTO`). Dates `YYYY-MM-DD`, times `HH:MM`. */
export type Booking = Schema<'CustomerBooking'>;

/** Provider-owner view of a booking (backend `ProviderBookingDTO`). */
export type ProviderBooking = Schema<'ProviderBooking'>;

export async function fetchMyBookings(limit?: number, offset?: number): Promise<Booking[]> {
  const res = await api.get<ApiResponse<'/api/bookings', 'get'>>('/bookings', { params: { limit, offset } });
  return res.data.bookings ?? [];
}

export async function fetchBooking(id: number | string): Promise<Booking> {
  const res = await api.get<ApiResponse<'/api/bookings/{bookingId}', 'get'>>(`/bookings/${id}`);
  return res.data.booking;
}

export async function cancelBooking(id: number | string, reason: string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/bookings/{bookingId}/cancel', 'post'>>(`/bookings/${id}/cancel`, { reason });
  return res.data;
}

export async function rescheduleBooking(id: number | string, slot: TimeSlot): Promise<MessageResponse> {
  const res = await api.put<ApiResponse<'/api/bookings/{bookingId}', 'put'>>(`/bookings/${id}`, { slot });
  return res.data;
}

export async function confirmBooking(id: number | string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/bookings/{bookingId}/confirm', 'post'>>(`/bookings/${id}/confirm`);
  return res.data;
}

export async function completeBooking(id: number | string): Promise<MessageResponse> {
  const res = await api.post<ApiResponse<'/api/bookings/{bookingId}/complete', 'post'>>(`/bookings/${id}/complete`);
  return res.data;
}

// ---------------------------------------------------------------------------
// Provider dashboard
// ---------------------------------------------------------------------------

/** Public provider shape (backend `PublicProvider`). */
export type ProviderProfile = Schema<'PublicProvider'>;

/** The provider owned by the signed-in user, or null when the user has none (404). */
export async function fetchMyProvider(): Promise<ProviderProfile | null> {
  try {
    const res = await api.get<ApiResponse<'/api/providers/me', 'get'>>('/providers/me');
    return res.data.provider;
  } catch (err) {
    if (getApiErrorStatus(err) === 404) return null;
    throw err;
  }
}

export async function fetchProviderBookings(providerId: string, status?: BookingStatus): Promise<ProviderBooking[]> {
  const res = await api.get<ApiResponse<'/api/providers/{providerId}/bookings', 'get'>>(`/providers/${providerId}/bookings`, {
    params: status ? { status } : undefined,
  });
  return res.data.bookings ?? [];
}

/** Weekly schedule row; `day_of_week` 0 (Sunday)–6, times `HH:MM`. */
export type AvailabilitySchedule = Schema<'Schedule'>;

/** Body the UI always sends when creating/updating a schedule (every field set). */
export type ScheduleInput = Required<Schema<'CreateScheduleRequest'>>;

/** Date exception; `date` is `YYYY-MM-DD`, times `HH:MM` or null for a full day. */
export type AvailabilityException = Schema<'Exception'>;

type ExceptionRequest = Schema<'CreateExceptionRequest'>;
/** Body the UI sends for an exception; times are omitted for a full-day exception. */
export type ExceptionInput = Required<Pick<ExceptionRequest, 'date' | 'is_available' | 'reason'>> &
  Pick<ExceptionRequest, 'start_time' | 'end_time'>;

export async function fetchSchedules(providerId: string): Promise<AvailabilitySchedule[]> {
  const res = await api.get<ApiResponse<'/api/providers/{providerId}/schedules', 'get'>>(`/providers/${providerId}/schedules`);
  return res.data.schedules ?? [];
}

export async function createSchedule(providerId: string, body: ScheduleInput): Promise<AvailabilitySchedule> {
  const res = await api.post<ApiResponse<'/api/providers/{providerId}/schedules', 'post', 201>>(`/providers/${providerId}/schedules`, body);
  return res.data.schedule;
}

export async function updateSchedule(providerId: string, id: string, body: ScheduleInput): Promise<AvailabilitySchedule> {
  const res = await api.put<ApiResponse<'/api/providers/{providerId}/schedules/{id}', 'put'>>(`/providers/${providerId}/schedules/${id}`, body);
  return res.data.schedule;
}

export async function deleteSchedule(providerId: string, id: string): Promise<MessageResponse> {
  const res = await api.delete<ApiResponse<'/api/providers/{providerId}/schedules/{id}', 'delete'>>(`/providers/${providerId}/schedules/${id}`);
  return res.data;
}

/** Lists exceptions with `start_date`/`end_date` (`YYYY-MM-DD`, inclusive) as the backend expects. */
export async function fetchExceptions(providerId: string, start: string, end: string): Promise<AvailabilityException[]> {
  const res = await api.get<ApiResponse<'/api/providers/{providerId}/exceptions', 'get'>>(`/providers/${providerId}/exceptions`, {
    params: { start_date: start, end_date: end },
  });
  return res.data.exceptions ?? [];
}

export async function createException(providerId: string, body: ExceptionInput): Promise<AvailabilityException> {
  const res = await api.post<ApiResponse<'/api/providers/{providerId}/exceptions', 'post', 201>>(`/providers/${providerId}/exceptions`, body);
  return res.data.exception;
}

export async function updateException(providerId: string, id: string, body: ExceptionInput): Promise<AvailabilityException> {
  const res = await api.put<ApiResponse<'/api/providers/{providerId}/exceptions/{id}', 'put'>>(`/providers/${providerId}/exceptions/${id}`, body);
  return res.data.exception;
}

export async function deleteException(providerId: string, id: string): Promise<MessageResponse> {
  const res = await api.delete<ApiResponse<'/api/providers/{providerId}/exceptions/{id}', 'delete'>>(`/providers/${providerId}/exceptions/${id}`);
  return res.data;
}

export default api;
