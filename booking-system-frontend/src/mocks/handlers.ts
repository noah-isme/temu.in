import { rest } from 'msw';

// Example handlers: adapt to your API routes and responses
export const handlers = [
  // GET /health
  rest.get('/health', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json({ status: 'ok' }));
  }),

  // Example: GET /providers
  rest.get('/providers', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json([
        { id: 'p1', name: 'Provider A', available: true },
        { id: 'p2', name: 'Provider B', available: false }
      ])
    );
  })
  ,
  // POST /bookings - create a booking (mock)
  rest.post('/bookings', async (req, res, ctx) => {
    const body = await req.json();
    // Create a simple mock booking id and echo data
    const booking = {
      id: `b_${Date.now()}`,
      providerId: body.providerId,
      customerName: body.customerName || 'Guest',
      slot: body.slot || null,
      status: 'confirmed'
    };
    return res(ctx.status(201), ctx.json(booking));
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
  // GET /availability?providerId=...
  rest.get('/availability', (req, res, ctx) => {
    const providerId = req.url.searchParams.get('providerId');
    const dateParam = req.url.searchParams.get('date');
    // parse date or default to today
    const base = dateParam ? new Date(dateParam) : new Date();
    const y = base.getFullYear();
    const m = base.getMonth();
    const d = base.getDate();
    // create slots at 09:00, 10:00, 14:00 for the requested date
    const slots = [9, 10, 14].map((hour) => new Date(y, m, d, hour, 0).toISOString());
    // simulate variable latency between 300-900ms
    const latency = Math.floor(300 + Math.random() * 600);
    return res(ctx.delay(latency), ctx.status(200), ctx.json({ providerId, date: base.toISOString().slice(0, 10), slots }));
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
  // POST /payments/create-intent
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
  // POST /payments/confirm
  rest.post('/payments/confirm', async (req, res, ctx) => {
    const { client_secret } = await req.json();
    // Simple mock: if client_secret contains 'fail' simulate failure
    if (client_secret && client_secret.includes('fail')) {
      return res(ctx.delay(300), ctx.status(402), ctx.json({ status: 'failed', message: 'Payment failed' }));
    }
    return res(ctx.delay(300), ctx.status(200), ctx.json({ status: 'succeeded', transactionId: `tx_${Date.now()}` }));
  })
];
