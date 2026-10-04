# Graph Report - temu.in  (2026-10-04)

## Corpus Check
- 246 files · ~144,580 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 22 file(s) not represented in the graph (top: (none) 14, .example 3, .toml 2)

## Summary
- 2163 nodes · 7524 edges · 118 communities (95 shown, 23 thin omitted)
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 352 edges (avg confidence: 0.87)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `dc67a9cb`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- router.go
- 000001_initial_schema.up.sql
- graphify Skill (SKILL.md)
- Graphify Full Pipeline (Steps 0-9)
- time.Time
- ExceptionManager.tsx
- package.json
- Service
- 🔒 Security Checklist
- Service
- devDependencies
- Frontend README
- RequireProviderAccess
- Redis
- Router
- ServiceProvider
- Transaction
- App.tsx
- FeatureCards.tsx
- Review
- 🎨 UI/UX Guidelines
- compilerOptions
- Booking
- handlers.ts
- 🗃️ Database Optimization
- Master Prompt: Platform Pemesanan Janji Temu (Booking System)
- Temu.in README
- calculateSlots
- Payment
- newAuthEnv
- Handlers
- dependencies
- github.com/gin-gonic/gin.RouterGroup
- openapi_test.go
- gorm.io/gorm.DB
- Favorite
- booking/service_test.go
- react
- github.com/gin-gonic/gin.Context
- health/handler.go
- ratelimit_test.go
- Handlers
- Repository
- AvailabilitySchedule
- seededHandlers
- SlotCache
- Repository
- New
- index.ts
- toast.tsx
- Server
- Worker
- 🐳 Docker Configuration
- cachedService
- Service
- context.Context
- Metrics to Track
- compilerOptions
- testing.T
- mockServiceWorker.js
- newWebhookEnv
- TestOpenAPIResponseMismatchDetectedIntegration
- Laporan Progres Temu.in
- memRepo
- @vitejs/plugin-react-swc
- Handlers
- serveReady
- scripts
- github.com/gin-gonic/gin.Engine
- newPaymentIT
- tsconfig.vitest.json
- tsconfig.json
- AGENTS.md graphify usage rules
- temu.in/backend
- EmailMessage
- NewBookingFinalizer
- Handler
- AvailabilityException
- newTestEngine
- toScheduleDTO
- Checklist Implementasi
- contextHandler
- requestid_test.go
- 🔌 API Endpoints Specification
- ParseClock
- 📝 Code Examples & Patterns
- docker-compose.yml stack
- ✅ Development Checklist
- Rekomendasi pengembangan berikutnya
- accesslog_test.go
- 🎓 Learning Resources
- main.tsx
- FormatValidationErrors
- 🧮 Core Business Logic
- 🎨 Frontend Key Features Implementation
- 🚀 Deployment Instructions
- 📚 Additional Resources & Best Practices
- 📋 Tech Stack Requirements
- 🌐 Internationalization (i18n)
- eslint.config.js
- sqlStateErr
- 🔧 Environment Variables
- 🏗️ Project Structure
- 🧪 Testing Guidelines
- 📱 Mobile Responsiveness
- 💳 Payment Integration (Midtrans)
- coverage-check.sh
- serviceDeps
- validateFutureDate

## God Nodes (most connected - your core abstractions)
1. `Booking` - 74 edges
2. `ServiceProvider` - 59 edges
3. `User` - 44 edges
4. `Payment` - 38 edges
5. `Service` - 36 edges
6. `Master Prompt: Platform Pemesanan Janji Temu (Booking System)` - 32 edges
7. `CreateUser()` - 28 edges
8. `PageTransition()` - 28 edges
9. `AvailabilitySchedule` - 27 edges
10. `AvailabilityException` - 27 edges

## Surprising Connections (you probably didn't know these)
- `Dependency audit` --references--> `RequireAuth()`  [INFERRED]
  docs/progress-report.md → frontend/src/components/RequireAuth.tsx
- `Item 2 — Real readiness (track B; independent)` --references--> `error()`  [INFERRED]
  .plans/hardening-items-1-7.md → frontend/src/lib/toast.tsx
- `Frontend` --references--> `AppNavLink()`  [INFERRED]
  .plans/hardening-items-1-7.md → frontend/src/App.tsx
