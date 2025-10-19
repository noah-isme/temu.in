# Master Prompt: Platform Pemesanan Janji Temu (Booking System)

## 🎯 Project Overview

Bangun platform pemesanan janji temu (booking system) yang lengkap untuk berbagai layanan seperti dokter, salon, konsultan, atau studio musik. Sistem ini harus production-ready dengan arsitektur yang scalable dan maintainable.

---

## 📋 Tech Stack Requirements

### Frontend
- **Framework**: React 18+ dengan TypeScript
- **Build Tool**: Vite
- **State Management**: Zustand
- **Styling**: Tailwind CSS + shadcn/ui components
- **Form Handling**: React Hook Form + Zod validation
- **Calendar**: FullCalendar atau react-big-calendar
- **HTTP Client**: Axios dengan interceptors
- **Date Handling**: date-fns atau dayjs
- **Routing**: React Router v6

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin (github.com/gin-gonic/gin)
- **Database**: PostgreSQL 15+
- **ORM**: GORM (gorm.io/gorm)
- **Cache**: Redis
- **Authentication**: JWT (golang-jwt/jwt)
- **Validation**: go-playground/validator
- **Documentation**: Swagger/OpenAPI
- **Email**: SendGrid SDK

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Reverse Proxy**: Nginx
- **Environment**: .env files dengan validation

---

## 🗄️ Database Schema

Buat database PostgreSQL dengan schema berikut:

### Tables

```sql
-- Users (customer, provider, admin)
users:
  - id (UUID, PK)
  - email (VARCHAR, UNIQUE)
  - password_hash (VARCHAR)
  - full_name (VARCHAR)
  - phone (VARCHAR)
  - role (ENUM: customer, provider, admin)
  - is_verified (BOOLEAN)
  - avatar_url (VARCHAR, nullable)
  - created_at, updated_at (TIMESTAMP)

-- Service Providers (dokter, salon, etc)
service_providers:
  - id (UUID, PK)
  - user_id (UUID, FK → users)
  - business_name (VARCHAR)
  - business_type (VARCHAR: doctor, salon, consultant, studio)
  - description (TEXT)
  - address (TEXT)
  - latitude, longitude (DECIMAL, nullable)
  - pricing_info (JSONB)
  - is_active (BOOLEAN)
  - created_at, updated_at (TIMESTAMP)

-- Services offered by providers
services:
  - id (UUID, PK)
  - provider_id (UUID, FK → service_providers)
  - name (VARCHAR)
  - description (TEXT)
  - duration (INTEGER, minutes)
  - price (DECIMAL)
  - is_active (BOOLEAN)
  - created_at, updated_at (TIMESTAMP)

-- Weekly recurring availability
availability_schedules:
  - id (UUID, PK)
  - provider_id (UUID, FK → service_providers)
  - day_of_week (INTEGER: 0-6, 0=Sunday)
  - start_time (TIME)
  - end_time (TIME)
  - is_active (BOOLEAN)
  - created_at (TIMESTAMP)

-- Exceptions (holidays, special hours)
availability_exceptions:
  - id (UUID, PK)
  - provider_id (UUID, FK → service_providers)
  - exception_date (DATE)
  - start_time (TIME, nullable)
  - end_time (TIME, nullable)
  - reason (VARCHAR)
  - is_available (BOOLEAN, false=blocked)
  - created_at (TIMESTAMP)

-- Bookings
bookings:
  - id (UUID, PK)
  - customer_id (UUID, FK → users)
  - provider_id (UUID, FK → service_providers)
  - service_id (UUID, FK → services)
  - booking_date (DATE)
  - start_time (TIME)
  - end_time (TIME)
  - status (ENUM: pending, confirmed, cancelled, completed, no_show)
  - customer_notes (TEXT)
  - admin_notes (TEXT)
  - cancellation_reason (TEXT)
  - payment_status (ENUM: unpaid, paid, refunded)
  - payment_id (VARCHAR)
  - created_at, updated_at (TIMESTAMP)
  - UNIQUE constraint: (provider_id, booking_date, start_time)

-- Notifications log
notifications:
  - id (UUID, PK)
  - user_id (UUID, FK → users)
  - booking_id (UUID, FK → bookings, nullable)
  - type (VARCHAR: booking_created, booking_confirmed, etc)
  - channel (VARCHAR: email, push)
  - status (VARCHAR: pending, sent, failed)
  - error_message (TEXT, nullable)
  - sent_at (TIMESTAMP, nullable)
  - created_at (TIMESTAMP)
```

**Indexes yang diperlukan:**
```sql
CREATE INDEX idx_bookings_provider_date ON bookings(provider_id, booking_date);
CREATE INDEX idx_bookings_customer ON bookings(customer_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_availability_provider ON availability_schedules(provider_id);
CREATE INDEX idx_services_provider ON services(provider_id);
CREATE INDEX idx_users_email ON users(email);
```

---

## 🔌 API Endpoints Specification

### Authentication (`/api/v1/auth`)
```
POST   /register
       Body: { email, password, full_name, phone, role }
       Response: { user, access_token, refresh_token }

POST   /login
       Body: { email, password }
       Response: { user, access_token, refresh_token }

POST   /refresh
       Body: { refresh_token }
       Response: { access_token }

POST   /logout
       Headers: Authorization Bearer
       Response: { message }

POST   /forgot-password
       Body: { email }
       Response: { message }

POST   /reset-password
       Body: { token, new_password }
       Response: { message }

GET    /verify-email/:token
       Response: { message }
```

### Providers (`/api/v1/providers`)
```
GET    /
       Query: ?search=&business_type=&page=1&limit=10
       Response: { data: [], pagination: {} }

GET    /:id
       Response: { provider, services[], schedules[] }

POST   /
       Auth: Required (admin or self-register)
       Body: { business_name, business_type, description, address }
       Response: { provider }

PUT    /:id
       Auth: Required (provider or admin)
       Body: { ...updateable fields }
       Response: { provider }

DELETE /:id
       Auth: Required (admin)
       Response: { message }
```

### Services (`/api/v1/providers/:provider_id/services`)
```
GET    /
       Response: { services[] }

POST   /
       Auth: Required (provider or admin)
       Body: { name, description, duration, price }
       Response: { service }

PUT    /:id
       Auth: Required (provider or admin)
       Response: { service }

DELETE /:id
       Auth: Required (provider or admin)
       Response: { message }
```

### Availability (`/api/v1/providers/:provider_id`)
```
GET    /availability
       Query: ?date=2024-01-15&service_id=xxx
       Response: { date, available_slots: ["09:00", "09:30", ...] }

GET    /schedules
       Auth: Required (provider or admin)
       Response: { schedules[], exceptions[] }

POST   /schedules
       Auth: Required (provider or admin)
       Body: { day_of_week, start_time, end_time }
       Response: { schedule }

PUT    /schedules/:id
       Auth: Required (provider or admin)
       Response: { schedule }

DELETE /schedules/:id
       Auth: Required (provider or admin)
       Response: { message }

POST   /exceptions
       Auth: Required (provider or admin)
       Body: { exception_date, start_time, end_time, reason, is_available }
       Response: { exception }
```

### Bookings (`/api/v1/bookings`)
```
GET    /
       Auth: Required
       Query: ?status=&start_date=&end_date=&page=1&limit=10
       Response: { data: [], pagination: {} }
       Note: Customer sees own bookings, Provider sees their bookings, Admin sees all

POST   /
       Auth: Required (customer)
       Body: { provider_id, service_id, booking_date, start_time, customer_notes }
       Response: { booking }
       Logic: 
         - Validate slot availability
         - Check for conflicts
         - Create booking with status=pending
         - Send notification email

GET    /:id
       Auth: Required
       Response: { booking, customer, provider, service }

PUT    /:id
       Auth: Required (customer or provider or admin)
       Body: { customer_notes, admin_notes }
       Response: { booking }

DELETE /:id (cancel booking)
       Auth: Required
       Body: { cancellation_reason }
       Response: { message }
       Logic: Update status=cancelled, send notification

POST   /:id/confirm
       Auth: Required (provider or admin)
       Response: { booking }
       Logic: Update status=confirmed, send notification

POST   /:id/reject
       Auth: Required (provider or admin)
       Body: { reason }
       Response: { message }
       Logic: Update status=cancelled, send notification

POST   /:id/complete
       Auth: Required (provider or admin)
       Response: { booking }
       Logic: Update status=completed

GET    /:id/reschedule-options
       Auth: Required
       Query: ?service_id=xxx
       Response: { available_dates_and_slots }

POST   /:id/reschedule
       Auth: Required
       Body: { new_date, new_start_time, reason }
       Response: { booking }
       Logic: Cancel old, create new, link them, send notification
```

