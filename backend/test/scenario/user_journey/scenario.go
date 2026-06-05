package user_journey

import (
	"context"
	"encoding/json"
	"fmt"

	"test-service/runner"
)

type Scenario struct {
	biz      BusinessClient
	internal InternalClient
	rec      SeedRecorder
}

func NewScenario(biz BusinessClient, internal InternalClient, rec SeedRecorder) *Scenario {
	return &Scenario{biz: biz, internal: internal, rec: rec}
}

func (s *Scenario) Type() string { return "user_journey" }

func (s *Scenario) Run(ctx context.Context, raw json.RawMessage, p runner.ProgressEmitter) (any, error) {
	cfg := Config{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("invalid user_journey config: %w", err)
		}
	}
	if cfg.TestID == "" {
		cfg.TestID = runner.TestIDFromContext(ctx)
	}
	return New(s.biz, s.internal, s.rec, cfg).Run(ctx, p)
}