- `Item 4 — Missing frontend screens (track D; after step 14 for endpoint contracts)` --references--> `setAuthToken()`  [INFERRED]
  .plans/hardening-items-1-7.md → frontend/src/api/index.ts
- `Item 1 — Test the booking core (see execution order in Approach)` --references--> `Provider`  [INFERRED]
  .plans/hardening-items-1-7.md → frontend/src/api/index.ts

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI QA pipeline (backend job, frontend job, migration smoke test)** — github_workflows_ci_backend_job, github_workflows_ci_frontend_job, github_workflows_ci_migration_smoke_test [EXTRACTED 1.00]
- **Docker Compose runtime stack (postgres, redis, backend, frontend, nginx)** — docker_compose_postgres, docker_compose_redis, docker_compose_backend, docker_compose_frontend, docker_compose_nginx [EXTRACTED 1.00]
- **Extraction Fragment Contract** — _agents_skills_graphify_references_extraction_spec_file_type_constraint, _agents_skills_graphify_references_extraction_spec_node_id_format, _agents_skills_graphify_references_extraction_spec_confidence_score_rubric, _agents_skills_graphify_references_extraction_spec_source_file_rule, _agents_skills_graphify_references_extraction_spec_hyperedges [EXTRACTED 1.00]
- **graphify skill and its on-demand reference documents** — codex_skills_graphify_skill, codex_skills_graphify_references_add_watch, codex_skills_graphify_references_exports, codex_skills_graphify_references_extraction_spec, codex_skills_graphify_references_github_and_merge, codex_skills_graphify_references_hooks, codex_skills_graphify_references_query, codex_skills_graphify_references_transcribe, codex_skills_graphify_references_update [EXTRACTED 1.00]
- **Graph Query Interfaces** — _agents_skills_graphify_references_query_graphify_query_cli, _agents_skills_graphify_references_query_graphify_path_explain, _agents_skills_graphify_references_exports_mcp_server, _agents_rules_graphify_wiki_navigation, _agents_rules_graphify_graph_report [INFERRED 0.75]
- **Booking, payment, and notification outbox flow** — readme_booking_contract, readme_payment_flow, readme_notification_outbox, readme_overlap_constraint [INFERRED 0.85]
- **Mechanisms for Keeping Graph Current** — _agents_skills_graphify_references_update_incremental_update, _agents_skills_graphify_references_hooks_post_commit_hook, _agents_skills_graphify_references_add_watch_watch_mode, _agents_skills_graphify_references_update_ast_only_code_update, _agents_rules_graphify_graphify_update_after_code [INFERRED 0.85]
- **Mechanisms that keep the graph current (update, watch, hook, add)** — codex_skills_graphify_references_update_incremental, codex_skills_graphify_references_add_watch_watch, codex_skills_graphify_references_hooks_post_commit, codex_skills_graphify_references_add_watch_ingest [INFERRED 0.85]
- **Query, save-result and reflect self-improving loop** — codex_skills_graphify_references_query_expansion, codex_skills_graphify_references_query_save_result, codex_skills_graphify_references_query_reflect_lessons, codex_skills_graphify_references_update_incremental [INFERRED 0.85]

## Communities (118 total, 23 thin omitted)

### Community 0 - "router.go"
Cohesion: 0.13
Nodes (12): promoteReq, forgotPasswordReq, loginReq, registerReq, resetPasswordReq, verifyEmailReq, CreateExceptionRequest, cancelRequest (+4 more)

### Community 1 - "000001_initial_schema.up.sql"
Cohesion: 0.07
Nodes (52): admin_audits, availability_exceptions, availability_schedules, bookings, idx_admin_audits_actor_id, idx_availability_exceptions_date, idx_availability_exceptions_provider_id, idx_availability_schedules_day_of_week (+44 more)

### Community 2 - "graphify Skill (SKILL.md)"
Cohesion: 0.06
Nodes (45): Reference: add URL and watch folder, graphify add (URL ingest into ./raw), graphify --watch Debounced Auto-Rebuild, Reference: extra exports and benchmark, Token Reduction Benchmark, FalkorDB Export and Push, graphify MCP stdio Server, Neo4j Export and Push (+37 more)

### Community 3 - "Graphify Full Pipeline (Steps 0-9)"
Cohesion: 0.06
Nodes (43): Graphify Always-On Rule, GRAPH_REPORT.md for Broad Review, Query Graph First, Then Raw Files, Run graphify update After Code Changes, Navigate graphify-out/wiki/index.md, Reference: Add URL and Watch, graphify add URL Ingest, --watch Auto-rebuild Watcher (+35 more)

