# Graph Report - temu.in  (2026-08-09)

## Corpus Check
- 156 files · ~64,876 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1259 nodes · 1862 edges · 116 communities (100 shown, 16 thin omitted)
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 86 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `83bbfe62`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- index.ts
- Repository
- Service
- dependencies
- BindAndValidate
- Repository
- devDependencies
- ServiceProvider
- compilerOptions
- Service
- Review
- Repository
- payment/provider.go
- Temu.in
- Favorite
- What You Must Do When Invoked
- HomePage.tsx
- Repository
- compilerOptions
- Handlers
- Booking
- Repository
- toast.tsx
- What You Must Do When Invoked
- 000001_initial_schema.up.sql
- Handlers
- ServicesPage.tsx
- mockServiceWorker.js
- health/handler.go
- AdminAudit
- User
- RegisterRoutes
- tsconfig.json
- temu.in/backend
- Master Prompt: Platform Pemesanan Janji Temu (Booking System)
- Router
- DB
- LoginForm.tsx
- tsconfig.vitest.json
- graphify reference: extra exports and benchmark
- Server
- graphify reference: extra exports and benchmark
- App.tsx
- 🔌 API Endpoints Specification
- 📝 Code Examples & Patterns
- Metrics to Track
- Repository
- graphify reference: query, path, explain
- Notification
- graphify reference: query, path, explain
- 🎨 UI/UX Guidelines
- ✅ Development Checklist
- Frontend Temu.in
- Payment
- 🎓 Learning Resources
- graphify reference: add a URL and watch a folder
- graphify reference: commit hook and native CLAUDE.md integration
- graphify reference: incremental update and cluster-only
- graphify reference: add a URL and watch a folder
- graphify reference: commit hook and native CLAUDE.md integration
- graphify reference: incremental update and cluster-only
- 🧮 Core Business Logic
- 🎨 Frontend Key Features Implementation
- 🚀 Deployment Instructions
- 📚 Additional Resources & Best Practices
- 📋 Tech Stack Requirements
- 🐳 Docker Configuration
- 🔒 Security Checklist
- 🗃️ Database Optimization
- 🌐 Internationalization (i18n)
- graphify reference: GitHub clone and cross-repo merge
- graphify reference: transcribe video and audio
- graphify reference: GitHub clone and cross-repo merge
- graphify reference: transcribe video and audio
- 🔧 Environment Variables
- 🏗️ Project Structure
- 🧪 Testing Guidelines
- 📱 Mobile Responsiveness
- 💳 Payment Integration (Midtrans)
- AGENTS.md
- rules/graphify.md
- .agents/skills/graphify/references/extraction-spec.md
- workflows/graphify.md
- time_slot.go
- .codex/skills/graphify/references/extraction-spec.md
- AdminPage.tsx
- policyContext
- worker_test.go
- Worker
- EmailMessage
- NewSendGridProvider
- RenderNotification
- adminPromote.test.tsx
- AvailabilityException
- notifications
- Service
- payment_webhook_events
- payments

## God Nodes (most connected - your core abstractions)
1. `Master Prompt: Platform Pemesanan Janji Temu (Booking System)` - 32 edges
2. `Booking` - 26 edges
3. `DB` - 26 edges
4. `ServiceProvider` - 26 edges
5. `BindAndValidate()` - 23 edges
6. `Service` - 21 edges
7. `Router` - 20 edges
8. `Handlers` - 18 edges
9. `compilerOptions` - 18 edges
10. `Handlers` - 17 edges

## Surprising Connections (you probably didn't know these)
- `TestRequireProviderAccess()` --calls--> `RequireProviderAccess()`  [INFERRED]
  backend/internal/auth/policy_test.go → backend/internal/auth/policy.go
- `main()` --calls--> `Load()`  [INFERRED]
  backend/cmd/migrate/main.go → backend/internal/config/config.go
- `New()` --calls--> `NewSendGridProvider()`  [INFERRED]
  backend/internal/router/router.go → backend/internal/notification/provider.go
