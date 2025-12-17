package runner

import (
	"context"
	"errors"
	"fmt"

	"helixrun/internal/agents"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	trpcrunner "trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
)

const defaultRunnerName = "helixrun-starter"

// ErrBuildAgent indicates that the agent could not be constructed from the registry.
var ErrBuildAgent = errors.New("runner: build agent failed")

// Service bundles all runner-related wiring so HTTP handlers only need to
// provide the request information.
type Service struct {
	registry       *agents.Registry
	sessionService session.Service
	runnerName     string
}

// NewService creates a Runner service with the default in-memory session store.
func NewService(reg *agents.Registry) *Service {
	return &Service{
		registry:       reg,
		sessionService: inmemory.NewSessionService(),
		runnerName:     defaultRunnerName,
	}
}

// WithSessionService overrides the default session backend.
func (s *Service) WithSessionService(svc session.Service) {
	if svc == nil {
		return
	}
	s.sessionService = svc
}

// WithRunnerName allows overriding the runner name used for telemetry/state.
func (s *Service) WithRunnerName(name string) {
	if name == "" {
		return
	}
	s.runnerName = name
}

// Run executes the requested agent with the provided message and streams events.
func (s *Service) Run(ctx context.Context, agentID, userID, sessionID string, message model.Message) (<-chan *event.Event, error) {
	if s == nil {
		return nil, fmt.Errorf("runner service is not initialized")
	}
	if s.registry == nil {
		return nil, fmt.Errorf("runner service registry is not configured")
	}
	agt, err := s.registry.BuildAgent(ctx, agentID)
	if err != nil {
		return nil, errors.Join(ErrBuildAgent, fmt.Errorf("build agent %q: %w", agentID, err))
	}

	appRunner := trpcrunner.NewRunner(
		s.runnerName,
		agt,
		trpcrunner.WithSessionService(s.sessionService),
	)

	return appRunner.Run(ctx, userID, sessionID, message)
}