### Community 4 - "time.Time"
Cohesion: 0.09
Nodes (14): BookingConflict, AdminAudit, FormatClock(), FormatClockHM(), isEndOfDayClock(), TestFormatClock(), Notification, fakeAvailability (+6 more)

### Community 5 - "ExceptionManager.tsx"
Cohesion: 0.09
Nodes (47): BookingStatus, completeBooking(), confirmBooking(), createException(), createSchedule(), deleteException(), deleteSchedule(), fetchExceptions() (+39 more)

### Community 6 - "package.json"
Cohesion: 0.08
Nodes (24): msw, workerDirectory, name, private, type, version, autoprefixer, class-variance-authority (+16 more)

### Community 7 - "Service"
Cohesion: 0.11
Nodes (16): NewHandlers(), NewHandlersWithAvailability(), NewHandlersWithDependencies(), NewService(), NewServiceWithDeps(), NewServiceWithFullDeps(), AvailabilityService, bookingStore (+8 more)

### Community 8 - "🔒 Security Checklist"
Cohesion: 0.50
Nodes (4): Backend Security, Frontend Security, Payment Security, 🔒 Security Checklist

### Community 9 - "Service"
Cohesion: 0.08
Nodes (12): catalogMap, fakeServiceCatalog, ServiceCatalog, Service, fakeCatalog, fakeCatalog, CreateServiceRequest, fakeRepo (+4 more)

### Community 10 - "devDependencies"
Cohesion: 0.08
Nodes (25): devDependencies, autoprefixer, eslint, eslint-config-prettier, eslint-plugin-react-hooks, eslint-plugin-react-refresh, globals, jsdom (+17 more)

### Community 11 - "Frontend README"
Cohesion: 0.20
Nodes (10): frontend index.html (SPA entry), Frontend README, Frontend container: static build served by Nginx with /api proxy, npm audit: 23 advisories needing triage, MSW (Mock Service Worker) via VITE_USE_MSW, SPA routes (/, /providers, /services, /booking, /login, /admin), GitHub Actions CI Workflow, CI backend job (gofmt, vet, race test) (+2 more)

### Community 12 - "RequireProviderAccess"
Cohesion: 0.13
Nodes (13): ProviderLookup, Claims, ParseToken(), TestTokenRoundTrip(), ClaimsFromContext(), RequireProviderAccess(), policyContext(), TestRequireProviderAccess() (+5 more)

### Community 13 - "Redis"
Cohesion: 0.19
Nodes (15): NewSlotCache(), hungListener(), sampleSlots(), TestSlotCacheClosedRedisDegrades(), TestSlotCacheEmptySlotsRoundTrip(), TestSlotCacheHungRedisRespectsOpTimeout(), TestSlotCacheInvalidate(), TestSlotCacheMissThenHit() (+7 more)

### Community 14 - "Router"
Cohesion: 0.12
Nodes (24): NewHandler(), adminEngine(), call(), TestAdminHandlerIntegration(), TestPromoteRejectsBeforeRepository(), Repository, NewRepository(), Middleware() (+16 more)

### Community 15 - "ServiceProvider"
Cohesion: 0.08
Nodes (12): policyProviderLookup, fakeProviders, ServiceProvider, Service, fakeProviderSvc, CreateProviderRequest, fakeRepository, fakeService (+4 more)

### Community 16 - "Transaction"
Cohesion: 0.15
Nodes (9): normalizeProviderStatus(), NormalizeWebhookStatus(), TestNormalizeWebhookStatus(), CreateTransactionRequest, fakeProvider, MidtransProvider, Provider, Transaction (+1 more)

### Community 17 - "App.tsx"
Cohesion: 0.11
Nodes (36): clearAuthToken(), fetchMyProvider(), fetchServices(), forgotPassword(), logout(), register(), resetPassword(), verifyEmail() (+28 more)

### Community 18 - "FeatureCards.tsx"
Cohesion: 0.18
Nodes (14): FeatureCards(), TiltCard(), HeroSection(), DATES, InteractiveBookingSimulator(), ServiceOption, SERVICES, TIME_SLOTS (+6 more)