- `TestWorkerDeadLettersAfterMaximumAttempts()` --calls--> `NewWorker()`  [INFERRED]
  backend/internal/notification/worker_test.go → backend/internal/notification/worker.go
- `TestWorkerMarksSuccessfulDelivery()` --calls--> `NewWorker()`  [INFERRED]
  backend/internal/notification/worker_test.go → backend/internal/notification/worker.go

## Import Cycles
- None detected.

## Communities (116 total, 16 thin omitted)

### Community 0 - "index.ts"
Cohesion: 0.13
Nodes (15): AdminAudit, api, BookingInput, BookingResponse, confirmPayment(), createBooking(), failedQueue, fetchAvailabilityForDate() (+7 more)

### Community 1 - "Repository"
Cohesion: 0.21
Nodes (7): DeletedAt, Time, Client, NewRepository(), NewRepositoryWithCache(), RefreshToken, Repository

### Community 2 - "Service"
Cohesion: 0.09
Nodes (33): Claims, Context, Repository, RouterGroup, Service, ServiceCatalog, Time, TimeSlot (+25 more)

### Community 3 - "dependencies"
Cohesion: 0.04
Nodes (46): axios, class-variance-authority, clsx, framer-motion, dependencies, axios, class-variance-authority, clsx (+38 more)

### Community 4 - "BindAndValidate"
Cohesion: 0.05
Nodes (36): ProviderLookup, CreateExceptionRequest, CreateScheduleRequest, Handlers, ServiceCatalog, ClaimsFromContext(), Claims, Context (+28 more)

### Community 5 - "Repository"
Cohesion: 0.08
Nodes (24): BookingConflict, Repository, RepositoryInterface, Service, calculateSlots(), AvailabilityException, AvailabilitySchedule, DB (+16 more)

### Community 6 - "devDependencies"
Cohesion: 0.04
Nodes (47): autoprefixer, eslint, eslint-config-prettier, eslint-plugin-react-hooks, eslint-plugin-react-refresh, devDependencies, autoprefixer, eslint (+39 more)

### Community 7 - "ServiceProvider"
Cohesion: 0.14
Nodes (7): Time, User, NewRepository(), ServiceProvider, fakeRepository, ProviderFilters, Repository

### Community 8 - "compilerOptions"
Cohesion: 0.08
Nodes (25): compilerOptions, allowJs, allowSyntheticDefaultImports, baseUrl, esModuleInterop, forceConsistentCasingInFileNames, isolatedModules, jsx (+17 more)

### Community 9 - "Service"
Cohesion: 0.19
Nodes (7): Repository, NewService(), T, TestUpdateProviderRequiresOwnerUnlessAdmin(), CreateProviderRequest, Service, UpdateProviderRequest

### Community 10 - "Review"
Cohesion: 0.12
Nodes (11): DeletedAt, Time, User, NewRepository(), Repository, NewService(), Review, CreateReviewRequest (+3 more)

### Community 11 - "Repository"
Cohesion: 0.12
Nodes (9): Service, NewRepository(), Repository, NewService(), CreateServiceRequest, Repository, Service, ServiceFilter (+1 more)

### Community 12 - "payment/provider.go"
Cohesion: 0.11
Nodes (20): Client, Context, normalizeProviderStatus(), NormalizeWebhookStatus(), T, TestMidtransWebhookSignature(), TestNewOrderIDIsRandomAndUnique(), TestNormalizeWebhookStatus() (+12 more)

### Community 13 - "Temu.in"
Cohesion: 0.06
Nodes (34): 1. Payment production path — selesai untuk jalur aplikasi, 2. Notification delivery — selesai untuk outbox dan worker, 3. QA dan keamanan — P1, 4. API contract dan operasional — P1, 5. Performance dan product completion — P2, Backend, Frontend, Infrastruktur dan database (+26 more)

### Community 14 - "Favorite"
Cohesion: 0.13
Nodes (10): NewRepository(), Repository, NewService(), Service, Time, User, Repository, Service (+2 more)

