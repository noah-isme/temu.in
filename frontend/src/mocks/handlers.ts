import { rest } from 'msw';

// Example handlers: adapt to your API routes and responses
export const handlers = [
  // Support both root paths and /api-prefixed paths so requests to
  // http://localhost:8080/api/* are intercepted by MSW in dev.
  // GET /health
  rest.get('/health', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json({ status: 'ok' }));
  }),
  // GET /api/health
  rest.get('/api/health', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json({ status: 'ok' }));
  }),
  rest.get('/api/me', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json({ user: null }));
  }),

  // GET /providers
  rest.get('/providers', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        { id: 'p1', name: 'Provider A', available: true },
        { id: 'p2', name: 'Provider B', available: false }
      ])
    );
  }),
  // GET /api/providers
  rest.get('/api/providers', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        { id: 'p1', name: 'Provider A', available: true },
        { id: 'p2', name: 'Provider B', available: false }
      ])
    );
  }),
  // POST /bookings - create a booking (mock)
  rest.post('/bookings', async (req, res, ctx) => {
    const body = await req.json();
    const booking = {
      id: 1,
      providerId: body.providerId,
      serviceId: body.serviceId,
      customerName: body.customerName || 'Guest',
      slot: body.slot,
      status: 'confirmed',
      payment_status: 'paid'
    };
    return res(ctx.status(201), ctx.json({ booking, payment: { id: 1, order_id: 'mock-order', client_secret: 'mock-client-secret', amount: 75000, currency: 'IDR', status: 'created' } }));
  })
  ,
  rest.post('/api/bookings', async (req, res, ctx) => {
    const body = await req.json();
    const booking = {
      id: 1,
      providerId: body.providerId,
      serviceId: body.serviceId,
      customerName: body.customerName || 'Guest',
      slot: body.slot,
      status: 'pending',
      payment_status: 'pending'
    };
    return res(ctx.status(201), ctx.json({
      booking,
      payment: {
        id: 1,
        order_id: 'mock-order',
        client_secret: 'mock-client-secret',
        amount: body.serviceId === 's2' ? 75000 : 15000,
        currency: 'IDR',
        status: 'created'
      }
    }));
  })
  ,
  // GET /services
  rest.get('/services', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        { id: 's1', name: 'Haircut', durationMinutes: 30, price: 15000 },
        { id: 's2', name: 'Massage', durationMinutes: 60, price: 75000 }
      ])
    );
  })
  ,
  rest.get('/api/services', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        { id: 's1', name: 'Haircut', durationMinutes: 30, price: 15000 },
        { id: 's2', name: 'Massage', durationMinutes: 60, price: 75000 }
      ])
    );
  })
  ,
  // GET /availability?providerId=...
  rest.get('/availability', (req, res, ctx) => {
    const providerId = req.url.searchParams.get('providerId');
    const dateParam = req.url.searchParams.get('date');
    const serviceId = req.url.searchParams.get('serviceId');
    const date = dateParam || new Date().toISOString().slice(0, 10);
    const duration = serviceId === 's2' ? 60 : 30;
    const slots = [9, 10, 14].map((hour) => ({ date, start_time: `${String(hour).padStart(2, '0')}:00`, end_time: `${String(hour + duration / 60).padStart(2, '0')}:00` }));
    // simulate variable latency between 300-900ms
    const latency = Math.floor(300 + Math.random() * 600);
    return res(ctx.delay(latency), ctx.status(200), ctx.json({ providerId, date, serviceId, slots }));
  })
  ,
  rest.get('/api/availability', (req, res, ctx) => {
    const providerId = req.url.searchParams.get('providerId');
    const dateParam = req.url.searchParams.get('date');
    const serviceId = req.url.searchParams.get('serviceId');
    const date = dateParam || new Date().toISOString().slice(0, 10);
    const duration = serviceId === 's2' ? 60 : 30;
    const slots = [9, 10, 14].map((hour) => ({ date, start_time: `${String(hour).padStart(2, '0')}:00`, end_time: `${String(hour + duration / 60).padStart(2, '0')}:00` }));
    const latency = Math.floor(300 + Math.random() * 600);
    return res(ctx.delay(latency), ctx.status(200), ctx.json({ providerId, date, serviceId, slots }));
  })
  ,
  // POST /auth/login
  rest.post('/auth/login', async (req, res, ctx) => {
    const { email, password } = await req.json();
    if (email === 'demo@user.test' && password === 'password') {
      return res(ctx.status(200), ctx.json({ token: 'mock-jwt-token', user: { id: 'u1', email } }));
    }
    return res(ctx.status(401), ctx.json({ message: 'Invalid credentials' }));
  })
  ,
  rest.post('/api/auth/login', async (req, res, ctx) => {
    const { email, password } = await req.json();
    if (email === 'demo@user.test' && password === 'password') {
      return res(ctx.status(200), ctx.json({ token: 'mock-jwt-token', user: { id: 'u1', email } }));
    }
    return res(ctx.status(401), ctx.json({ message: 'Invalid credentials' }));
  })
  ,
  // POST /payments/create-intent and /api/payments/create-intent
  rest.post('/payments/create-intent', async (req, res, ctx) => {
    const body = await req.json();
    // Simulate creating a payment intent and return a client_secret-like token
    const intent = {
      id: `pi_${Date.now()}`,
      amount: body.amount || 0,
      currency: body.currency || 'IDR',
      client_secret: `cs_${Math.random().toString(36).slice(2)}`
    };
    // simulate latency
    return res(ctx.delay(400), ctx.status(201), ctx.json(intent));
  })
  ,
  rest.post('/api/payments/create-intent', async (req, res, ctx) => {
    const body = await req.json();
    const intent = {
      id: `pi_${Date.now()}`,
      amount: body.amount || 0,
      currency: body.currency || 'IDR',
      client_secret: `cs_${Math.random().toString(36).slice(2)}`
    };
    return res(ctx.delay(400), ctx.status(201), ctx.json(intent));
  })
  ,
  // POST /payments/confirm and /api/payments/confirm
  rest.post('/payments/confirm', async (req, res, ctx) => {
    const { client_secret } = await req.json();
    // Simple mock: if client_secret contains 'fail' simulate failure
    if (client_secret && client_secret.includes('fail')) {
      return res(ctx.delay(300), ctx.status(402), ctx.json({ status: 'failed', message: 'Payment failed' }));
    }
    return res(ctx.delay(300), ctx.status(200), ctx.json({ status: 'succeeded', transactionId: `tx_${Date.now()}` }));
  })
  ,
  rest.post('/api/payments/confirm', async (req, res, ctx) => {
    const { client_secret } = await req.json();
    if (client_secret && client_secret.includes('fail')) {
      return res(ctx.delay(300), ctx.status(402), ctx.json({ status: 'failed', message: 'Payment failed' }));
    }
    return res(ctx.delay(300), ctx.status(200), ctx.json({ status: 'succeeded', transactionId: `tx_${Date.now()}` }));
  })
];