### Community 19 - "Review"
Cohesion: 0.08
Nodes (14): Review, engineFor(), piiIn(), reviewFixture(), TestProviderReviewsShape(), TestReviewHandlerErrors(), TestReviewResponsesExposeNoPII(), NewService() (+6 more)

### Community 20 - "🎨 UI/UX Guidelines"
Cohesion: 0.33
Nodes (6): Color Scheme Recommendations, Customer Booking Flow, Design Principles, Key User Flows, Provider Management Flow, 🎨 UI/UX Guidelines

### Community 21 - "compilerOptions"
Cohesion: 0.10
Nodes (19): compilerOptions, allowJs, allowSyntheticDefaultImports, baseUrl, esModuleInterop, forceConsistentCasingInFileNames, isolatedModules, jsx (+11 more)

### Community 22 - "Booking"
Cohesion: 0.07
Nodes (14): NewProviderBookingDTO(), newProviderBookingDTOs(), serviceName(), page(), TestClampPage(), Repository, clampPage(), isBookingStatus() (+6 more)

### Community 23 - "handlers.ts"
Cohesion: 0.07
Nodes (42): ApiResponse, ExceptionInput, ProviderProfile, adminHandlers, users, prepare(), worker, availabilityResolver() (+34 more)

### Community 24 - "🗃️ Database Optimization"
Cohesion: 0.50
Nodes (4): Database Maintenance, 🗃️ Database Optimization, Indexing Strategy, Query Optimization Tips

### Community 25 - "Master Prompt: Platform Pemesanan Janji Temu (Booking System)"
Cohesion: 0.11
Nodes (19): 📖 API Documentation Template, 🔐 Authentication & Authorization Flow, Authorization Rules, Backend Optimization, 🔄 CI/CD Pipeline, 🗄️ Database Schema, ✨ Final Notes, Frontend Optimization (+11 more)

### Community 26 - "Temu.in README"
Cohesion: 0.36
Nodes (7): Temu.in README, Canonical /api prefix (not /api/v1), Customer/provider/admin authorization policy, Booking API contract (POST /api/bookings with serviceId and slot), Notification outbox with SendGrid worker, PostgreSQL overlapping booking constraint, Midtrans payment intent, confirm, and signed webhook

### Community 27 - "calculateSlots"
Cohesion: 0.13
Nodes (14): calculateSlots(), parseClock(), assertSlots(), TestCalculateSlotsAppliesBlockedException(), TestCalculateSlotsEdgeCases(), TestCalculateSlotsFullDayBlock(), TestCalculateSlotsUsesServiceDuration(), TestParseClockLayouts() (+6 more)

### Community 28 - "Payment"
Cohesion: 0.13
Nodes (8): CanTransitionPaymentStatus(), Payment, TestCanTransitionPaymentStatus(), Repository, CreateRequest, fakePaymentCreator, fakeFinalizerPayments, fakeStore

### Community 29 - "newAuthEnv"
Cohesion: 0.15
Nodes (11): authEnv, authResponse, newAuthEnv(), TestForgotAndResetPassword(), TestRegisterLoginRefreshLogoutFlow(), TestRegisterValidation(), TestVerifyEmail(), RefreshToken (+3 more)

### Community 30 - "Handlers"
Cohesion: 0.16
Nodes (9): NewHandlers(), NewHandlersWithDependencies(), NewHandlersWithFinalizer(), serviceAmount(), TestServiceAmount(), BookingFinalizer, Handlers, paymentStore (+1 more)

### Community 31 - "dependencies"
Cohesion: 0.12
Nodes (17): dependencies, axios, class-variance-authority, clsx, framer-motion, @hookform/resolvers, lucide-react, @radix-ui/react-slot (+9 more)

### Community 32 - "github.com/gin-gonic/gin.RouterGroup"
Cohesion: 0.14
Nodes (5): NewPublicReview(), newPublicReviews(), NewHandlers(), Handlers, PublicReview

### Community 33 - "openapi_test.go"
Cohesion: 0.06
Nodes (35): run(), TestMain(), TestMain(), TestMain(), TestMain(), TestMain(), Connect(), DB (+27 more)

### Community 34 - "gorm.io/gorm.DB"
Cohesion: 0.13
Nodes (39): TestRecordAndListNewestFirst(), TestIntegrationAvailableFullDayExceptionRejected(), TestIntegrationFullDayExceptionRoundTrip(), TestIntegrationScheduleDTONormalisesDBClock(), TestIntegrationScheduleIsAvailableFalsePersists(), TestIntegrationExceptionsDateRangeInclusive(), TestIntegrationGetAvailableSlotsWithBlockingException(), TestIntegrationScheduleCRUDScopedByProvider() (+31 more)

