package booking

import (
	"errors"
	"fmt"
	"testing"
)

type sqlStateErr struct{ code string }

func (e sqlStateErr) Error() string    { return "sqlstate " + e.code }
func (e sqlStateErr) SQLState() string { return e.code }

func TestIsConflictError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"exclusion violation 23P01", sqlStateErr{"23P01"}, true},
		{"unique violation 23505", sqlStateErr{"23505"}, true},
		{"foreign key violation 23503", sqlStateErr{"23503"}, false},
		{"wrapped exclusion violation", fmt.Errorf("create booking: %w", sqlStateErr{"23P01"}), true},
		{"wrapped foreign key violation", fmt.Errorf("create booking: %w", sqlStateErr{"23503"}), false},
	}
	for _, tc := range cases {
		if got := IsConflictError(tc.err); got != tc.want {
			t.Errorf("%s: IsConflictError = %v, want %v", tc.name, got, tc.want)
		}
	}
}