### Community 15 - "What You Must Do When Invoked"
Cohesion: 0.08
Nodes (24): For /graphify add and --watch, For /graphify query, For the commit hook and native CLAUDE.md integration, For --update and --cluster-only, /graphify, Honesty Rules, Interpreter guard for subcommands, Part A - Structural extraction for code files (+16 more)

### Community 16 - "HomePage.tsx"
Cohesion: 0.14
Nodes (12): FeatureCards(), HeroSection(), DATES, InteractiveBookingSimulator(), ServiceOption, SERVICES, TIME_SLOTS, STATS (+4 more)

### Community 17 - "Repository"
Cohesion: 0.17
Nodes (3): User, NewRepository(), Repository

### Community 18 - "compilerOptions"
Cohesion: 0.13
Nodes (14): compilerOptions, allowSyntheticDefaultImports, composite, lib, module, moduleResolution, target, types (+6 more)

### Community 19 - "Handlers"
Cohesion: 0.07
Nodes (23): Handler, promoteReq, Claims, Context, Repository, RouterGroup, NewHandler(), ParseToken() (+15 more)

### Community 20 - "Booking"
Cohesion: 0.10
Nodes (24): DB, Time, NewRepository(), CanTransitionBookingStatus(), DeletedAt, ServiceProvider, Time, User (+16 more)

### Community 21 - "Repository"
Cohesion: 0.33
Nodes (5): Context, DB, Time, NewRepository(), Repository

### Community 22 - "toast.tsx"
Cohesion: 0.32
Nodes (9): AnimatedToast(), messages, setLocale(), t(), error(), info(), show(), success() (+1 more)

### Community 23 - "What You Must Do When Invoked"
Cohesion: 0.08
Nodes (24): For /graphify add and --watch, For /graphify query, For the commit hook and native CLAUDE.md integration, For --update and --cluster-only, /graphify, Honesty Rules, Interpreter guard for subcommands, Part A - Structural extraction for code files (+16 more)

### Community 24 - "000001_initial_schema.up.sql"
Cohesion: 0.42
Nodes (10): admin_audits, availability_exceptions, availability_schedules, bookings, payments, refresh_tokens, reviews, service_providers (+2 more)

### Community 25 - "Handlers"
Cohesion: 0.18
Nodes (14): Claims, Context, Repository, RouterGroup, Service, NewHandlers(), NewHandlersWithDependencies(), NewHandlersWithFinalizer() (+6 more)

### Community 26 - "ServicesPage.tsx"
Cohesion: 0.21
Nodes (7): fetchProviders(), fetchServices(), Service, BookingFlow(), ProvidersList(), Spinner(), ServicesPage()

### Community 27 - "mockServiceWorker.js"
Cohesion: 0.46
Nodes (7): activeClientIds, getResponse(), handleRequest(), resolveMainClient(), respondWithMock(), sendToClient(), sleep()

### Community 28 - "health/handler.go"
Cohesion: 0.67
Nodes (3): Context, liveness(), readiness()

### Community 29 - "AdminAudit"
Cohesion: 0.50
Nodes (3): DeletedAt, Time, AdminAudit

### Community 30 - "User"
Cohesion: 0.50
Nodes (3): DeletedAt, Time, User

### Community 50 - "Master Prompt: Platform Pemesanan Janji Temu (Booking System)"
Cohesion: 0.11
Nodes (19): 📖 API Documentation Template, 🔐 Authentication & Authorization Flow, Authorization Rules, Backend Optimization, 🔄 CI/CD Pipeline, 🗄️ Database Schema, ✨ Final Notes, Frontend Optimization (+11 more)

### Community 51 - "Router"
Cohesion: 0.06
Nodes (31): forgotPasswordReq, Handler, loginReq, registerReq, resetPasswordReq, verifyEmailReq, main(), generateSecureToken() (+23 more)

### Community 52 - "DB"
Cohesion: 0.19
Nodes (4): main(), Connect(), SeedAdmin(), DB

### Community 53 - "LoginForm.tsx"
Cohesion: 0.23
Nodes (10): clearAuthToken(), getAuthToken(), login(), logout(), me(), LoginForm(), RequireAuth(), AppState (+2 more)