### Community 35 - "Favorite"
Cohesion: 0.11
Nodes (9): NewService(), matches(), ptr(), TestToggleFavoriteAddsThenRemoves(), TestToggleFavoriteErrors(), Favorite, fakeRepo, Repository (+1 more)

### Community 36 - "booking/service_test.go"
Cohesion: 0.20
Nodes (23): newFakeStore(), TestBookingStoreImplementations(), TestServiceListProviderBookingsValidation(), newTestService(), seedBooking(), TestCancelBooking(), TestCompleteBooking(), TestConfirmBooking() (+15 more)

### Community 37 - "react"
Cohesion: 0.10
Nodes (29): adminListAudit(), adminListUsers(), adminPromoteUser(), AdminUser, createPaymentIntent(), fetchAvailability(), getAuthToken(), login() (+21 more)

### Community 38 - "github.com/gin-gonic/gin.Context"
Cohesion: 0.12
Nodes (7): Handler, CreateScheduleRequest, Handlers, liveness(), NewHandlers(), BindAndValidate(), Handlers

### Community 39 - "health/handler.go"
Cohesion: 0.22
Nodes (7): isNil(), CheckResult, Handler, namedCheck, Pinger, pingResult, ReadinessResponse

### Community 40 - "ratelimit_test.go"
Cohesion: 0.28
Nodes (11): NewRateLimiter(), RateLimitByRoute(), hit(), limitedEngine(), TestRateLimitByRoute(), TestRateLimiterFailsOpenWhenRedisIsDown(), TestRateLimiterNilCachePassesThrough(), TestRateLimiterUnderAndOverLimit() (+3 more)

### Community 41 - "Handlers"
Cohesion: 0.16
Nodes (9): NewCustomerBookingDTO(), newCustomerBookingDTOs(), NewHandlersWithService(), pageParams(), parseTimeSlot(), createBookingRequest, CustomerBookingDTO, Handlers (+1 more)

### Community 44 - "seededHandlers"
Cohesion: 0.27
Nodes (10): assertNoKeys(), clock(), doJSON(), doRescheduleJSON(), seededHandlers(), TestCustomerBookingResponsesUseDTO(), testEngine(), TestListProviderBookingsAuthorization() (+2 more)

### Community 45 - "SlotCache"
Cohesion: 0.27
Nodes (4): SlotCache, newVersionToken(), slotsKey(), versionKey()

### Community 47 - "New"
Cohesion: 0.09
Nodes (23): NewHandler(), Config, Load(), setRequired(), TestLoadDefaults(), TestLoadTrimsFrontendBaseURLTrailingSlash(), NewService(), NewServiceWithBaseURL() (+15 more)

### Community 48 - "index.ts"
Cohesion: 0.06
Nodes (60): AdminAudit, api, ApiErrorBody, AuthResponse, AuthUser, AvailabilityException, AvailabilityResponse, AvailabilitySchedule (+52 more)

### Community 49 - "toast.tsx"
Cohesion: 0.19
Nodes (18): cancelBooking(), fetchBooking(), getApiErrorStatus(), rescheduleBooking(), AnimatedToast(), Props, RescheduleDialog(), messages (+10 more)

### Community 50 - "Server"
Cohesion: 0.21
Nodes (6): applyDefaults(), NewClient(), NewClientFromOptions(), TestNewClientAppliesDefaults(), TestNewClientInvalidURL(), Server

### Community 51 - "Worker"
Cohesion: 0.13
Nodes (16): Worker, NewWorker(), redactEmails(), newLoggedWorker(), testNotification(), TestRedactEmails(), TestWorkerDeadLettersAfterMaximumAttempts(), TestWorkerFirstFailureLogsWarnOnly() (+8 more)

### Community 52 - "🐳 Docker Configuration"
Cohesion: 0.50
Nodes (4): Backend Dockerfile, docker-compose.yml, 🐳 Docker Configuration, Frontend Dockerfile

