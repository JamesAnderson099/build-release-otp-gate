package release

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrBuildNotFound = errors.New("build not found")

type CodeGateway interface {
	RequestOTP(context.Context, string, string) error
	VerifyOTP(context.Context, string, string, string) error
}

type Build struct {
	ID        string    `json:"build_id"`
	Commit    string    `json:"commit"`
	Actor     string    `json:"actor"`
	Phone     string    `json:"-"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type Diagnostic struct {
	BuildID string `json:"build_id"`
	Stage   string `json:"stage"`
	Result  string `json:"result"`
	Detail  string `json:"detail"`
}

type Gate struct {
	mu     sync.Mutex
	sms    CodeGateway
	builds map[string]*Build
}

func NewGate(sms CodeGateway) *Gate {
	return &Gate{sms: sms, builds: map[string]*Build{
		"build-1842": {
			ID: "build-1842", Commit: "8fd3c91", Actor: "pipeline-bot",
			Phone: "+14155550123", State: "awaiting_verification",
			CreatedAt: time.Date(2026, 8, 13, 9, 30, 0, 0, time.UTC),
		},
	}}
}

func (g *Gate) RequestCode(ctx context.Context, buildID string) (Diagnostic, error) {
	g.mu.Lock()
	build, ok := g.builds[buildID]
	g.mu.Unlock()
	if !ok {
		return Diagnostic{}, ErrBuildNotFound
	}
	if err := g.sms.RequestOTP(ctx, build.Phone, "release:"+build.ID+":request"); err != nil {
		return Diagnostic{}, fmt.Errorf("request release code: %w", err)
	}
	return Diagnostic{BuildID: build.ID, Stage: "verification", Result: "code_sent", Detail: "release remains pending"}, nil
}

func (g *Gate) VerifyAndRelease(ctx context.Context, buildID, code string) (Diagnostic, error) {
	g.mu.Lock()
	build, ok := g.builds[buildID]
	g.mu.Unlock()
	if !ok {
		return Diagnostic{}, ErrBuildNotFound
	}
	if err := g.sms.VerifyOTP(ctx, build.Phone, code, "release:"+build.ID+":verify"); err != nil {
		return Diagnostic{BuildID: build.ID, Stage: "verification", Result: "rejected", Detail: "release remains pending"}, err
	}

	g.mu.Lock()
	build.State = "released"
	g.mu.Unlock()
	return Diagnostic{BuildID: build.ID, Stage: "release", Result: "released", Detail: "verified operator approved commit " + build.Commit}, nil
}

func (g *Gate) Build(buildID string) (Build, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	build, ok := g.builds[buildID]
	if !ok {
		return Build{}, false
	}
	return *build, true
}