### Community 54 - "tsconfig.vitest.json"
Cohesion: 0.18
Nodes (10): compilerOptions, types, extends, include, src, @testing-library/jest-dom, vitest/globals, src/**/*.test.ts (+2 more)

### Community 55 - "graphify reference: extra exports and benchmark"
Cohesion: 0.22
Nodes (8): graphify reference: extra exports and benchmark, Step 6b - Wiki (only if --wiki flag), Step 7 - Neo4j export (only if --neo4j or --neo4j-push flag), Step 7a - FalkorDB export (only if --falkordb or --falkordb-push flag), Step 7b - SVG export (only if --svg flag), Step 7c - GraphML export (only if --graphml flag), Step 7d - MCP server (only if --mcp flag), Step 8 - Token reduction benchmark (only if total_words > 5000)

### Community 56 - "Server"
Cohesion: 0.18
Nodes (11): Client, Duration, HandlerFunc, NewRateLimiter(), RateLimitByRoute(), Client, DB, Engine (+3 more)

### Community 57 - "graphify reference: extra exports and benchmark"
Cohesion: 0.22
Nodes (8): graphify reference: extra exports and benchmark, Step 6b - Wiki (only if --wiki flag), Step 7 - Neo4j export (only if --neo4j or --neo4j-push flag), Step 7a - FalkorDB export (only if --falkordb or --falkordb-push flag), Step 7b - SVG export (only if --svg flag), Step 7c - GraphML export (only if --graphml flag), Step 7d - MCP server (only if --mcp flag), Step 8 - Token reduction benchmark (only if total_words > 5000)

### Community 58 - "App.tsx"
Cohesion: 0.23
Nodes (4): App(), PageTransition(), prepare(), queryClient

### Community 59 - "🔌 API Endpoints Specification"
Cohesion: 0.25
Nodes (8): Admin Dashboard (`/api/v1/admin`), 🔌 API Endpoints Specification, Authentication (`/api/v1/auth`), Availability (`/api/v1/providers/:provider_id`), Bookings (`/api/v1/bookings`), Payments (`/api/v1/payments`), Providers (`/api/v1/providers`), Services (`/api/v1/providers/:provider_id/services`)

### Community 60 - "📝 Code Examples & Patterns"
Cohesion: 0.29
Nodes (7): 1. Standardized API Response Format, 2. Middleware Examples, 3. Repository Pattern Example, 4. Service Layer Example, 5. Frontend API Client with Interceptors, 6. React Hook for Bookings, 📝 Code Examples & Patterns

### Community 61 - "Metrics to Track"
Cohesion: 0.29
Nodes (7): Alerting Rules (Production), Application Metrics, Business Metrics, Infrastructure Metrics, Logging Strategy, Metrics to Track, 📊 Monitoring & Observability

### Community 62 - "Repository"
Cohesion: 0.47
Nodes (3): AdminAudit, Repository, NewRepository()

### Community 63 - "graphify reference: query, path, explain"
Cohesion: 0.33
Nodes (5): For /graphify explain, For /graphify path, graphify reference: query, path, explain, Step 0 — Constrained query expansion (REQUIRED before traversal), Step 1 — Traversal

### Community 64 - "Notification"
Cohesion: 0.26
Nodes (6): Time, Context, Time, Notification, NotificationDelivery, fakeOutboxRepository

### Community 65 - "graphify reference: query, path, explain"
Cohesion: 0.33
Nodes (5): For /graphify explain, For /graphify path, graphify reference: query, path, explain, Step 0 — Constrained query expansion (REQUIRED before traversal), Step 1 — Traversal

### Community 66 - "🎨 UI/UX Guidelines"
Cohesion: 0.33
Nodes (6): Color Scheme Recommendations, Customer Booking Flow, Design Principles, Key User Flows, Provider Management Flow, 🎨 UI/UX Guidelines

### Community 67 - "✅ Development Checklist"
Cohesion: 0.33
Nodes (6): ✅ Development Checklist, Phase 1: Foundation (Week 1-2), Phase 2: Core Booking System (Week 3-4), Phase 3: Notifications & Payments (Week 5), Phase 4: Admin Dashboard (Week 6), Phase 5: Polish & Testing (Week 7-8)