### Community 53 - "cachedService"
Cohesion: 0.19
Nodes (8): countingRepo, cachedService(), TestScheduleValidation(), TestServiceFailedMutationDoesNotInvalidate(), TestServiceGetAvailableSlotsRepoErrorNotCached(), TestServiceGetAvailableSlotsUsesCache(), TestServiceScheduleAndExceptionChangesInvalidate(), TestServiceWithoutCache()

### Community 54 - "Service"
Cohesion: 0.23
Nodes (5): Service, TestIsValidTimeRejectsInvalidHour(), isValidTime(), TestIsValidTime(), validateExceptionTimes()

### Community 55 - "context.Context"
Cohesion: 0.11
Nodes (14): User, addProviderData(), bookingData(), bookingID(), service, providerID(), userID(), Service (+6 more)

### Community 56 - "Metrics to Track"
Cohesion: 0.29
Nodes (7): Alerting Rules (Production), Application Metrics, Business Metrics, Infrastructure Metrics, Logging Strategy, Metrics to Track, 📊 Monitoring & Observability

### Community 57 - "compilerOptions"
Cohesion: 0.20
Nodes (9): compilerOptions, allowSyntheticDefaultImports, composite, lib, module, moduleResolution, target, types (+1 more)

### Community 58 - "testing.T"
Cohesion: 0.15
Nodes (32): createInput(), newITFixture(), newPayment(), stampOf(), TestIntegrationCheckConflict(), TestIntegrationCheckConflictNonUTCClock(), TestIntegrationCreateWithPayment(), TestIntegrationEndOfDayBooking() (+24 more)

### Community 59 - "mockServiceWorker.js"
Cohesion: 0.39
Nodes (7): activeClientIds, getResponse(), handleRequest(), resolveMainClient(), respondWithMock(), sendToClient(), sleep()

### Community 60 - "newWebhookEnv"
Cohesion: 0.11
Nodes (26): TestWebhookSucceedsWhenSlotLookupFails(), bearer(), decode(), do(), newEngine(), newFakeStore(), newIntentEnv(), newWebhookEnv() (+18 more)

### Community 61 - "TestOpenAPIResponseMismatchDetectedIntegration"
Cohesion: 0.24
Nodes (10): call(), mutateDateFormat(), newSpecValidator(), signedWebhook(), TestOpenAPIResponseFormatValidationIsActive(), TestOpenAPIResponseMismatchDetectedIntegration(), TestOpenAPIResponsesIntegration(), TestOpenAPIResponsesUnitMode() (+2 more)

### Community 62 - "Laporan Progres Temu.in"
Cohesion: 0.18
Nodes (11): Backend, Dependency audit, Frontend, Infrastruktur dan database, Laporan Progres Temu.in, Perintah verifikasi, Progres terverifikasi, QA dan dokumentasi (+3 more)

### Community 66 - "serveReady"
Cohesion: 0.18
Nodes (10): NewHandler(), serveReady(), TestLiveness(), TestReadinessAllOK(), TestReadinessBlockingPingerTimesOut(), TestReadinessCriticalFailureIs503WithoutErrorText(), TestReadinessNilDependenciesSkipped(), TestReadinessNonCriticalFailureIsDegraded() (+2 more)

### Community 67 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, gen:api, lint, msw:init, preview, test

### Community 68 - "github.com/gin-gonic/gin.Engine"
Cohesion: 0.14
Nodes (21): caller, RequireRole(), get(), mustToken(), protectedEngine(), TestMiddleware(), TestMiddlewareRejectsNoneAlgorithm(), TestRequireRole() (+13 more)

### Community 69 - "newPaymentIT"
Cohesion: 0.33
Nodes (8): count(), newPaymentIT(), rows(), TestRepositoryLookupsIntegration(), TestSettlementOnCancelledBookingIntegration(), TestUpdateProviderStatusNeverMovesBackwardsIntegration(), TestWebhookReplayIdempotenceIntegration(), paymentIT

### Community 70 - "tsconfig.vitest.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, ./tsconfig.app.json

### Community 82 - "EmailMessage"
Cohesion: 0.15
Nodes (8): RenderNotification(), TestRenderNotificationRejectsUnknownTemplate(), TestRenderNotificationUsesStoredPayloadAndEscapesHTML(), EmailMessage, failingEmailSender, leakyEmailSender, SendGridProvider, successfulEmailSender

