import { rest, type ResponseResolver, type RestContext, type RestHandler, type RestRequest } from 'msw';
import type {
  ApiResponse,
  AvailabilityException,
  AvailabilitySchedule,
  Booking,
  BookingStatus,
  ExceptionInput,
  ProviderBooking,
  ProviderProfile,
  ScheduleInput,
  TimeSlot,
} from '../api';
import type { components } from '../api/schema';

// ---------------------------------------------------------------------------
// Helpers + in-memory state for the stateful mocks (bookings, provider
// dashboard). Tests may seed `mockState` directly; `resetMockState()` runs
// after every test (see vitest.setup.ts) so mutations never leak.
// ---------------------------------------------------------------------------

function clock(totalMinutes: number): string {
  const h = Math.floor(totalMinutes / 60);
  const m = totalMinutes % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

/** Slots shaped like the backend's `available_slots`: `HH:MM` start/end. */
export function mockSlots(date: string, durationMinutes: number): TimeSlot[] {
  return [9, 10, 14].map((hour) => ({
    date,
    start_time: clock(hour * 60),
    end_time: clock(hour * 60 + durationMinutes),
  }));
}

const STAMP = '2025-01-01T00:00:00Z';

export const MOCK_PROVIDER: ProviderProfile = {
  id: 'p1',
  business_name: 'Provider A',
  business_type: 'salon',
  description: 'Mock provider',
  address: 'Jl. Contoh 1',
  latitude: null,
  longitude: null,
  pricing_info: '',
  is_active: true,
  owner_name: 'Demo Owner',
  created_at: STAMP,
  updated_at: STAMP,
};

type MockState = {
  bookings: Booking[];
  providerBookings: ProviderBooking[];
  /** Provider owned by the signed-in user; null makes `GET /providers/me` 404. */
  myProvider: ProviderProfile | null;
  schedules: AvailabilitySchedule[];
  exceptions: AvailabilityException[];
  nextId: number;
};

function initialState(): MockState {
  return {
    bookings: [
      { id: 101, provider_id: 'p1', provider_name: 'Provider A', service_id: 's1', service_name: 'Haircut', booking_date: '2025-07-01', start_time: '09:00', end_time: '09:30', status: 'pending', payment_status: 'pending' },
      { id: 102, provider_id: 'p1', provider_name: 'Provider A', service_id: 's2', service_name: 'Massage', booking_date: '2025-07-02', start_time: '10:00', end_time: '11:00', status: 'confirmed', payment_status: 'paid', payment_id: 7 },
      { id: 103, provider_id: 'p2', provider_name: 'Provider B', service_id: 's1', service_name: 'Haircut', booking_date: '2025-06-01', start_time: '14:00', end_time: '14:30', status: 'completed', payment_status: 'paid' },
    ],
    providerBookings: [
      { id: 201, customer_name: 'Budi', service_id: 's1', service_name: 'Haircut', booking_date: '2025-07-03', start_time: '09:00', end_time: '09:30', status: 'pending', payment_status: 'paid' },
      { id: 202, customer_name: 'Sari', service_id: 's2', service_name: 'Massage', booking_date: '2025-07-04', start_time: '10:00', end_time: '11:00', status: 'confirmed', payment_status: 'pending' },
    ],
    myProvider: null,
    schedules: [
      { id: 'sch1', provider_id: 'p1', day_of_week: 1, start_time: '09:00', end_time: '17:00', is_available: true, created_at: STAMP, updated_at: STAMP },
    ],
    exceptions: [],
    nextId: 1000,
  };
}

export const mockState: MockState = initialState();

export function resetMockState(): void {
  Object.assign(mockState, initialState());
}

function nextId(): number {
  mockState.nextId += 1;
  return mockState.nextId;
}

type ErrorBody = components['schemas']['ErrorResponse'];

type Resolver = ResponseResolver<RestRequest, RestContext>;
type Method = 'get' | 'post' | 'put' | 'delete';

/** Registers the same resolver for `/path` and `/api/path`, like the handlers below. */
export function mockRoute(method: Method, path: string, resolver: Resolver): RestHandler[] {
  return [rest[method](path, resolver), rest[method](`/api${path}`, resolver)];
}

const CLOCK_RE = /^\d{2}:\d{2}$/;
const BOOKING_STATUSES: BookingStatus[] = ['pending', 'confirmed', 'completed', 'cancelled', 'payment_failed'];

function findAnyBooking(id: number): Booking | ProviderBooking | undefined {
  return mockState.bookings.find((b) => b.id === id) ?? mockState.providerBookings.find((b) => b.id === id);
}

function scheduleFromInput(body: Partial<ScheduleInput>): ScheduleInput | null {
  if (typeof body.day_of_week !== 'number' || body.day_of_week < 0 || body.day_of_week > 6) return null;
  if (!body.start_time || !CLOCK_RE.test(body.start_time) || !body.end_time || !CLOCK_RE.test(body.end_time)) return null;
  return { day_of_week: body.day_of_week, start_time: body.start_time, end_time: body.end_time, is_available: !!body.is_available };
}

function exceptionFromInput(providerId: string, id: string, body: Partial<ExceptionInput>): AvailabilityException | null {
  if (!body.date || !/^\d{4}-\d{2}-\d{2}$/.test(body.date)) return null;
  return {
    id,
    provider_id: providerId,
    date: body.date,
    start_time: body.start_time ? body.start_time : null,
    end_time: body.end_time ? body.end_time : null,
    is_available: !!body.is_available,
    reason: body.reason ?? '',
    created_at: STAMP,
    updated_at: STAMP,
  };
}

const statefulHandlers: RestHandler[] = [
  // --- auth flows ---------------------------------------------------------
  ...mockRoute('post', '/auth/register', async (req, res, ctx) => {
    const body = await req.json<{ email?: string; password?: string; name?: string }>();
    if (!body.email || !body.password || body.password.length < 6) {
      return res(ctx.status(400), ctx.json({ error: 'validation failed' } satisfies ErrorBody));
    }
    if (body.email === 'taken@user.test') {
      return res(ctx.status(400), ctx.json({ error: 'email exists' } satisfies ErrorBody));
    }
    return res(ctx.status(200), ctx.json({ token: 'mock-jwt-token', user: { id: 2, email: body.email, role: 'user' } } satisfies ApiResponse<'/api/auth/register', 'post'>));
  }),
  ...mockRoute('post', '/auth/forgot-password', (_req, res, ctx) =>
    res(ctx.status(200), ctx.json({ message: 'If email exists, reset link sent' } satisfies ApiResponse<'/api/auth/forgot-password', 'post'>))
  ),
  ...mockRoute('post', '/auth/reset-password', async (req, res, ctx) => {
    const body = await req.json<{ token?: string; password?: string }>();
    if (!body.token || !body.password || body.password.length < 6) {
      return res(ctx.status(400), ctx.json({ error: 'validation failed' } satisfies ErrorBody));
    }
    if (body.token === 'expired') {
      return res(ctx.status(400), ctx.json({ error: 'invalid or expired token' } satisfies ErrorBody));
    }
    return res(ctx.status(200), ctx.json({ message: 'password updated successfully' } satisfies ApiResponse<'/api/auth/reset-password', 'post'>));
  }),
  ...mockRoute('post', '/auth/verify-email', async (req, res, ctx) => {
    const body = await req.json<{ token?: string }>();
    if (!body.token || body.token === 'expired') {
      return res(ctx.status(400), ctx.json({ error: 'invalid or expired token' } satisfies ErrorBody));
    }
    return res(ctx.status(200), ctx.json({ message: 'email verified successfully' } satisfies ApiResponse<'/api/auth/verify-email', 'post'>));
  }),

  // --- customer bookings --------------------------------------------------
  ...mockRoute('get', '/bookings', (_req, res, ctx) => res(ctx.status(200), ctx.json({ bookings: mockState.bookings } satisfies ApiResponse<'/api/bookings', 'get'>))),
  ...mockRoute('get', '/bookings/:bookingId', (req, res, ctx) => {
    const booking = mockState.bookings.find((b) => b.id === Number(req.params.bookingId));
    if (!booking) return res(ctx.status(404), ctx.json({ error: 'booking not found' } satisfies ErrorBody));
    return res(ctx.status(200), ctx.json({ booking } satisfies ApiResponse<'/api/bookings/{bookingId}', 'get'>));
  }),
  ...mockRoute('post', '/bookings/:bookingId/cancel', async (req, res, ctx) => {
    const body = await req.json<{ reason?: string }>();
    if (!body.reason) return res(ctx.status(400), ctx.json({ error: 'validation failed' } satisfies ErrorBody));
    const booking = mockState.bookings.find((b) => b.id === Number(req.params.bookingId));
    if (!booking) return res(ctx.status(404), ctx.json({ error: 'booking not found' } satisfies ErrorBody));
    if (booking.status === 'completed') return res(ctx.status(400), ctx.json({ error: 'cannot cancel completed booking' } satisfies ErrorBody));
    if (booking.status !== 'cancelled') {
      booking.status = 'cancelled';
      booking.cancellation_reason = body.reason;
    }
    return res(ctx.status(200), ctx.json({ message: 'booking cancelled' } satisfies ApiResponse<'/api/bookings/{bookingId}/cancel', 'post'>));
  }),
  ...mockRoute('put', '/bookings/:bookingId', async (req, res, ctx) => {
    const body = await req.json<{ slot?: TimeSlot }>();
    const slot = body.slot;
    if (!slot || !slot.date || !slot.start_time || !slot.end_time) {
      return res(ctx.status(400), ctx.json({ error: 'slot date and time are required' } satisfies ErrorBody));
    }
    const booking = mockState.bookings.find((b) => b.id === Number(req.params.bookingId));
    if (!booking) return res(ctx.status(404), ctx.json({ error: 'booking not found' } satisfies ErrorBody));
    if (booking.status === 'cancelled' || booking.status === 'completed') {
      return res(ctx.status(500), ctx.json({ error: 'failed to reschedule booking' } satisfies ErrorBody));
    }
    booking.booking_date = slot.date;
    booking.start_time = slot.start_time;
    booking.end_time = slot.end_time;
    return res(ctx.status(200), ctx.json({ message: 'booking rescheduled' } satisfies ApiResponse<'/api/bookings/{bookingId}', 'put'>));
  }),
  ...mockRoute('post', '/bookings/:bookingId/confirm', (req, res, ctx) => {
    const booking = findAnyBooking(Number(req.params.bookingId));
    if (!booking) return res(ctx.status(404), ctx.json({ error: 'booking not found' } satisfies ErrorBody));
    if (booking.status !== 'pending') return res(ctx.status(400), ctx.json({ error: 'booking is not in pending state' } satisfies ErrorBody));
    if (booking.payment_status !== 'paid') {
      return res(ctx.status(400), ctx.json({ error: 'payment must succeed before confirming booking' } satisfies ErrorBody));
    }
    booking.status = 'confirmed';
    return res(ctx.status(200), ctx.json({ message: 'booking confirmed' } satisfies ApiResponse<'/api/bookings/{bookingId}/confirm', 'post'>));
  }),
  ...mockRoute('post', '/bookings/:bookingId/complete', (req, res, ctx) => {
    const booking = findAnyBooking(Number(req.params.bookingId));
    if (!booking) return res(ctx.status(404), ctx.json({ error: 'booking not found' } satisfies ErrorBody));
    if (booking.status !== 'confirmed') {
      return res(ctx.status(400), ctx.json({ error: 'booking must be confirmed before completion' } satisfies ErrorBody));
    }
    if (booking.payment_status !== 'paid') {
      return res(ctx.status(400), ctx.json({ error: 'payment must succeed before completion' } satisfies ErrorBody));
    }
    booking.status = 'completed';
    return res(ctx.status(200), ctx.json({ message: 'booking completed' } satisfies ApiResponse<'/api/bookings/{bookingId}/complete', 'post'>));
  }),

  // --- provider dashboard -------------------------------------------------
  ...mockRoute('get', '/providers/me', (_req, res, ctx) => {
    if (!mockState.myProvider) return res(ctx.status(404), ctx.json({ error: 'provider not found' } satisfies ErrorBody));
    return res(ctx.status(200), ctx.json({ provider: mockState.myProvider } satisfies ApiResponse<'/api/providers/me', 'get'>));
  }),
  ...mockRoute('get', '/providers/:providerId/bookings', (req, res, ctx) => {
    const status = req.url.searchParams.get('status');
    if (status && !BOOKING_STATUSES.includes(status as BookingStatus)) {
      return res(ctx.status(400), ctx.json({ error: 'invalid status filter' } satisfies ErrorBody));
    }
    const bookings = status ? mockState.providerBookings.filter((b) => b.status === status) : mockState.providerBookings;
    return res(ctx.status(200), ctx.json({ bookings } satisfies ApiResponse<'/api/providers/{providerId}/bookings', 'get'>));
  }),
  ...mockRoute('get', '/providers/:providerId/schedules', (req, res, ctx) => {
    const schedules = mockState.schedules.filter((s) => s.provider_id === req.params.providerId);
    return res(ctx.status(200), ctx.json({ schedules } satisfies ApiResponse<'/api/providers/{providerId}/schedules', 'get'>));
  }),
  ...mockRoute('post', '/providers/:providerId/schedules', async (req, res, ctx) => {
    const input = scheduleFromInput(await req.json<Partial<ScheduleInput>>());
    if (!input) return res(ctx.status(400), ctx.json({ error: 'validation failed' } satisfies ErrorBody));
    const schedule: AvailabilitySchedule = { id: `sch${nextId()}`, provider_id: String(req.params.providerId), ...input, created_at: STAMP, updated_at: STAMP };
    mockState.schedules.push(schedule);
    return res(ctx.status(201), ctx.json({ schedule } satisfies ApiResponse<'/api/providers/{providerId}/schedules', 'post', 201>));
  }),
  ...mockRoute('put', '/providers/:providerId/schedules/:id', async (req, res, ctx) => {
    const input = scheduleFromInput(await req.json<Partial<ScheduleInput>>());
    if (!input) return res(ctx.status(400), ctx.json({ error: 'validation failed' } satisfies ErrorBody));
    const existing = mockState.schedules.find((s) => s.id === req.params.id && s.provider_id === req.params.providerId);
    if (!existing) return res(ctx.status(404), ctx.json({ error: 'schedule not found' } satisfies ErrorBody));
    Object.assign(existing, input);
    return res(ctx.status(200), ctx.json({ schedule: existing } satisfies ApiResponse<'/api/providers/{providerId}/schedules/{id}', 'put'>));
  }),
  ...mockRoute('delete', '/providers/:providerId/schedules/:id', (req, res, ctx) => {
    const before = mockState.schedules.length;
    mockState.schedules = mockState.schedules.filter((s) => !(s.id === req.params.id && s.provider_id === req.params.providerId));
    if (mockState.schedules.length === before) return res(ctx.status(404), ctx.json({ error: 'schedule not found' } satisfies ErrorBody));
    return res(ctx.status(200), ctx.json({ message: 'schedule deleted' } satisfies ApiResponse<'/api/providers/{providerId}/schedules/{id}', 'delete'>));
  }),
  ...mockRoute('get', '/providers/:providerId/exceptions', (req, res, ctx) => {
    const start = req.url.searchParams.get('start_date') ?? '0000-01-01';
    const end = req.url.searchParams.get('end_date') ?? '9999-12-31';
    const exceptions = mockState.exceptions.filter(
      (e) => e.provider_id === req.params.providerId && e.date >= start && e.date <= end
    );
    return res(ctx.status(200), ctx.json({ exceptions } satisfies ApiResponse<'/api/providers/{providerId}/exceptions', 'get'>));
  }),
  ...mockRoute('post', '/providers/:providerId/exceptions', async (req, res, ctx) => {
    const exception = exceptionFromInput(String(req.params.providerId), `exc${nextId()}`, await req.json<Partial<ExceptionInput>>());
    if (!exception) return res(ctx.status(400), ctx.json({ error: 'invalid date format' } satisfies ErrorBody));
    mockState.exceptions.push(exception);
    return res(ctx.status(201), ctx.json({ exception } satisfies ApiResponse<'/api/providers/{providerId}/exceptions', 'post', 201>));
  }),
  ...mockRoute('put', '/providers/:providerId/exceptions/:id', async (req, res, ctx) => {
    const idx = mockState.exceptions.findIndex((e) => e.id === req.params.id && e.provider_id === req.params.providerId);
    if (idx < 0) return res(ctx.status(404), ctx.json({ error: 'exception not found' } satisfies ErrorBody));
    const exception = exceptionFromInput(String(req.params.providerId), String(req.params.id), await req.json<Partial<ExceptionInput>>());
    if (!exception) return res(ctx.status(400), ctx.json({ error: 'invalid date format' } satisfies ErrorBody));
    mockState.exceptions[idx] = exception;
    return res(ctx.status(200), ctx.json({ exception } satisfies ApiResponse<'/api/providers/{providerId}/exceptions/{id}', 'put'>));
  }),
  ...mockRoute('delete', '/providers/:providerId/exceptions/:id', (req, res, ctx) => {
    const before = mockState.exceptions.length;
    mockState.exceptions = mockState.exceptions.filter((e) => !(e.id === req.params.id && e.provider_id === req.params.providerId));
    if (mockState.exceptions.length === before) return res(ctx.status(404), ctx.json({ error: 'exception not found' } satisfies ErrorBody));
    return res(ctx.status(200), ctx.json({ message: 'exception deleted' } satisfies ApiResponse<'/api/providers/{providerId}/exceptions/{id}', 'delete'>));
  }),
];

const MOCK_PROVIDERS: components['schemas']['PublicProvider'][] = [
  MOCK_PROVIDER,
  { ...MOCK_PROVIDER, id: 'p2', business_name: 'Provider B', owner_name: 'Second Owner', is_active: false },
];

const MOCK_SERVICES: components['schemas']['Service'][] = [
  { id: 's1', provider_id: 'p1', name: 'Haircut', description: 'Mock haircut', duration: 30, price: 15000, is_active: true, created_at: STAMP, updated_at: STAMP },
  { id: 's2', provider_id: 'p1', name: 'Massage', description: 'Mock massage', duration: 60, price: 75000, is_active: true, created_at: STAMP, updated_at: STAMP },
];

function mockPayment(orderId: string, amount: number): components['schemas']['Payment'] {
  return {
    id: 1,
    created_at: STAMP,
    updated_at: STAMP,
    provider: 'midtrans',
    order_id: orderId,
    intent_id: orderId,
    client_secret: `cs_${Math.random().toString(36).slice(2)}`,
    amount,
    currency: 'IDR',
    status: 'created',
  };
}

function serviceAmount(serviceId: string | undefined): number {
  return MOCK_SERVICES.find((s) => s.id === serviceId)?.price ?? 0;
}

/** Mirrors `GET /availability`: `date` defaults to today, slots are sized to the service duration. */
const availabilityResolver: Resolver = (req, res, ctx) => {
  const providerId = req.url.searchParams.get('providerId') ?? '';
  const serviceId = req.url.searchParams.get('serviceId') ?? '';
  const date = req.url.searchParams.get('date') || new Date().toISOString().slice(0, 10);
  const slots = mockSlots(date, serviceId === 's2' ? 60 : 30);
  // simulate variable latency between 300-900ms
  const latency = Math.floor(300 + Math.random() * 600);
  return res(
    ctx.delay(latency),
    ctx.status(200),
    ctx.json({
      date,
      provider_id: providerId,
      service_id: serviceId,
      available_slots: slots,
      slots,
    } satisfies ApiResponse<'/api/availability', 'get'>)
  );
};

// Default handlers. Every body is checked against the generated OpenAPI types
// (`satisfies ApiResponse<...>`) so drift from backend/openapi/openapi.yaml
// fails `tsc`. Each route is registered for both `/x` and `/api/x`.
export const handlers: RestHandler[] = [
  // Stateful handlers first so specific routes (e.g. /providers/me) win.
  ...statefulHandlers,
  ...mockRoute('get', '/health/live', (_req, res, ctx) =>
    res(ctx.status(200), ctx.json({ status: 'alive' } satisfies ApiResponse<'/api/health/live', 'get'>))
  ),
  // Signed-out default. Deliberately not spec-shaped: the backend answers 401
  // here, which would route every test through the refresh interceptor.
  ...mockRoute('get', '/me', (_req, res, ctx) => res(ctx.status(200), ctx.json({ user: null }))),
  ...mockRoute('get', '/providers', (_req, res, ctx) =>
    res(
      ctx.status(200),
      ctx.json({
        providers: MOCK_PROVIDERS,
        pagination: { page: 1, limit: 20, total: MOCK_PROVIDERS.length },
      } satisfies ApiResponse<'/api/providers', 'get'>)
    )
  ),
  ...mockRoute('post', '/bookings', async (req, res, ctx) => {
    const body = await req.json<Partial<components['schemas']['CreateBookingRequest']>>();
    const slot = body.slot ?? { date: '', start_time: '', end_time: '' };
    return res(
      ctx.status(201),
      ctx.json({
        booking: {
          id: 1,
          provider_id: body.providerId ?? '',
          service_id: body.serviceId,
          booking_date: slot.date,
          start_time: slot.start_time,
          end_time: slot.end_time,
          status: 'pending',
          payment_status: 'pending',
          payment_id: 1,
        },
        payment: mockPayment('mock-order', serviceAmount(body.serviceId)),
      } satisfies ApiResponse<'/api/bookings', 'post', 201>)
    );
  }),
  ...mockRoute('get', '/services', (_req, res, ctx) =>
    res(ctx.status(200), ctx.json({ services: MOCK_SERVICES } satisfies ApiResponse<'/api/services', 'get'>))
  ),
  ...mockRoute('get', '/availability', availabilityResolver),
  ...mockRoute('post', '/auth/login', async (req, res, ctx) => {
    const { email, password } = await req.json<Partial<components['schemas']['LoginRequest']>>();
    if (email === 'demo@user.test' && password === 'password') {
      return res(
        ctx.status(200),
        ctx.json({ token: 'mock-jwt-token', user: { id: 1, email, role: 'user' } } satisfies ApiResponse<'/api/auth/login', 'post'>)
      );
    }
    return res(ctx.status(401), ctx.json({ error: 'Invalid credentials' } satisfies components['schemas']['ErrorResponse']));
  }),
  ...mockRoute('post', '/payments/create-intent', async (req, res, ctx) => {
    const body = await req.json<Partial<components['schemas']['CreateIntentRequest']>>();
    return res(
      ctx.delay(400),
      ctx.status(201),
      ctx.json(mockPayment(`order_${Date.now()}`, serviceAmount(body.serviceId)) satisfies ApiResponse<'/api/payments/create-intent', 'post', 201>)
    );
  }),
  // A client_secret containing 'fail' simulates a declined payment (the
  // backend still answers 200 and reports the payment status).
  ...mockRoute('post', '/payments/confirm', async (req, res, ctx) => {
    const { client_secret, order_id } = await req.json<Partial<components['schemas']['ConfirmPaymentRequest']>>();
    const failed = !!client_secret?.includes('fail');
    return res(
      ctx.delay(300),
      ctx.status(200),
      ctx.json({
        status: failed ? 'failed' : 'succeeded',
        transactionId: failed ? '' : `tx_${Date.now()}`,
        order_id: order_id ?? 'mock-order',
      } satisfies ApiResponse<'/api/payments/confirm', 'post'>)
    );
  }),
];