### Community 68 - "Frontend Temu.in"
Cohesion: 0.33
Nodes (5): Container, Frontend Temu.in, Menjalankan frontend, QA, Route aplikasi

### Community 69 - "Payment"
Cohesion: 0.18
Nodes (8): CanTransitionPaymentStatus(), DeletedAt, Time, DB, NewRepository(), Payment, PaymentWebhookEvent, Repository

### Community 70 - "🎓 Learning Resources"
Cohesion: 0.40
Nodes (5): Database & Caching, DevOps, Go (Backend), 🎓 Learning Resources, React (Frontend)

### Community 71 - "graphify reference: add a URL and watch a folder"
Cohesion: 0.50
Nodes (3): For /graphify add, For --watch, graphify reference: add a URL and watch a folder

### Community 72 - "graphify reference: commit hook and native CLAUDE.md integration"
Cohesion: 0.50
Nodes (3): For git commit hook, For native CLAUDE.md integration, graphify reference: commit hook and native CLAUDE.md integration

### Community 73 - "graphify reference: incremental update and cluster-only"
Cohesion: 0.50
Nodes (3): For --cluster-only, For --update (incremental re-extraction), graphify reference: incremental update and cluster-only

### Community 74 - "graphify reference: add a URL and watch a folder"
Cohesion: 0.50
Nodes (3): For /graphify add, For --watch, graphify reference: add a URL and watch a folder

### Community 75 - "graphify reference: commit hook and native CLAUDE.md integration"
Cohesion: 0.50
Nodes (3): For git commit hook, For native CLAUDE.md integration, graphify reference: commit hook and native CLAUDE.md integration

### Community 76 - "graphify reference: incremental update and cluster-only"
Cohesion: 0.50
Nodes (3): For --cluster-only, For --update (incremental re-extraction), graphify reference: incremental update and cluster-only

### Community 77 - "🧮 Core Business Logic"
Cohesion: 0.50
Nodes (4): 1. Available Slots Calculation Algorithm, 2. Booking Creation with Conflict Prevention, 3. Email Notification System, 🧮 Core Business Logic

### Community 78 - "🎨 Frontend Key Features Implementation"
Cohesion: 0.50
Nodes (4): 1. Booking Calendar Component, 2. Time Slot Picker, 3. Real-time Slot Updates (Optional Enhancement), 🎨 Frontend Key Features Implementation

### Community 79 - "🚀 Deployment Instructions"
Cohesion: 0.50
Nodes (4): 1. Build for Production, 2. Deploy to VPS (Example: DigitalOcean), 3. Nginx Configuration, 🚀 Deployment Instructions

### Community 80 - "📚 Additional Resources & Best Practices"
Cohesion: 0.50
Nodes (4): 📚 Additional Resources & Best Practices, Database Migration Strategy, Error Codes Convention, Logging Best Practices

### Community 81 - "📋 Tech Stack Requirements"
Cohesion: 0.50
Nodes (4): Backend, Frontend, Infrastructure, 📋 Tech Stack Requirements

### Community 82 - "🐳 Docker Configuration"
Cohesion: 0.50
Nodes (4): Backend Dockerfile, docker-compose.yml, 🐳 Docker Configuration, Frontend Dockerfile

### Community 83 - "🔒 Security Checklist"
Cohesion: 0.50
Nodes (4): Backend Security, Frontend Security, Payment Security, 🔒 Security Checklist

### Community 84 - "🗃️ Database Optimization"
Cohesion: 0.50
Nodes (4): Database Maintenance, 🗃️ Database Optimization, Indexing Strategy, Query Optimization Tips

### Community 85 - "🌐 Internationalization (i18n)"
Cohesion: 0.50
Nodes (4): 🌐 Internationalization (i18n), Setup i18n, Translation Files, Usage in Components

### Community 90 - "🔧 Environment Variables"
Cohesion: 0.67
Nodes (3): Backend (.env), 🔧 Environment Variables, Frontend (.env)

