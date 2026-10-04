package models

import "testing"

func TestTableNames(t *testing.T) {
	cases := map[string]interface{ TableName() string }{
		"availability_schedules":  AvailabilitySchedule{},
		"availability_exceptions": AvailabilityException{},
		"notifications":           Notification{},
		"notification_deliveries": NotificationDelivery{},
		"service_providers":       ServiceProvider{},
		"reviews":                 Review{},
		"services":                Service{},
	}
	for want, m := range cases {
		if got := m.TableName(); got != want {
			t.Errorf("%T.TableName() = %q, want %q", m, got, want)
		}
	}
}

func TestBeforeCreateAssignsUUIDOnlyWhenEmpty(t *testing.T) {
	check := func(name string, id *string, run func() error) {
		t.Helper()
		*id = ""
		if err := run(); err != nil || len(*id) != 36 {
			t.Errorf("%s: generated id %q, err %v", name, *id, err)
		}
		*id = "keep"
		if err := run(); err != nil || *id != "keep" {
			t.Errorf("%s: existing id overwritten: %q", name, *id)
		}
	}
	var (
		sched AvailabilitySchedule
		exc   AvailabilityException
		prov  ServiceProvider
		rev   Review
		svc   Service
	)
	check("schedule", &sched.ID, func() error { return sched.BeforeCreate(nil) })
	check("exception", &exc.ID, func() error { return exc.BeforeCreate(nil) })
	check("provider", &prov.ID, func() error { return prov.BeforeCreate(nil) })
	check("review", &rev.ID, func() error { return rev.BeforeCreate(nil) })
	check("service", &svc.ID, func() error { return svc.BeforeCreate(nil) })
}