### Payments (`/api/v1/payments`)
```
POST   /create
       Auth: Required
       Body: { booking_id, payment_method }
       Response: { payment_url, payment_token }
       Integration: Midtrans Snap API

POST   /callback
       Body: { ...midtrans_notification }
       Response: { message }
       Logic: 
         - Verify signature
         - Update booking payment_status
         - Update booking status if paid
         - Send confirmation email

GET    /:payment_id
       Auth: Required
       Response: { payment_details }
```

### Admin Dashboard (`/api/v1/admin`)
```
GET    /analytics
       Auth: Required (admin)
       Query: ?start_date=&end_date=
       Response: {
         total_bookings, confirmed_bookings, cancelled_bookings,
         total_revenue, bookings_by_status[], bookings_by_date[],
         top_providers[], top_services[]
       }

GET    /bookings
       Auth: Required (admin)
       Query: Full filter options
       Response: { data: [], pagination: {} }

GET    /users
       Auth: Required (admin)
       Query: ?role=&search=&page=1
       Response: { data: [], pagination: {} }

PUT    /users/:id/verify
       Auth: Required (admin)
       Response: { user }

DELETE /users/:id
       Auth: Required (admin)
       Response: { message }
```

---

## 🏗️ Project Structure

### Backend (Go)
```
booking-system-backend/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration loader
│   ├── database/
│   │   ├── database.go            # DB connection
│   │   └── migrations/            # SQL migrations
│   ├── middleware/
│   │   ├── auth.go                # JWT authentication
│   │   ├── cors.go                # CORS handler
│   │   ├── logger.go              # Request logging
│   │   └── rate_limit.go          # Rate limiting
│   ├── models/
│   │   ├── user.go
│   │   ├── provider.go
│   │   ├── service.go
│   │   ├── booking.go
│   │   └── notification.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── provider_repository.go
│   │   ├── booking_repository.go
│   │   └── availability_repository.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── booking_service.go     # Core booking logic
│   │   ├── availability_service.go # Slot calculation
│   │   ├── notification_service.go
│   │   └── payment_service.go
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── provider_handler.go
│   │   ├── booking_handler.go
│   │   └── admin_handler.go
│   ├── dto/
│   │   └── *.go                   # Request/Response DTOs
│   └── utils/
│       ├── jwt.go
│       ├── password.go            # bcrypt helpers
│       ├── validator.go
│       └── response.go            # Standard API responses
├── pkg/
│   ├── email/
│   │   ├── client.go              # SendGrid client
│   │   └── templates/             # Email templates
│   └── cache/
│       └── redis.go               # Redis client
├── docs/
│   └── swagger.yaml               # API documentation
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```

### Frontend (React)
```
booking-system-frontend/
├── public/
├── src/
│   ├── api/
│   │   ├── axios.ts               # Axios instance with interceptors
│   │   ├── auth.api.ts
│   │   ├── booking.api.ts
│   │   ├── provider.api.ts
│   │   └── admin.api.ts
│   ├── components/
│   │   ├── ui/                    # shadcn/ui components
│   │   ├── layout/
│   │   │   ├── Navbar.tsx
│   │   │   ├── Footer.tsx
│   │   │   └── Sidebar.tsx
│   │   ├── auth/
│   │   │   ├── LoginForm.tsx
│   │   │   ├── RegisterForm.tsx
│   │   │   └── ProtectedRoute.tsx
│   │   ├── booking/
│   │   │   ├── BookingCalendar.tsx
│   │   │   ├── BookingForm.tsx
│   │   │   ├── BookingCard.tsx
│   │   │   └── TimeSlotPicker.tsx
│   │   ├── provider/
│   │   │   ├── ProviderCard.tsx
│   │   │   ├── ProviderList.tsx
│   │   │   └── ServiceList.tsx
│   │   └── admin/
│   │       ├── DashboardStats.tsx
│   │       ├── BookingTable.tsx
│   │       └── UserManagement.tsx
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useBooking.ts
│   │   └── useDebounce.ts
│   ├── pages/
│   │   ├── Home.tsx
│   │   ├── auth/
│   │   │   ├── Login.tsx
│   │   │   └── Register.tsx
│   │   ├── providers/
│   │   │   ├── ProviderList.tsx
│   │   │   └── ProviderDetail.tsx
│   │   ├── bookings/
│   │   │   ├── CreateBooking.tsx
│   │   │   ├── MyBookings.tsx
│   │   │   └── BookingDetail.tsx
│   │   ├── dashboard/
│   │   │   ├── CustomerDashboard.tsx
│   │   │   ├── ProviderDashboard.tsx
│   │   │   └── AdminDashboard.tsx
│   │   └── NotFound.tsx
│   ├── store/
│   │   ├── authStore.ts           # Zustand store for auth
│   │   ├── bookingStore.ts
│   │   └── uiStore.ts
│   ├── types/
│   │   ├── auth.types.ts
│   │   ├── booking.types.ts
│   │   └── provider.types.ts
│   ├── utils/
│   │   ├── dateHelpers.ts
│   │   ├── formatters.ts
│   │   └── constants.ts
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── package.json
├── tsconfig.json
├── tailwind.config.js
└── vite.config.ts
```

---

## 🔐 Authentication & Authorization Flow

### JWT Implementation
```go
// Access Token: 15 minutes expiry
// Refresh Token: 7 days expiry, stored in httpOnly cookie

Claims structure:
{
  "user_id": "uuid",
  "email": "user@example.com",
  "role": "customer|provider|admin",
  "exp": timestamp
}
```

### Authorization Rules
- **Customer**: 
  - Can create bookings
  - Can view/cancel own bookings
  - Can view provider list and details
  
- **Provider**:
  - All customer permissions
  - Can manage own services
  - Can manage own availability
  - Can view/confirm/reject bookings for their services
  - Can view own analytics
  
- **Admin**:
  - Full access to all resources
  - Can manage all users, providers, bookings
  - Can view system-wide analytics

---

## 🧮 Core Business Logic

### 1. Available Slots Calculation Algorithm

```go
// High-level pseudocode
func GetAvailableSlots(providerID, date, serviceDuration) []TimeSlot {
    // 1. Get day of week
    dayOfWeek := date.Weekday()
    
    // 2. Get availability schedules for this day
    schedules := GetSchedulesByDayOfWeek(providerID, dayOfWeek)
    if schedules is empty {
        return []
    }
    
    // 3. Check for exceptions (holidays, special hours)
    exception := GetException(providerID, date)
    if exception exists and is_available == false {
        return [] // Fully blocked
    }
    if exception exists and is_available == true {
        schedules = exception schedules // Override with special hours
    }
    
    // 4. Generate all possible time slots
    allSlots := []
    for each schedule in schedules {
        currentTime := schedule.start_time
        while currentTime + serviceDuration <= schedule.end_time {
            allSlots.append(currentTime)
            currentTime += 30 minutes // or configurable interval
        }
    }
    
    // 5. Get existing bookings for this date
    existingBookings := GetBookingsByProviderAndDate(providerID, date)
    
    // 6. Filter out booked slots
    availableSlots := []
    for each slot in allSlots {
        isAvailable := true
        for each booking in existingBookings {
            if slot overlaps with [booking.start_time, booking.end_time] {
                isAvailable = false
                break
            }
        }
        if isAvailable {
            availableSlots.append(slot)
        }
    }
    
    // 7. Cache result in Redis (TTL: 5 minutes)
    cache.Set("slots:" + providerID + ":" + date, availableSlots, 5*time.Minute)
    
    return availableSlots
}
```

### 2. Booking Creation with Conflict Prevention

