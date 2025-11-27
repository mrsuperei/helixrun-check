package agents

import "helixrun/internal/model"

// AgentType values.
const (
	AgentTypeSingle    = "single"      // single LLM agent
	AgentTypeMultiChain = "multi_chain" // chain multi-agent
	AgentTypeGraph     = "graph"       // graph-based agent
)

// AgentConfig is the JSON schema used to describe agents.
type AgentConfig struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Instruction string         `json:"instruction,omitempty"`
	Stream      bool           `json:"stream"`
	Model       model.Config   `json:"model"`
	Tools       []ToolConfig   `json:"tools,omitempty"`
	Multi       *MultiConfig   `json:"multi,omitempty"`
	Graph       *GraphConfig   `json:"graph,omitempty"`
}

// ToolConfig configures tools by name/type. For this starter, we only
// implement a simple "calculator" function tool.
type ToolConfig struct {
	Name string `json:"name"`
	Type string `json:"type"` // e.g. "calculator"
}

// MultiConfig configures multi-agent flows.
type MultiConfig struct {
	Mode   string            `json:"mode"`   // e.g. "chain"
	Agents []SubAgentConfig  `json:"agents"` // sub-agents for the chain
}

// SubAgentConfig describes a single step agent in a multi-agent flow.
type SubAgentConfig struct {
	ID          string `json:"id"`
	Instruction string `json:"instruction"`
	Description string `json:"description,omitempty"`
}

// GraphConfig describes a simple state graph for GraphAgent.
type GraphConfig struct {
	Nodes []GraphNodeConfig `json:"nodes"`
	Edges []GraphEdgeConfig `json:"edges"`
	Entry string            `json:"entry"`
	Finish string           `json:"finish"`
}

// GraphNodeConfig describes a node in the graph.
type GraphNodeConfig struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "entry" or "llm"
	Instruction string `json:"instruction,omitempty"`
}

// GraphEdgeConfig describes a directed edge between nodes.
type GraphEdgeConfig struct {
	From string `json:"from"`
	To   string `json:"to"`
}