### Community 83 - "NewBookingFinalizer"
Cohesion: 0.27
Nodes (8): invalidateBookingSlots(), NewBookingFinalizer(), TestBookingFinalizerErrors(), TestBookingFinalizerGating(), TestBookingFinalizerSlotInvalidation(), FinalizerBookings, FinalizerPayments, SlotInvalidator

### Community 84 - "Handler"
Cohesion: 0.35
Nodes (5): Handler, generateSecureToken(), hashToken(), NewToken(), TestTokenHelpers()

### Community 85 - "AvailabilityException"
Cohesion: 0.33
Nodes (3): stubRepo, TestExceptionValidation(), AvailabilityException

### Community 86 - "newTestEngine"
Cohesion: 0.36
Nodes (8): findKeys(), newTestEngine(), ownerFixture(), request(), TestMe(), TestMeDoesNotShadowProviderByID(), TestProviderListEnvelopeAndOwnerName(), TestProviderResponsesExposeNoPII()

### Community 87 - "toScheduleDTO"
Cohesion: 0.39
Nodes (9): ExceptionDTO, ScheduleDTO, clockHHMM(), nullableHHMM(), toExceptionDTO(), toExceptionDTOs(), toScheduleDTO(), toScheduleDTOs() (+1 more)

### Community 88 - "Checklist Implementasi"
Cohesion: 0.22
Nodes (9): Checklist Implementasi, Checklist Implementasi & Rencana Sprint, Developer notes — local dev & MSW, Issue 1 – Fondasi Infrastruktur & Lingkungan, Issue 2 – Backend API & Layanan Inti, Issue 3 – Frontend Aplikasi Pemesanan, Issue 4 – Observability, QA, dan Peluncuran, Rencana Sprint (3 Sprint, masing-masing 2 minggu) (+1 more)

### Community 90 - "requestid_test.go"
Cohesion: 0.11
Nodes (25): main(), main(), RequestIDFromContext(), TestContextRequestIDAddedToRecords(), TestRequestIDFromContextEmpty(), WithRequestID(), IsDevEnv(), New() (+17 more)

### Community 91 - "🔌 API Endpoints Specification"
Cohesion: 0.25
Nodes (8): Admin Dashboard (`/api/v1/admin`), 🔌 API Endpoints Specification, Authentication (`/api/v1/auth`), Availability (`/api/v1/providers/:provider_id`), Bookings (`/api/v1/bookings`), Payments (`/api/v1/payments`), Providers (`/api/v1/providers`), Services (`/api/v1/providers/:provider_id/services`)

### Community 92 - "ParseClock"
Cohesion: 0.32
Nodes (4): ParseClock(), TestParseClock(), TestParseClockEndOfDay(), clockSerializer

### Community 93 - "📝 Code Examples & Patterns"
Cohesion: 0.29
Nodes (7): 1. Standardized API Response Format, 2. Middleware Examples, 3. Repository Pattern Example, 4. Service Layer Example, 5. Frontend API Client with Interceptors, 6. React Hook for Bookings, 📝 Code Examples & Patterns

### Community 94 - "docker-compose.yml stack"
Cohesion: 0.67
Nodes (6): docker-compose.yml stack, backend service (Go API), frontend service (SPA on Nginx), nginx reverse proxy service, postgres service (16-alpine), redis service (7-alpine, requirepass)

### Community 95 - "✅ Development Checklist"
Cohesion: 0.33
Nodes (6): ✅ Development Checklist, Phase 1: Foundation (Week 1-2), Phase 2: Core Booking System (Week 3-4), Phase 3: Notifications & Payments (Week 5), Phase 4: Admin Dashboard (Week 6), Phase 5: Polish & Testing (Week 7-8)

### Community 96 - "Rekomendasi pengembangan berikutnya"
Cohesion: 0.33
Nodes (6): 1. Payment production path — selesai untuk jalur aplikasi, 2. Notification delivery — selesai untuk outbox dan worker, 3. QA dan keamanan — P1, 4. API contract dan operasional — P1, 5. Performance dan product completion — P2, Rekomendasi pengembangan berikutnya

### Community 97 - "accesslog_test.go"
Cohesion: 0.57
Nodes (6): accessLogEngine(), decodeLines(), serve(), TestAccessLogLevelsByStatus(), TestAccessLogOneLineWithStatusAndRequestID(), TestAccessLogSkipsHealthProbes()