```go
func CreateBooking(bookingData) (Booking, error) {
    // 1. Start database transaction
    tx := db.Begin()
    defer tx.Rollback()
    
    // 2. Acquire distributed lock (Redis)
    lockKey := "booking-lock:" + providerID + ":" + date + ":" + startTime
    lock := redis.SetNX(lockKey, "locked", 10*time.Second)
    if !lock {
        return error("Slot is being booked by another user")
    }
    defer redis.Del(lockKey)
    
    // 3. Double-check slot availability within transaction
    existingBooking := tx.Where(
        "provider_id = ? AND booking_date = ? AND start_time = ?",
        providerID, date, startTime
    ).First(&Booking{})
    
    if existingBooking exists {
        return error("Slot is no longer available")
    }
    
    // 4. Create booking with status = "pending"
    booking := Booking{
        CustomerID: customerID,
        ProviderID: providerID,
        ServiceID: serviceID,
        BookingDate: date,
        StartTime: startTime,
        EndTime: startTime + serviceDuration,
        Status: "pending",
        PaymentStatus: "unpaid"
    }
    tx.Create(&booking)
    
    // 5. Commit transaction
    tx.Commit()
    
    // 6. Invalidate cache
    cache.Delete("slots:" + providerID + ":" + date)
    
    // 7. Queue notification (async)
    notificationQueue.Enqueue({
        Type: "booking_created",
        BookingID: booking.ID,
        Recipients: [customer.Email, provider.Email]
    })
    
    return booking, nil
}
```

### 3. Email Notification System

```go
// Email templates needed:
1. booking_confirmation.html
   - For customer: booking details, next steps
   - For provider: new booking notification

2. booking_reminder.html (24h before appointment)
   - For customer: reminder with details

3. booking_cancelled.html
   - For both: cancellation notification with reason

4. booking_rescheduled.html
   - For both: old and new schedule

5. payment_receipt.html
   - For customer: payment confirmation with invoice

// Implementation pattern:
func SendEmail(templateName, recipient, data) {
    template := LoadTemplate(templateName)
    htmlContent := template.Render(data)
    
    sendgrid.Send({
        From: "noreply@bookingsystem.com",
        To: recipient,
        Subject: data.Subject,
        HTML: htmlContent
    })
}

// Use worker queue for async processing
```

---

## 💳 Payment Integration (Midtrans)

### Setup Steps
```
1. Register Midtrans account (sandbox for development)
2. Get Server Key and Client Key
3. Use Snap API for hosted payment page
```

### Implementation Flow

```go
// 1. Create Payment
func CreatePayment(bookingID) (PaymentResponse, error) {
    booking := GetBooking(bookingID)
    service := GetService(booking.ServiceID)
    
    // Prepare Midtrans request
    midtransReq := {
        "transaction_details": {
            "order_id": booking.ID,
            "gross_amount": service.Price
        },
        "customer_details": {
            "first_name": customer.FullName,
            "email": customer.Email,
            "phone": customer.Phone
        },
        "item_details": [{
            "id": service.ID,
            "price": service.Price,
            "quantity": 1,
            "name": service.Name
        }]
    }
    
    // Call Midtrans Snap API
    response := midtrans.CreateSnapTransaction(midtransReq)
    
    // Save payment_id to booking
    booking.PaymentID = response.Token
    db.Save(&booking)
    
    return {
        "payment_url": response.RedirectURL,
        "payment_token": response.Token
    }
}

// 2. Handle Midtrans Callback
func HandlePaymentCallback(notificationData) {
    // Verify signature
    isValid := midtrans.VerifySignature(notificationData)
    if !isValid {
        return error("Invalid signature")
    }
    
    orderID := notificationData["order_id"]
    transactionStatus := notificationData["transaction_status"]
    fraudStatus := notificationData["fraud_status"]
    
    booking := GetBookingByID(orderID)
    
    if transactionStatus == "capture" || transactionStatus == "settlement" {
        if fraudStatus == "accept" {
            // Payment successful
            booking.PaymentStatus = "paid"
            booking.Status = "confirmed"
            db.Save(&booking)
            
            // Send confirmation email
            SendEmail("booking_confirmation", [customer, provider], booking)
        }
    } else if transactionStatus == "cancel" || transactionStatus == "deny" {
        booking.PaymentStatus = "unpaid"
        booking.Status = "cancelled"
        db.Save(&booking)
    }
}
```

---

## 🎨 Frontend Key Features Implementation

### 1. Booking Calendar Component

```tsx
// Use FullCalendar with custom event rendering
<FullCalendar
  plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
  initialView="timeGridWeek"
  events={bookings}
  selectable={true}
  select={handleDateSelect}
  eventClick={handleEventClick}
  slotDuration="00:30:00"
  slotLabelInterval="01:00"
  businessHours={providerSchedule}
/>
```

### 2. Time Slot Picker

```tsx
// Fetch available slots when date/service changes
const TimeSlotPicker = ({ providerId, serviceId, selectedDate }) => {
  const [slots, setSlots] = useState([]);
  const [loading, setLoading] = useState(false);
  
  useEffect(() => {
    if (selectedDate && serviceId) {
      fetchAvailableSlots();
    }
  }, [selectedDate, serviceId]);
  
  const fetchAvailableSlots = async () => {
    setLoading(true);
    const response = await api.get(
      `/providers/${providerId}/availability`,
      { params: { date: selectedDate, service_id: serviceId } }
    );
    setSlots(response.data.available_slots);
    setLoading(false);
  };
  
  return (
    <div className="grid grid-cols-4 gap-2">
      {slots.map(slot => (
        <Button
          key={slot}
          onClick={() => onSelectSlot(slot)}
          variant={selectedSlot === slot ? "default" : "outline"}
        >
          {slot}
        </Button>
      ))}
    </div>
  );
};
```

### 3. Real-time Slot Updates (Optional Enhancement)

```tsx
// Use polling or WebSocket
useEffect(() => {
  const interval = setInterval(() => {
    fetchAvailableSlots();
  }, 30000); // Refresh every 30 seconds
  
  return () => clearInterval(interval);
}, [selectedDate, serviceId]);
```

---

## 🐳 Docker Configuration

### Backend Dockerfile
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["./main"]
```

### Frontend Dockerfile
```dockerfile
# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Run stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: booking_system
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: booking_system
      REDIS_HOST: redis
      REDIS_PORT: 6379
      JWT_SECRET: your-secret-key-change-in-production
      SENDGRID_API_KEY: ${SENDGRID_API_KEY}
      MIDTRANS_SERVER_KEY: ${MIDTRANS_SERVER_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
    volumes:
      - ./backend:/app

  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
    environment:
      VITE_API_URL: http://localhost:8080/api/v1

volumes:
  postgres_data:
  redis_data:
```

---

## ✅ Development Checklist

### Phase 1: Foundation (Week 1-2)
- [ ] Setup project structure (backend + frontend)
- [ ] Configure Docker environment
- [ ] Setup PostgreSQL with migrations
- [ ] Setup Redis connection
- [ ] Implement authentication system (register, login, JWT)
- [ ] Create user, provider, service models and repositories
- [ ] Build basic CRUD endpoints for providers and services
- [ ] Create frontend authentication pages (login, register)
- [ ] Setup React Router and protected routes
- [ ] Implement Zustand store for auth state
- [ ] Create basic layout components (Navbar, Footer)

### Phase 2: Core Booking System (Week 3-4)
- [ ] Implement availability schedules CRUD
- [ ] Implement availability exceptions CRUD
- [ ] Build available slots calculation algorithm
- [ ] Implement Redis caching for slots
- [ ] Create booking model and repository
- [ ] Implement booking creation with conflict prevention
- [ ] Build booking CRUD endpoints
- [ ] Create provider list and detail pages
- [ ] Implement booking calendar component
- [ ] Create time slot picker component
- [ ] Build booking form with validation
- [ ] Implement booking list and detail pages
- [ ] Add booking status management (confirm, cancel)

### Phase 3: Notifications & Payments (Week 5)
- [ ] Setup SendGrid integration
- [ ] Create email templates (HTML)
- [ ] Implement notification service (async queue)
- [ ] Send booking confirmation emails
- [ ] Send booking reminder emails (24h before)
- [ ] Setup Midtrans integration (sandbox)
- [ ] Implement payment creation endpoint
- [ ] Implement payment callback handler
- [ ] Build payment flow in frontend
- [ ] Add payment status tracking
- [ ] Send payment receipt emails

### Phase 4: Admin Dashboard (Week 6)
- [ ] Build admin analytics endpoint
- [ ] Create admin dashboard with stats
- [ ] Implement booking management for admin
- [ ] Add user management features
- [ ] Create charts for analytics (recharts)
- [ ] Add export functionality (CSV/PDF)
- [ ] Implement provider approval workflow
- [ ] Build admin notification center

### Phase 5: Polish & Testing (Week 7-8)
- [ ] Add form validation with Zod schemas
- [ ] Implement error handling and user feedback
- [ ] Add loading states and skeletons
- [ ] Implement responsive design (mobile-first)
- [ ] Add search and filter functionality
- [ ] Implement pagination for all lists
- [ ] Write unit tests (backend)
- [ ] Write integration tests (API endpoints)
- [ ] Write E2E tests (Playwright/Cypress)
- [ ] Add API rate limiting
- [ ] Implement request logging
- [ ] Setup error tracking (Sentry optional)
- [ ] Optimize performance (lazy loading, code splitting)
- [ ] Add SEO meta tags
- [ ] Create API documentation (Swagger)
- [ ] Write README and setup guides

---

## 🔧 Environment Variables

### Backend (.env)
```bash
# Server
PORT=8080
ENV=development # development, staging, production
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=booking_system
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h # 7 days

# SendGrid
SENDGRID_API_KEY=your-sendgrid-api-key
SENDGRID_FROM_EMAIL=noreply@yourdomain.com
SENDGRID_FROM_NAME=Booking System

# Midtrans
MIDTRANS_SERVER_KEY=your-midtrans-server-key
MIDTRANS_CLIENT_KEY=your-midtrans-client-key
MIDTRANS_IS_PRODUCTION=false

# Google Calendar (Optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# Application
APP_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000

# Email Templates
EMAIL_TEMPLATE_DIR=./pkg/email/templates

# File Upload (Optional)
MAX_UPLOAD_SIZE=5242880 # 5MB in bytes
UPLOAD_DIR=./uploads
```

### Frontend (.env)
```bash
# API
VITE_API_URL=http://localhost:8080/api/v1
VITE_API_TIMEOUT=30000

# Midtrans
VITE_MIDTRANS_CLIENT_KEY=your-midtrans-client-key
VITE_MIDTRANS_ENVIRONMENT=sandbox # sandbox or production

# Google Maps (Optional)
VITE_GOOGLE_MAPS_API_KEY=your-google-maps-api-key

# App Config
VITE_APP_NAME=Booking System
VITE_APP_DESCRIPTION=Platform pemesanan janji temu untuk berbagai layanan

# Feature Flags
VITE_ENABLE_PAYMENTS=true
VITE_ENABLE_GOOGLE_CALENDAR=false
VITE_ENABLE_ANALYTICS=false
```

---

## 📝 Code Examples & Patterns

### 1. Standardized API Response Format

```go
// utils/response.go
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorData  `json:"error,omitempty"`
}

