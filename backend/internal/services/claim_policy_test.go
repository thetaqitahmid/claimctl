package services

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReservationPolicyValidation(t *testing.T) {
	for _, tc := range []struct{ key, value string }{{"RESERVATION_DEFAULT_DURATION", "0s"}, {"RESERVATION_DEFAULT_DURATION", "25h"}, {"RESERVATION_MAX_DURATION", "invalid"}, {"RESERVATION_MAX_LIVE_CLAIMS", "0"}, {"RESERVATION_MAX_LIVE_CLAIMS", "9223372036854775808"}, {"RESERVATION_ALLOW_INDEFINITE", "sometimes"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			_, err := LoadReservationPolicy()
			require.Error(t, err)
		})
	}
}

func TestClaimDurationValidationBeforeTransaction(t *testing.T) {
	p, err := LoadReservationPolicy()
	require.NoError(t, err)
	s := &reservationService{policy: p}
	for _, v := range []string{"", "0s", "-1h", "25h", "999999999999999999999999d", "768614336404564651d", "0.1d", "bad"} {
		t.Run(v, func(t *testing.T) {
			_, err := s.Claim(context.Background(), uuid.New(), ClaimRequest{ResourceID: uuid.New(), Duration: &v}, "")
			require.Error(t, err)
		})
	}
	s.policy.AllowIndefinite = true
	v := "1h"
	_, err = s.Claim(context.Background(), uuid.New(), ClaimRequest{ResourceID: uuid.New(), Indefinite: true, Duration: &v}, "")
	require.Error(t, err)
}
