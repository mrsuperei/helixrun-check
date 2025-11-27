package agents

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// calculatorTool returns a very simple arithmetic tool.
func calculatorTool() tool.Tool {
	fn := func(ctx context.Context, req struct {
		Operation string  `json:"operation"`
		A         float64 `json:"a"`
		B         float64 `json:"b"`
	}) (map[string]any, error) {
		switch req.Operation {
		case "add":
			return map[string]any{"result": req.A + req.B}, nil
		case "multiply":
			return map[string]any{"result": req.A * req.B}, nil
		default:
			return nil, fmt.Errorf("unsupported operation: %s", req.Operation)
		}
	}

	return function.NewFunctionTool(
		fn,
		function.WithName("calculator"),
		function.WithDescription("Simple calculator tool (add/multiply)."),
	)
}

// buildTools instantiates tools defined in the agent config.
func buildTools(cfg AgentConfig) ([]tool.Tool, error) {
	var tools []tool.Tool
	for _, tc := range cfg.Tools {
		switch tc.Type {
		case "calculator":
			tools = append(tools, calculatorTool())
		default:
			return nil, fmt.Errorf("unsupported tool type: %s", tc.Type)
		}
	}
	return tools, nil
}