type ErrorData struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

type PaginationMeta struct {
    Page        int   `json:"page"`
    Limit       int   `json:"limit"`
    TotalPages  int   `json:"total_pages"`
    TotalItems  int64 `json:"total_items"`
}

type PaginatedResponse struct {
    Success    bool           `json:"success"`
    Data       interface{}    `json:"data"`
    Pagination PaginationMeta `json:"pagination"`
}

// Helper functions
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
    c.JSON(statusCode, APIResponse{
        Success: true,
        Message: message,
        Data:    data,
    })
}

func ErrorResponse(c *gin.Context, statusCode int, code string, message string, details map[string]string) {
    c.JSON(statusCode, APIResponse{
        Success: false,
        Error: &ErrorData{
            Code:    code,
            Message: message,
            Details: details,
        },
    })
}

func PaginatedSuccessResponse(c *gin.Context, data interface{}, pagination PaginationMeta) {
    c.JSON(http.StatusOK, PaginatedResponse{
        Success:    true,
        Data:       data,
        Pagination: pagination,
    })
}
```

### 2. Middleware Examples

```go
// middleware/auth.go
func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            ErrorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authorization header is required", nil)
            c.Abort()
            return
        }

        // Extract token (Bearer <token>)
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        
        // Validate JWT
        claims, err := utils.ValidateJWT(tokenString)
        if err != nil {
            ErrorResponse(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error(), nil)
            c.Abort()
            return
        }

        // Set user info in context
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_role", claims.Role)
        
        c.Next()
    }
}

func RoleRequired(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, exists := c.Get("user_role")
        if !exists {
            ErrorResponse(c, http.StatusForbidden, "ROLE_NOT_FOUND", "User role not found", nil)
            c.Abort()
            return
        }

        role := userRole.(string)
        for _, allowedRole := range allowedRoles {
            if role == allowedRole {
                c.Next()
                return
            }
        }

        ErrorResponse(c, http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", "You don't have permission to access this resource", nil)
        c.Abort()
    }
}

