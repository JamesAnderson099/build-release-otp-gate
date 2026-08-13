package release

import (
	"context"
	"errors"
	"testing"
)

type stubCodes struct {
	verifyErr error
	verified  int
}

func (s *stubCodes) RequestOTP(context.Context, string, string) error { return nil }
func (s *stubCodes) VerifyOTP(context.Context, string, string, string) error {
	s.verified++
	return s.verifyErr
}

func TestVerifyAndReleaseDecision(t *testing.T) {
	tests := []struct {
		name        string
		buildID     string
		verifyErr   error
		wantResult  string
		wantState   string
		wantCalls   int
		wantErr     bool
		wantMissing bool
	}{
		{name: "verified operator releases build", buildID: "build-1842", wantResult: "released", wantState: "released", wantCalls: 1},
		{name: "rejected code keeps build pending", buildID: "build-1842", verifyErr: errors.New("code rejected"), wantResult: "rejected", wantState: "awaiting_verification", wantCalls: 1, wantErr: true},
		{name: "unknown build stops before SMS", buildID: "build-missing", wantState: "missing", wantErr: true, wantMissing: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sms := &stubCodes{verifyErr: tt.verifyErr}
			gate := NewGate(sms)
			diagnostic, err := gate.VerifyAndRelease(context.Background(), tt.buildID, "481205")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrBuildNotFound) != tt.wantMissing {
				t.Errorf("errors.Is(error, ErrBuildNotFound) = %v, want %v", errors.Is(err, ErrBuildNotFound), tt.wantMissing)
			}
			if diagnostic.Result != tt.wantResult {
				t.Errorf("result = %q, want %q", diagnostic.Result, tt.wantResult)
			}
			if sms.verified != tt.wantCalls {
				t.Errorf("verify calls = %d, want %d", sms.verified, tt.wantCalls)
			}
			build, ok := gate.Build(tt.buildID)
			state := "missing"
			if ok {
				state = build.State
			}
			if state != tt.wantState {
				t.Errorf("state = %q, want %q", state, tt.wantState)
			}
		})
	}
}
