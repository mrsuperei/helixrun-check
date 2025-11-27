package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appmodel "helixrun/internal/model"

	"trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/agent/chainagent"
	"trpc.group/trpc-go/trpc-agent-go/agent/graphagent"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/graph"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/tool"
)

// Registry holds JSON-based agent configs.
type Registry struct {
	configs map[string]AgentConfig
}

// LoadRegistry loads all *.json configs from a directory.
func LoadRegistry(dir string) (*Registry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	reg := &Registry{
		configs: make(map[string]AgentConfig),
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var cfg AgentConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", path, err)
		}

		if cfg.ID == "" {
			cfg.ID = strings.TrimSuffix(e.Name(), ".json")
		}

		reg.configs[cfg.ID] = cfg
	}

	if len(reg.configs) == 0 {
		return nil, fmt.Errorf("no agent configs found in %s", dir)
	}

	return reg, nil
}

// ListAgentIDs returns all known agent IDs.
func (r *Registry) ListAgentIDs() []string {
	out := make([]string, 0, len(r.configs))
	for id := range r.configs {
		out = append(out, id)
	}
	return out
}

// BuildAgent builds a fresh agent.Agent instance from config.
func (r *Registry) BuildAgent(ctx context.Context, id string) (agent.Agent, error) {
	cfg, ok := r.configs[id]
	if !ok {
		return nil, fmt.Errorf("unknown agent ID: %s", id)
	}

	llm, genCfg, err := appmodel.NewModelFromConfig(cfg.Model, cfg.Stream)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	tools, err := buildTools(cfg)
	if err != nil {
		return nil, fmt.Errorf("build tools: %w", err)
	}

	switch cfg.Type {
	case AgentTypeSingle:
		return buildSingleAgent(cfg, llm, genCfg, tools)
	case AgentTypeMultiChain:
		return buildMultiChainAgent(cfg, llm, genCfg, tools)
	case AgentTypeGraph:
		return buildGraphAgent(cfg, llm, tools)
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", cfg.Type)
	}
}

func buildSingleAgent(cfg AgentConfig, llmModel model.Model, genCfg model.GenerationConfig, tools []tool.Tool) (agent.Agent, error) {
	agt := llmagent.New(
		cfg.ID,
		llmagent.WithModel(llmModel),
		llmagent.WithDescription(cfg.Description),
		llmagent.WithInstruction(cfg.Instruction),
		llmagent.WithGenerationConfig(genCfg),
		llmagent.WithTools(tools),
	)
	return agt, nil
}

func buildMultiChainAgent(
	cfg AgentConfig,
	llmModel model.Model,
	genCfg model.GenerationConfig,
	tools []tool.Tool,
) (agent.Agent, error) {
	if cfg.Multi == nil || strings.ToLower(cfg.Multi.Mode) != "chain" {
		return nil, fmt.Errorf("multi-agent config must have mode=chain")
	}
	if len(cfg.Multi.Agents) == 0 {
		return nil, fmt.Errorf("multi-agent chain must define at least one sub-agent")
	}

	// hier de subagents uit JSON bouwen
	subs := make([]agent.Agent, 0, len(cfg.Multi.Agents))
	for _, subCfg := range cfg.Multi.Agents {
		subAgent := llmagent.New(
			subCfg.ID,
			llmagent.WithModel(llmModel),
			llmagent.WithDescription(subCfg.Description),
			llmagent.WithInstruction(subCfg.Instruction),
			llmagent.WithGenerationConfig(genCfg),
			llmagent.WithTools(tools),
		)
		subs = append(subs, subAgent)
	}

	// BELANGRIJK: geen ... gebruiken, WithSubAgents verwacht []agent.Agent
	chain := chainagent.New(
		cfg.ID,
		chainagent.WithSubAgents(subs),
	)
	return chain, nil
}

func buildGraphAgent(cfg AgentConfig, llmModel model.Model, _ []tool.Tool) (agent.Agent, error) {
	if cfg.Graph == nil {
		return nil, fmt.Errorf("graph config is required for type=graph")
	}

	schema := graph.MessagesStateSchema()
	sg := graph.NewStateGraph(schema)

	for _, node := range cfg.Graph.Nodes {
		switch node.Type {
		case "entry":
			// Simple passthrough node.
			sg.AddNode(node.ID, func(ctx context.Context, s graph.State) (any, error) {
				return graph.State{}, nil
			})
		case "llm":

			sg.AddLLMNode(node.ID, llmModel, node.Instruction, nil)
		default:
			return nil, fmt.Errorf("unsupported graph node type: %s", node.Type)
		}
	}

	for _, edge := range cfg.Graph.Edges {
		sg.AddEdge(edge.From, edge.To)
	}

	sg.SetEntryPoint(cfg.Graph.Entry).SetFinishPoint(cfg.Graph.Finish)

	compiled, err := sg.Compile()
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	ga, err := graphagent.New(
		cfg.ID,
		compiled,
		graphagent.WithDescription(cfg.Description),
		graphagent.WithInitialState(graph.State{}),
	)
	if err != nil {
		return nil, fmt.Errorf("new graph agent: %w", err)
	}

	return ga, nil
}