// middleware/rate_limit.go
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
    // Use Redis for distributed rate limiting
    return func(c *gin.Context) {
        ip := c.ClientIP()
        key := fmt.Sprintf("rate_limit:%s", ip)
        
        count, err := redis.Incr(key)
        if err != nil {
            c.Next()
            return
        }

        if count == 1 {
            redis.Expire(key, window)
        }

        if count > limit {
            ErrorResponse(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", 
                fmt.Sprintf("Too many requests. Limit: %d requests per %v", limit, window), nil)
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 3. Repository Pattern Example

```go
// repository/booking_repository.go
type BookingRepository interface {
    Create(booking *models.Booking) error
    FindByID(id string) (*models.Booking, error)
    FindByCustomerID(customerID string, filters BookingFilters) ([]models.Booking, int64, error)
    FindByProviderID(providerID string, filters BookingFilters) ([]models.Booking, int64, error)
    Update(booking *models.Booking) error
    Delete(id string) error
    CheckConflict(providerID string, date time.Time, startTime, endTime time.Time) (bool, error)
    FindOverlappingBookings(providerID string, date time.Time, startTime, endTime time.Time) ([]models.Booking, error)
    GetBookingStatistics(providerID string, startDate, endDate time.Time) (*BookingStatistics, error)
}

type bookingRepository struct {
    db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
    return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(booking *models.Booking) error {
    return r.db.Create(booking).Error
}

func (r *bookingRepository) FindByID(id string) (*models.Booking, error) {
    var booking models.Booking
    err := r.db.Preload("Customer").Preload("Provider").Preload("Service").
        First(&booking, "id = ?", id).Error
    if err != nil {
        return nil, err
    }
    return &booking, nil
}

func (r *bookingRepository) FindByCustomerID(customerID string, filters BookingFilters) ([]models.Booking, int64, error) {
    var bookings []models.Booking
    var total int64

    query := r.db.Model(&models.Booking{}).Where("customer_id = ?", customerID)

    // Apply filters
    if filters.Status != "" {
        query = query.Where("status = ?", filters.Status)
    }
    if !filters.StartDate.IsZero() {
        query = query.Where("booking_date >= ?", filters.StartDate)
    }
    if !filters.EndDate.IsZero() {
        query = query.Where("booking_date <= ?", filters.EndDate)
    }

    // Count total
    query.Count(&total)

    // Paginate and fetch
    offset := (filters.Page - 1) * filters.Limit
    err := query.Preload("Provider").Preload("Service").
        Offset(offset).Limit(filters.Limit).
        Order("booking_date DESC, start_time DESC").
        Find(&bookings).Error

    return bookings, total, err
}

func (r *bookingRepository) CheckConflict(providerID string, date time.Time, startTime, endTime time.Time) (bool, error) {
    var count int64
    err := r.db.Model(&models.Booking{}).
        Where("provider_id = ? AND booking_date = ? AND status NOT IN (?)", 
            providerID, date, []string{"cancelled", "completed"}).
        Where("(start_time < ? AND end_time > ?) OR (start_time >= ? AND start_time < ?)",
            endTime, startTime, startTime, endTime).
        Count(&count).Error
    
    return count > 0, err
}
```

### 4. Service Layer Example

```go
// service/booking_service.go
type BookingService interface {
    CreateBooking(req *dto.CreateBookingRequest, customerID string) (*models.Booking, error)
    GetBooking(id string, userID string, role string) (*models.Booking, error)
    ListBookings(userID string, role string, filters BookingFilters) ([]models.Booking, PaginationMeta, error)
    CancelBooking(id string, userID string, role string, reason string) error
    ConfirmBooking(id string, providerID string) error
    RescheduleBooking(id string, newDate time.Time, newStartTime time.Time) (*models.Booking, error)
}

type bookingService struct {
    bookingRepo      repository.BookingRepository
    availabilityRepo repository.AvailabilityRepository
    serviceRepo      repository.ServiceRepository
    notificationSvc  NotificationService
    cache           *redis.Client
}

func NewBookingService(
    bookingRepo repository.BookingRepository,
    availabilityRepo repository.AvailabilityRepository,
    serviceRepo repository.ServiceRepository,
    notificationSvc NotificationService,
    cache *redis.Client,
) BookingService {
    return &bookingService{
        bookingRepo:      bookingRepo,
        availabilityRepo: availabilityRepo,
        serviceRepo:      serviceRepo,
        notificationSvc:  notificationSvc,
        cache:           cache,
    }
}

func (s *bookingService) CreateBooking(req *dto.CreateBookingRequest, customerID string) (*models.Booking, error) {
    // 1. Validate service exists
    service, err := s.serviceRepo.FindByID(req.ServiceID)
    if err != nil {
        return nil, fmt.Errorf("service not found")
    }

    // 2. Calculate end time
    endTime := req.StartTime.Add(time.Duration(service.Duration) * time.Minute)

    // 3. Acquire distributed lock
    lockKey := fmt.Sprintf("booking-lock:%s:%s:%s", req.ProviderID, req.BookingDate, req.StartTime)
    locked, err := s.cache.SetNX(context.Background(), lockKey, "locked", 10*time.Second).Result()
    if err != nil || !locked {
        return nil, fmt.Errorf("slot is being booked by another user")
    }
    defer s.cache.Del(context.Background(), lockKey)

    // 4. Check for conflicts
    hasConflict, err := s.bookingRepo.CheckConflict(req.ProviderID, req.BookingDate, req.StartTime, endTime)
    if err != nil {
        return nil, err
    }
    if hasConflict {
        return nil, fmt.Errorf("time slot is not available")
    }

    // 5. Create booking
    booking := &models.Booking{
        ID:            uuid.New().String(),
        CustomerID:    customerID,
        ProviderID:    req.ProviderID,
        ServiceID:     req.ServiceID,
        BookingDate:   req.BookingDate,
        StartTime:     req.StartTime,
        EndTime:       endTime,
        CustomerNotes: req.CustomerNotes,
        Status:        "pending",
        PaymentStatus: "unpaid",
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    err = s.bookingRepo.Create(booking)
    if err != nil {
        return nil, err
    }

    // 6. Invalidate availability cache
    cacheKey := fmt.Sprintf("slots:%s:%s", req.ProviderID, req.BookingDate.Format("2006-01-02"))
    s.cache.Del(context.Background(), cacheKey)

    // 7. Send notifications (async)
    go s.notificationSvc.SendBookingCreatedNotification(booking)

    return booking, nil
}
```

### 5. Frontend API Client with Interceptors

```typescript
// api/axios.ts
import axios, { AxiosInstance, AxiosError } from 'axios';
import { useAuthStore } from '@/store/authStore';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

const apiClient: AxiosInstance = axios.create({
  baseURL: API_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor: Add auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().accessToken;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor: Handle token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as any;

    // If 401 and not already retried
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        const refreshToken = useAuthStore.getState().refreshToken;
        if (!refreshToken) {
          throw new Error('No refresh token');
        }

        // Attempt to refresh token
        const response = await axios.post(`${API_URL}/auth/refresh`, {
          refresh_token: refreshToken,
        });

        const { access_token } = response.data.data;
        useAuthStore.getState().setAccessToken(access_token);

        // Retry original request with new token
        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        return apiClient(originalRequest);
      } catch (refreshError) {
        // Refresh failed, logout user
        useAuthStore.getState().logout();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;
```

### 6. React Hook for Bookings

```typescript
// hooks/useBooking.ts
import { useState, useEffect } from 'react';
import { bookingApi } from '@/api/booking.api';
import { Booking, BookingFilters } from '@/types/booking.types';
import { toast } from 'sonner';

export const useBookings = (filters: BookingFilters) => {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 10,
    totalPages: 0,
    totalItems: 0,
  });

  const fetchBookings = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await bookingApi.getBookings(filters);
      setBookings(response.data.data);
      setPagination(response.data.pagination);
    } catch (err: any) {
      setError(err.response?.data?.error?.message || 'Failed to fetch bookings');
      toast.error('Failed to load bookings');
    } finally {
      setLoading(false);
    }
  };

  const createBooking = async (bookingData: CreateBookingData) => {
    setLoading(true);
    try {
      const response = await bookingApi.createBooking(bookingData);
      toast.success('Booking created successfully!');
      fetchBookings(); // Refresh list
      return response.data.data;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error?.message || 'Failed to create booking';
      toast.error(errorMsg);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const cancelBooking = async (bookingId: string, reason: string) => {
    setLoading(true);
    try {
      await bookingApi.cancelBooking(bookingId, reason);
      toast.success('Booking cancelled successfully');
      fetchBookings(); // Refresh list
    } catch (err: any) {
      toast.error('Failed to cancel booking');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBookings();
  }, [filters.page, filters.status, filters.startDate, filters.endDate]);

  return {
    bookings,
    loading,
    error,
    pagination,
    createBooking,
    cancelBooking,
    refetch: fetchBookings,
  };
};
```

---

## 🧪 Testing Guidelines

### Backend Unit Test Example

```go
// service/booking_service_test.go
package service

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock repository
type MockBookingRepository struct {
    mock.Mock
}

func (m *MockBookingRepository) Create(booking *models.Booking) error {
    args := m.Called(booking)
    return args.Error(0)
}

func (m *MockBookingRepository) CheckConflict(providerID string, date time.Time, startTime, endTime time.Time) (bool, error) {
    args := m.Called(providerID, date, startTime, endTime)
    return args.Bool(0), args.Error(1)
}

func TestCreateBooking_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockBookingRepository)
    mockServiceRepo := new(MockServiceRepository)
    mockCache := &redis.Client{} // Use mock or testcontainer
    
    service := NewBookingService(mockRepo, nil, mockServiceRepo, nil, mockCache)
    
    serviceData := &models.Service{
        ID:       "service-1",
        Duration: 60,
    }
    
    mockServiceRepo.On("FindByID", "service-1").Return(serviceData, nil)
    mockRepo.On("CheckConflict", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
    mockRepo.On("Create", mock.AnythingOfType("*models.Booking")).Return(nil)
    
    req := &dto.CreateBookingRequest{
        ProviderID:   "provider-1",
        ServiceID:    "service-1",
        BookingDate:  time.Now().AddDate(0, 0, 1),
        StartTime:    time.Date(0, 0, 0, 10, 0, 0, 0, time.UTC),
    }
    
    // Act
    booking, err := service.CreateBooking(req, "customer-1")
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, booking)
    assert.Equal(t, "pending", booking.Status)
    mockRepo.AssertExpectations(t)
}

func TestCreateBooking_ConflictError(t *testing.T) {
    // Similar setup but CheckConflict returns true
    mockRepo := new(MockBookingRepository)
    mockRepo.On("CheckConflict", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
    
    // ... rest of test
    
    booking, err := service.CreateBooking(req, "customer-1")
    
    assert.Error(t, err)
    assert.Nil(t, booking)
    assert.Contains(t, err.Error(), "not available")
}
```

### Frontend Component Test Example

```typescript
// components/booking/__tests__/BookingForm.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BookingForm } from '../BookingForm';
import { vi } from 'vitest';

describe('BookingForm', () => {
  const mockOnSubmit = vi.fn();
  const mockProviders = [
    { id: '1', business_name: 'Test Provider' }
  ];

  it('renders form fields correctly', () => {
    render(<BookingForm providers={mockProviders} onSubmit={mockOnSubmit} />);
    
    expect(screen.getByLabelText(/select provider/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/select date/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /book now/i })).toBeInTheDocument();
  });

  it('validates required fields', async () => {
    render(<BookingForm providers={mockProviders} onSubmit={mockOnSubmit} />);
    
    const submitButton = screen.getByRole('button', { name: /book now/i });
    fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText(/provider is required/i)).toBeInTheDocument();
      expect(screen.getByText(/date is required/i)).toBeInTheDocument();
    });
    
    expect(mockOnSubmit).not.toHaveBeenCalled();
  });

  it('submits form with valid data', async () => {
    render(<BookingForm providers={mockProviders} onSubmit={mockOnSubmit} />);
    
    // Fill form...
    const providerSelect = screen.getByLabelText(/select provider/i);
    fireEvent.change(providerSelect, { target: { value: '1' } });
    
    const submitButton = screen.getByRole('button', { name: /book now/i });
    fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(mockOnSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          provider_id: '1',
        })
      );
    });
  });
});
```

---

## 🚀 Deployment Instructions

### 1. Build for Production

```bash
# Backend
cd backend
CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Frontend
cd frontend
npm run build
# Output will be in dist/
```

### 2. Deploy to VPS (Example: DigitalOcean)

```bash
# 1. Setup server
ssh root@your-server-ip
apt update && apt upgrade -y
apt install docker.io docker-compose nginx certbot python3-certbot-nginx -y

# 2. Clone repository
git clone https://github.com/yourusername/booking-system.git
cd booking-system

# 3. Setup environment variables
cp .env.example .env
nano .env  # Edit with production values

# 4. Run with Docker Compose
docker-compose -f docker-compose.prod.yml up -d

# 5. Setup Nginx reverse proxy
nano /etc/nginx/sites-available/booking-system
# Add configuration (see below)

ln -s /etc/nginx/sites-available/booking-system /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx

# 6. Setup SSL with Let's Encrypt
certbot --nginx -d yourdomain.com -d api.yourdomain.com
```

### 3. Nginx Configuration

```nginx
# /etc/nginx/sites-available/booking-system

# Frontend
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;
    
    root /var/www/booking-system/dist;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
}

# Backend API
server {
    listen 80;
    server_name api.yourdomain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

---

## 📚 Additional Resources & Best Practices

### Error Codes Convention
```
AUTH_001: Invalid credentials
AUTH_002: Token expired
AUTH_003: Insufficient permissions

BOOKING_001: Slot not available
BOOKING_002: Booking not found
BOOKING_003: Cannot cancel confirmed booking

PAYMENT_001: Payment failed
PAYMENT_002: Invalid payment method

VALIDATION_001: Invalid input
VALIDATION_002: Missing required fields
```

### Logging Best Practices
```go
// Use structured logging
logger.Info("Booking created",
    zap.String("booking_id", booking.ID),
    zap.String("customer_id", customerID),
    zap.String("provider_id", providerID),
    zap.Time("booking_date", booking.BookingDate),
)

logger.Error("Failed to create booking",
    zap.Error(err),
    zap.String("customer_id", customerID),
    zap.Any("request", req),
)
```

### Database Migration Strategy
```bash
# Use migration tool like golang-migrate
migrate create -ext sql -dir database/migrations -seq create_users_table
migrate create -ext sql -dir database/migrations -seq add_avatar_to_users

# Run migrations
migrate -path database/migrations -database "postgresql://user:pass@localhost/dbname?sslmode=disable" up

# Rollback
migrate -path database/migrations -database "postgresql://user:pass@localhost/dbname?sslmode=disable" down 1
```

---

## 🎯 Success Criteria

Your implementation is complete when:

1. ✅ All API endpoints are functional and return correct responses
2. ✅ Authentication and authorization work properly
3. ✅ Booking system correctly handles slot availability and conflicts
4. ✅ Email notifications are sent for all key events
5. ✅ Payment integration works in sandbox mode
6. ✅ Admin dashboard displays analytics correctly
7. ✅ Frontend is responsive and works on mobile devices
8. ✅ All forms have proper validation and error handling
9. ✅ Code is well-documented with comments
10. ✅ Docker Compose setup works for local development
11. ✅ README contains clear setup and API documentation
12. ✅ At least 60% test coverage for critical business logic

---

## 💡 Tips for Implementation

1. **Start with authentication** - Get user management working first
2. **Build incrementally** - Don't try to implement everything at once
3. **Test as you go** - Write tests for critical features immediately
4. **Use git branches** - Feature branches with descriptive names
5. **Commit often** - Small, focused commits with clear messages
6. **Document APIs** - Keep Swagger/OpenAPI docs updated
7. **Handle errors gracefully** - Never expose internal errors to users
8. **Validate all inputs** - Both frontend and backend validation
9. **Think about edge cases** - What happens when slots are fully booked?
10. **Performance matters** - Use caching for frequently accessed data

---

## 📞 Support & Troubleshooting

Common issues and solutions:

**Issue**: PostgreSQL connection refused
**Solution**: Check if PostgreSQL is running, verify credentials in .env

**Issue**: Redis connection timeout
**Solution**: Ensure Redis is running: `docker-compose up redis`

**Issue**: CORS errors in frontend
**Solution**: Add frontend URL to ALLOWED_ORIGINS in backend .env

**Issue**: JWT token keeps expiring
**Solution**: Implement token refresh logic properly, check expiry times

**Issue**: Email not sending
**Solution**: Verify SendGrid API key, check from email is verified

**Issue**: Payment callback not working
**Solution**: Make sure callback URL is publicly accessible (use ngrok for local testing)

**Issue**: Time zone issues with bookings
**Solution**: Store all times in UTC, convert to local time only in frontend

**Issue**: Concurrent booking conflicts
**Solution**: Ensure distributed lock (Redis) is working correctly

---

## 🎨 UI/UX Guidelines

### Design Principles
1. **Mobile-First**: Design for mobile, enhance for desktop
2. **Accessibility**: WCAG 2.1 Level AA compliance
3. **Performance**: Page load under 3 seconds
4. **Consistency**: Use design system (shadcn/ui) consistently
5. **Feedback**: Always provide visual feedback for user actions

### Key User Flows

#### Customer Booking Flow
```
1. Browse Providers
   └─> Search/Filter by business type, location, availability
   
2. View Provider Details
   └─> See services, pricing, reviews, availability calendar
   
3. Select Service & Date
   └─> Choose from available time slots
   
4. Fill Booking Form
   └─> Enter personal details and special requests
   
5. Review & Confirm
   └─> Summary of booking details
   
6. Payment (if required)
   └─> Midtrans payment gateway
   
7. Confirmation
   └─> Email confirmation sent
   └─> Add to calendar option
```

#### Provider Management Flow
```
1. Dashboard Overview
   └─> Today's appointments, upcoming bookings, statistics
   
2. Manage Schedule
   └─> Set weekly availability
   └─> Add exceptions (holidays, breaks)
   
3. Manage Services
   └─> Add/edit/remove services
   └─> Set pricing and duration
   
4. Review Bookings
   └─> Approve/reject pending bookings
   └─> View customer details
   └─> Add notes
   
5. Analytics
   └─> Revenue tracking
   └─> Booking trends
   └─> Customer insights
```

### Color Scheme Recommendations
```css
/* Light Mode */
--primary: #2563eb;        /* Blue */
--secondary: #7c3aed;      /* Purple */
--success: #10b981;        /* Green */
--warning: #f59e0b;        /* Amber */
--error: #ef4444;          /* Red */
--background: #ffffff;
--foreground: #1f2937;
--muted: #f3f4f6;

/* Dark Mode */
--primary: #3b82f6;
--secondary: #8b5cf6;
--background: #0f172a;
--foreground: #f1f5f9;
--muted: #1e293b;
```

---

## 🔒 Security Checklist

### Backend Security
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS prevention (sanitize output)
- [ ] CSRF protection for state-changing operations
- [ ] Rate limiting on authentication endpoints (5 attempts per 15 min)
- [ ] Password strength requirements (min 8 chars, mix of letters/numbers)
- [ ] Secure password hashing (bcrypt with cost 12)
- [ ] JWT with short expiry times
- [ ] Refresh token rotation
- [ ] HTTPS only in production
- [ ] CORS properly configured
- [ ] Sensitive data not logged
- [ ] API versioning for backward compatibility
- [ ] Dependency security scanning
- [ ] Environment variables for secrets (never commit)

### Frontend Security
- [ ] Input validation and sanitization
- [ ] XSS prevention (React automatically escapes)
- [ ] Secure token storage (httpOnly cookies for refresh token)
- [ ] No sensitive data in localStorage
- [ ] HTTPS only
- [ ] Content Security Policy headers
- [ ] Subresource Integrity for CDN resources
- [ ] Regular dependency updates

### Payment Security
- [ ] Never store credit card data
- [ ] Use Midtrans hosted payment page
- [ ] Verify webhook signatures
- [ ] Log all payment transactions
- [ ] Handle payment failures gracefully
- [ ] Implement idempotency for payment requests

---

## 📊 Monitoring & Observability

### Metrics to Track

#### Application Metrics
```go
// Example metrics to instrument
- Request count by endpoint
- Request duration by endpoint
- Error rate by endpoint and error type
- Database query duration
- Cache hit/miss ratio
- Background job processing time
- Email sending success/failure rate
```

#### Business Metrics
```
- Bookings created per day/week/month
- Booking confirmation rate
- Booking cancellation rate
- Average booking value
- Revenue per provider
- Most popular services
- Peak booking times
- Customer retention rate
```

#### Infrastructure Metrics
```
- CPU usage
- Memory usage
- Disk usage
- Network I/O
- Database connections
- Redis memory usage
- API response times
- Uptime percentage
```

### Logging Strategy

```go
// Log levels
DEBUG   - Development only, verbose information
INFO    - Normal operations (booking created, user logged in)
WARN    - Unusual but handled situations (payment retry, cache miss)
ERROR   - Errors that need attention (payment failed, email send failed)
FATAL   - Critical errors that stop the application

// Structured logging example
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "message": "Booking created successfully",
  "booking_id": "uuid-here",
  "customer_id": "uuid-here",
  "provider_id": "uuid-here",
  "service_name": "Haircut",
  "booking_date": "2024-01-20",
  "amount": 150000,
  "request_id": "req-uuid"
}
```

### Alerting Rules (Production)

```yaml
alerts:
  - name: high_error_rate
    condition: error_rate > 5% for 5 minutes
    severity: critical
    
  - name: slow_api_response
    condition: p95_response_time > 2000ms for 10 minutes
    severity: warning
    
  - name: database_connection_pool_exhausted
    condition: available_db_connections < 5
    severity: critical
    
  - name: payment_failure_spike
    condition: payment_failures > 10 in 5 minutes
    severity: critical
    
  - name: booking_creation_failure
    condition: booking_errors > 5 in 5 minutes
    severity: warning
```

---

## 🔄 CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/ci-cd.yml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  # Backend Tests
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Run tests
        working-directory: ./backend
        env:
          DB_HOST: localhost
          DB_PORT: 5432
          REDIS_HOST: localhost
        run: |
          go test -v -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out
          
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./backend/coverage.out

  # Frontend Tests
  frontend-test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: ./frontend/package-lock.json
          
      - name: Install dependencies
        working-directory: ./frontend
        run: npm ci
        
      - name: Run linter
        working-directory: ./frontend
        run: npm run lint
        
      - name: Run tests
        working-directory: ./frontend
        run: npm test -- --coverage
        
      - name: Build
        working-directory: ./frontend
        run: npm run build

  # Security Scan
  security-scan:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
          
      - name: Upload Trivy results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'

  # Deploy to Production
  deploy:
    needs: [backend-test, frontend-test]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Deploy to VPS
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USERNAME }}
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd /var/www/booking-system
            git pull origin main
            docker-compose -f docker-compose.prod.yml down
            docker-compose -f docker-compose.prod.yml up -d --build
            docker system prune -f
            
      - name: Health Check
        run: |
          sleep 30
          curl -f https://api.yourdomain.com/health || exit 1
```

---

## 🗃️ Database Optimization

### Indexing Strategy

```sql
-- Essential indexes for performance
CREATE INDEX idx_bookings_provider_date_status ON bookings(provider_id, booking_date, status);
CREATE INDEX idx_bookings_customer_status ON bookings(customer_id, status);
CREATE INDEX idx_bookings_created_at ON bookings(created_at DESC);

CREATE INDEX idx_users_email_verified ON users(email, is_verified);
CREATE INDEX idx_users_role ON users(role);

CREATE INDEX idx_services_provider_active ON services(provider_id, is_active);

CREATE INDEX idx_availability_schedules_provider_day ON availability_schedules(provider_id, day_of_week, is_active);

CREATE INDEX idx_notifications_user_status ON notifications(user_id, status, created_at DESC);

-- Composite index for common query
CREATE INDEX idx_bookings_filter ON bookings(status, booking_date, provider_id) 
  WHERE status NOT IN ('cancelled', 'completed');

-- Partial index for active providers
CREATE INDEX idx_active_providers ON service_providers(id) 
  WHERE is_active = true;
```

### Query Optimization Tips

```go
// Bad: N+1 query problem
bookings := GetAllBookings()
for _, booking := range bookings {
    provider := GetProvider(booking.ProviderID) // Separate query for each!
}

// Good: Use eager loading
bookings := db.Preload("Provider").Preload("Service").Preload("Customer").
    Find(&bookings).Error

// Use pagination for large datasets
func GetBookings(page, limit int) ([]Booking, error) {
    var bookings []Booking
    offset := (page - 1) * limit
    
    err := db.Offset(offset).Limit(limit).
        Order("booking_date DESC").
        Find(&bookings).Error
        
    return bookings, err
}

// Use database-level aggregation
type BookingStats struct {
    TotalBookings     int64
    ConfirmedBookings int64
    CancelledBookings int64
    TotalRevenue      float64
}

func GetStats(providerID string) (*BookingStats, error) {
    var stats BookingStats
    
    db.Model(&Booking{}).
        Select(`
            COUNT(*) as total_bookings,
            COUNT(*) FILTER (WHERE status = 'confirmed') as confirmed_bookings,
            COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_bookings,
            SUM(CASE WHEN status = 'confirmed' THEN services.price ELSE 0 END) as total_revenue
        `).
        Joins("LEFT JOIN services ON bookings.service_id = services.id").
        Where("bookings.provider_id = ?", providerID).
        Scan(&stats)
        
    return &stats, nil
}
```

### Database Maintenance

```sql
-- Regular maintenance tasks

-- 1. Update table statistics
ANALYZE bookings;
ANALYZE users;
ANALYZE service_providers;

-- 2. Rebuild indexes (monthly)
REINDEX TABLE bookings;

-- 3. Vacuum to reclaim space (weekly)
VACUUM ANALYZE bookings;

-- 4. Check for bloat
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 5. Archive old bookings (older than 2 years)
CREATE TABLE bookings_archive (LIKE bookings INCLUDING ALL);

INSERT INTO bookings_archive 
SELECT * FROM bookings 
WHERE booking_date < CURRENT_DATE - INTERVAL '2 years';

DELETE FROM bookings 
WHERE booking_date < CURRENT_DATE - INTERVAL '2 years';
```

---

## 📱 Mobile Responsiveness

### Breakpoints (Tailwind)
```javascript
// tailwind.config.js
module.exports = {
  theme: {
    screens: {
      'xs': '375px',   // Small phones
      'sm': '640px',   // Large phones
      'md': '768px',   // Tablets
      'lg': '1024px',  // Laptops
      'xl': '1280px',  // Desktops
      '2xl': '1536px', // Large desktops
    }
  }
}
```

### Responsive Design Patterns

```tsx
// Mobile-first calendar
<div className="w-full">
  {/* Mobile: Stack view */}
  <div className="md:hidden">
    <MobileCalendarView />
  </div>
  
  {/* Desktop: Full calendar */}
  <div className="hidden md:block">
    <FullCalendar />
  </div>
</div>

// Responsive booking form
<form className="space-y-4">
  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
    <FormField label="Date" />
    <FormField label="Time" />
  </div>
  
  <div className="flex flex-col md:flex-row gap-4">
    <Button className="w-full md:w-auto">Cancel</Button>
    <Button className="w-full md:w-auto">Book Now</Button>
  </div>
</form>

// Responsive navigation
<nav className="fixed bottom-0 md:top-0 left-0 right-0 bg-white shadow-lg">
  {/* Mobile: Bottom nav */}
  <div className="md:hidden flex justify-around p-4">
    <NavItem icon={<Home />} label="Home" />
    <NavItem icon={<Calendar />} label="Bookings" />
    <NavItem icon={<User />} label="Profile" />
  </div>
  
  {/* Desktop: Top nav */}
  <div className="hidden md:flex items-center justify-between px-8 py-4">
    <Logo />
    <DesktopMenu />
    <UserMenu />
  </div>
</nav>
```

---

## 🌐 Internationalization (i18n)

### Setup i18n

```typescript
// i18n/config.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import enTranslations from './locales/en.json';
import idTranslations from './locales/id.json';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: enTranslations },
      id: { translation: idTranslations },
    },
    fallbackLng: 'id',
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
```

### Translation Files

```json
// i18n/locales/en.json
{
  "nav": {
    "home": "Home",
    "bookings": "My Bookings",
    "providers": "Find Providers"
  },
  "booking": {
    "create": "Create Booking",
    "select_date": "Select Date",
    "select_time": "Select Time",
    "confirm": "Confirm Booking",
    "success": "Booking created successfully!",
    "error": "Failed to create booking. Please try again."
  },
  "status": {
    "pending": "Pending",
    "confirmed": "Confirmed",
    "cancelled": "Cancelled",
    "completed": "Completed"
  }
}

// i18n/locales/id.json
{
  "nav": {
    "home": "Beranda",
    "bookings": "Pemesanan Saya",
    "providers": "Cari Layanan"
  },
  "booking": {
    "create": "Buat Pemesanan",
    "select_date": "Pilih Tanggal",
    "select_time": "Pilih Waktu",
    "confirm": "Konfirmasi Pemesanan",
    "success": "Pemesanan berhasil dibuat!",
    "error": "Gagal membuat pemesanan. Silakan coba lagi."
  },
  "status": {
    "pending": "Menunggu",
    "confirmed": "Dikonfirmasi",
    "cancelled": "Dibatalkan",
    "completed": "Selesai"
  }
}
```

### Usage in Components

```tsx
import { useTranslation } from 'react-i18next';

function BookingForm() {
  const { t, i18n } = useTranslation();
  
  return (
    <div>
      <h2>{t('booking.create')}</h2>
      
      <Button onClick={() => i18n.changeLanguage('id')}>
        Bahasa Indonesia
      </Button>
      <Button onClick={() => i18n.changeLanguage('en')}>
        English
      </Button>
      
      <form>
        <label>{t('booking.select_date')}</label>
        {/* ... */}
      </form>
    </div>
  );
}
```

---

## 🚀 Performance Optimization

### Backend Optimization

```go
// 1. Database Connection Pooling
func InitDB() *gorm.DB {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    
    sqlDB, _ := db.DB()
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    return db
}

// 2. Response Caching
func CacheMiddleware(duration time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        cacheKey := "cache:" + c.Request.URL.Path + "?" + c.Request.URL.RawQuery
        
        // Try to get from cache
        if cached, err := redis.Get(cacheKey); err == nil {
            c.Data(200, "application/json", []byte(cached))
            c.Abort()
            return
        }
        
        // Capture response
        w := &responseWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
        c.Writer = w
        
        c.Next()
        
        // Cache successful responses
        if c.Writer.Status() == 200 {
            redis.Set(cacheKey, w.body.Bytes(), duration)
        }
    }
}

// 3. Batch Operations
func CreateNotifications(notifications []Notification) error {
    // Instead of individual inserts
    return db.CreateInBatches(notifications, 100).Error
}

// 4. Select Only Needed Fields
func GetBookingsList() ([]BookingDTO, error) {
    var bookings []BookingDTO
    
    err := db.Model(&Booking{}).
        Select("bookings.id, bookings.booking_date, bookings.status, "+
               "providers.business_name, services.name as service_name").
        Joins("LEFT JOIN service_providers providers ON bookings.provider_id = providers.id").
        Joins("LEFT JOIN services ON bookings.service_id = services.id").
        Find(&bookings).Error
        
    return bookings, err
}

// 5. Background Job Queue (for heavy operations)
func ProcessBookingReminders() {
    tomorrow := time.Now().AddDate(0, 0, 1)
    
    bookings := GetBookingsForDate(tomorrow)
    
    for _, booking := range bookings {
        // Queue email job instead of sending synchronously
        jobQueue.Enqueue(SendReminderJob{
            BookingID: booking.ID,
            Email:     booking.Customer.Email,
        })
    }
}
```

### Frontend Optimization

```typescript
// 1. Code Splitting
import { lazy, Suspense } from 'react';

const AdminDashboard = lazy(() => import('./pages/dashboard/AdminDashboard'));
const ProviderDashboard = lazy(() => import('./pages/dashboard/ProviderDashboard'));

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/admin" element={<AdminDashboard />} />
        <Route path="/provider" element={<ProviderDashboard />} />
      </Routes>
    </Suspense>
  );
}

// 2. Debouncing Search
import { useDebounce } from '@/hooks/useDebounce';

function ProviderSearch() {
  const [searchTerm, setSearchTerm] = useState('');
  const debouncedSearch = useDebounce(searchTerm, 500);
  
  useEffect(() => {
    if (debouncedSearch) {
      searchProviders(debouncedSearch);
    }
  }, [debouncedSearch]);
  
  return <input onChange={(e) => setSearchTerm(e.target.value)} />;
}

// 3. Virtualization for Long Lists
import { FixedSizeList } from 'react-window';

function BookingList({ bookings }) {
  const Row = ({ index, style }) => (
    <div style={style}>
      <BookingCard booking={bookings[index]} />
    </div>
  );
  
  return (
    <FixedSizeList
      height={600}
      itemCount={bookings.length}
      itemSize={120}
      width="100%"
    >
      {Row}
    </FixedSizeList>
  );
}

// 4. Memoization
import { useMemo } from 'react';

function BookingStats({ bookings }) {
  const stats = useMemo(() => {
    return {
      total: bookings.length,
      confirmed: bookings.filter(b => b.status === 'confirmed').length,
      pending: bookings.filter(b => b.status === 'pending').length,
    };
  }, [bookings]);
  
  return <StatsDisplay stats={stats} />;
}

// 5. Image Optimization
<img 
  src={provider.avatar_url}
  alt={provider.business_name}
  loading="lazy"
  className="w-20 h-20 object-cover rounded-full"
  srcSet={`${provider.avatar_url}?w=80 1x, ${provider.avatar_url}?w=160 2x`}
/>
```

---

## 📖 API Documentation Template

### Swagger/OpenAPI Example

```yaml
openapi: 3.0.0
info:
  title: Booking System API
  version: 1.0.0
  description: API untuk platform pemesanan janji temu

servers:
  - url: http://localhost:8080/api/v1
    description: Development server
  - url: https://api.bookingsystem.com/api/v1
    description: Production server

paths:
  /auth/login:
    post:
      tags:
        - Authentication
      summary: User login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                email:
                  type: string
                  format: email
                password:
                  type: string
                  format: password
      responses:
        '200':
          description: Login successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                  data:
                    type: object
                    properties:
                      user:
                        $ref: '#/components/schemas/User'
                      access_token:
                        type: string
                      refresh_token:
                        type: string
        '401':
          description: Invalid credentials
          
  /bookings:
    post:
      tags:
        - Bookings
      summary: Create new booking
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateBookingRequest'
      responses:
        '201':
          description: Booking created
        '400':
          description: Validation error
        '409':
          description: Slot not available

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
        full_name:
          type: string
        role:
          type: string
          enum: [customer, provider, admin]
          
    CreateBookingRequest:
      type: object
      required:
        - provider_id
        - service_id
        - booking_date
        - start_time
      properties:
        provider_id:
          type: string
          format: uuid
        service_id:
          type: string
          format: uuid
        booking_date:
          type: string
          format: date
        start_time:
          type: string
          format: time
        customer_notes:
          type: string
```

---

## 🎓 Learning Resources

### Go (Backend)
- Official Go Tour: https://go.dev/tour/
- Gin Framework: https://gin-gonic.com/docs/
- GORM Documentation: https://gorm.io/docs/
- Go by Example: https://gobyexample.com/

### React (Frontend)
- React Official Docs: https://react.dev/
- TypeScript Handbook: https://www.typescriptlang.org/docs/
- Tailwind CSS: https://tailwindcss.com/docs
- shadcn/ui: https://ui.shadcn.com/

### Database & Caching
- PostgreSQL Tutorial: https://www.postgresqltutorial.com/
- Redis Documentation: https://redis.io/docs/
- Database Design Best Practices

### DevOps
- Docker Documentation: https://docs.docker.com/
- Docker Compose: https://docs.docker.com/compose/
- Nginx Configuration: https://nginx.org/en/docs/

---

## ✨ Final Notes

This is a comprehensive prompt that covers all aspects of building a production-ready booking system. Use this as your complete guide and reference throughout the development process.

**Remember**:
1. Build incrementally - don't try to implement everything at once
2. Test thoroughly at each stage
3. Keep code clean and well-documented
4. Follow security best practices
5. Prioritize user experience
6. Think about scalability from the start

**Good luck with your implementation!** 🚀

If you encounter any issues or need clarification on any part, refer back to the relevant sections in this document. Happy coding!