### Community 91 - "🏗️ Project Structure"
Cohesion: 0.67
Nodes (3): Backend (Go), Frontend (React), 🏗️ Project Structure

### Community 92 - "🧪 Testing Guidelines"
Cohesion: 0.67
Nodes (3): Backend Unit Test Example, Frontend Component Test Example, 🧪 Testing Guidelines

### Community 93 - "📱 Mobile Responsiveness"
Cohesion: 0.67
Nodes (3): Breakpoints (Tailwind), 📱 Mobile Responsiveness, Responsive Design Patterns

### Community 94 - "💳 Payment Integration (Midtrans)"
Cohesion: 0.67
Nodes (3): Implementation Flow, 💳 Payment Integration (Midtrans), Setup Steps

### Community 102 - "AdminPage.tsx"
Cohesion: 0.30
Nodes (9): adminListAudit(), adminListUsers(), adminPromoteUser(), AdminUser, Modal(), Props, usePromoteUser(), useUsers() (+1 more)

### Community 103 - "policyContext"
Cohesion: 0.24
Nodes (8): policyProviderLookup, Claims, Context, ServiceProvider, T, policyContext(), TestRequireProviderAccess(), ResponseRecorder

### Community 105 - "worker_test.go"
Cohesion: 0.29
Nodes (5): T, TestWorkerDeadLettersAfterMaximumAttempts(), TestWorkerMarksSuccessfulDelivery(), failingEmailSender, successfulEmailSender

### Community 106 - "Worker"
Cohesion: 0.22
Nodes (10): Service, Context, Duration, Time, NewWorker(), OutboxRepository, PaymentReceiptService, ReminderScheduler (+2 more)

### Community 107 - "EmailMessage"
Cohesion: 0.33
Nodes (5): Client, Context, EmailMessage, EmailSender, SendGridProvider

### Community 108 - "NewSendGridProvider"
Cohesion: 0.31
Nodes (7): NewSendGridProvider(), T, TestSendGridProviderRequiresConfiguration(), TestSendGridProviderSendsDynamicTemplate(), roundTripFunc, Request, Response

### Community 109 - "RenderNotification"
Cohesion: 0.47
Nodes (4): RenderNotification(), T, TestRenderNotificationRejectsUnknownTemplate(), TestRenderNotificationUsesStoredPayloadAndEscapesHTML()

### Community 110 - "adminPromote.test.tsx"
Cohesion: 0.29
Nodes (5): adminHandlers, users, worker, handlers, server

### Community 111 - "AvailabilityException"
Cohesion: 0.29
Nodes (3): Time, AvailabilityException, AvailabilitySchedule

### Community 112 - "notifications"
Cohesion: 0.40
Nodes (5): notification_deliveries, notifications, bookings, service_providers, users

## Knowledge Gaps
- **328 isolated node(s):** `graphify`, `Usage`, `What graphify is for`, `Step 0 - GitHub repos and multi-path merge (only if a URL or several paths)`, `Step 1 - Ensure graphify is installed` (+323 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **16 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `BindAndValidate()` connect `BindAndValidate` to `Router`, `Service`, `Handlers`, `Handlers`?**
  _High betweenness centrality (0.099) - this node is a cross-community bridge._
- **Why does `DB` connect `DB` to `Repository`, `ServiceProvider`, `Review`, `Repository`, `Favorite`, `AvailabilityException`, `Repository`, `Service`, `Repository`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **What connects `graphify`, `Usage`, `What graphify is for` to the rest of the system?**
  _328 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `index.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.13157894736842105 - nodes in this community are weakly interconnected._
- **Should `Service` be split into smaller, more focused modules?**
  _Cohesion score 0.08896103896103896 - nodes in this community are weakly interconnected._
- **Should `dependencies` be split into smaller, more focused modules?**
  _Cohesion score 0.0425531914893617 - nodes in this community are weakly interconnected._
- **Should `BindAndValidate` be split into smaller, more focused modules?**
  _Cohesion score 0.05217391304347826 - nodes in this community are weakly interconnected._