### Community 98 - "🎓 Learning Resources"
Cohesion: 0.40
Nodes (5): Database & Caching, DevOps, Go (Backend), 🎓 Learning Resources, React (Frontend)

### Community 99 - "main.tsx"
Cohesion: 0.40
Nodes (3): queryClient, react-dom, react-toastify

### Community 100 - "FormatValidationErrors"
Cohesion: 0.50
Nodes (3): formatErrorMessage(), FormatValidationErrors(), ValidationError

### Community 101 - "🧮 Core Business Logic"
Cohesion: 0.50
Nodes (4): 1. Available Slots Calculation Algorithm, 2. Booking Creation with Conflict Prevention, 3. Email Notification System, 🧮 Core Business Logic

### Community 102 - "🎨 Frontend Key Features Implementation"
Cohesion: 0.50
Nodes (4): 1. Booking Calendar Component, 2. Time Slot Picker, 3. Real-time Slot Updates (Optional Enhancement), 🎨 Frontend Key Features Implementation

### Community 103 - "🚀 Deployment Instructions"
Cohesion: 0.50
Nodes (4): 1. Build for Production, 2. Deploy to VPS (Example: DigitalOcean), 3. Nginx Configuration, 🚀 Deployment Instructions

### Community 104 - "📚 Additional Resources & Best Practices"
Cohesion: 0.50
Nodes (4): 📚 Additional Resources & Best Practices, Database Migration Strategy, Error Codes Convention, Logging Best Practices

### Community 105 - "📋 Tech Stack Requirements"
Cohesion: 0.50
Nodes (4): Backend, Frontend, Infrastructure, 📋 Tech Stack Requirements

### Community 106 - "🌐 Internationalization (i18n)"
Cohesion: 0.50
Nodes (4): 🌐 Internationalization (i18n), Setup i18n, Translation Files, Usage in Components

### Community 107 - "eslint.config.js"
Cohesion: 0.33
Nodes (4): eslint-plugin-react-hooks, eslint-plugin-react-refresh, globals, typescript-eslint

### Community 109 - "🔧 Environment Variables"
Cohesion: 0.67
Nodes (3): Backend (.env), 🔧 Environment Variables, Frontend (.env)

### Community 110 - "🏗️ Project Structure"
Cohesion: 0.67
Nodes (3): Backend (Go), Frontend (React), 🏗️ Project Structure

### Community 111 - "🧪 Testing Guidelines"
Cohesion: 0.67
Nodes (3): Backend Unit Test Example, Frontend Component Test Example, 🧪 Testing Guidelines

### Community 112 - "📱 Mobile Responsiveness"
Cohesion: 0.67
Nodes (3): Breakpoints (Tailwind), 📱 Mobile Responsiveness, Responsive Design Patterns

### Community 113 - "💳 Payment Integration (Midtrans)"
Cohesion: 0.67
Nodes (3): Implementation Flow, 💳 Payment Integration (Midtrans), Setup Steps

## Knowledge Gaps
- **314 isolated node(s):** `temu.in/backend`, `promoteReq`, `verifyEmailReq`, `forgotPasswordReq`, `resetPasswordReq` (+309 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 451 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **23 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Booking` connect `Booking` to `router.go`, `gorm.io/gorm.DB`, `booking/service_test.go`, `time.Time`, `Service`, `Handlers`, `ServiceProvider`, `NewBookingFinalizer`, `context.Context`, `testing.T`, `newWebhookEnv`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **Why does `Payment` connect `Payment` to `router.go`, `time.Time`, `newPaymentIT`, `Service`, `NewBookingFinalizer`, `Booking`, `context.Context`, `testing.T`, `newWebhookEnv`, `Handlers`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Why does `RequireAuth()` connect `react` to `index.ts`, `App.tsx`, `Laporan Progres Temu.in`, `handlers.ts`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._
- **What connects `temu.in/backend`, `promoteReq`, `verifyEmailReq` to the rest of the system?**
  _314 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `router.go` be split into smaller, more focused modules?**
  _Cohesion score 0.1327181106186631 - nodes in this community are weakly interconnected._
- **Should `000001_initial_schema.up.sql` be split into smaller, more focused modules?**
  _Cohesion score 0.07207792207792207 - nodes in this community are weakly interconnected._
- **Should `graphify Skill (SKILL.md)` be split into smaller, more focused modules?**
  _Cohesion score 0.0596078431372549 - nodes in this community are weakly interconnected._