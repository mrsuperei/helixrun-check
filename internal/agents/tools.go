package agents

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// 1. Definieer de argumenten als een publieke struct.
// De 'description' tags zijn CRUCIAAL voor de LLM om te snappen wat hij moet invullen.
type CalculatorArgs struct {
	Operation string  `json:"operation" description:"The operation to perform. Allowed values: 'add', 'multiply'."`
	A         float64 `json:"a" description:"The first number."`
	B         float64 `json:"b" description:"The second number."`
}

// calculatorTool returns a robust arithmetic tool.
func calculatorTool() tool.Tool {
	// 2. Gebruik de getypeerde struct in de functie-signatuur
	fn := func(ctx context.Context, args CalculatorArgs) (map[string]any, error) {
		switch args.Operation {
		case "add":
			// Tip: Geef context terug (bv. "result"), niet alleen een kaal getal.
			return map[string]any{
				"result": args.A + args.B,
				"status": "success",
			}, nil
		case "multiply":
			return map[string]any{
				"result": args.A * args.B,
				"status": "success",
			}, nil
		default:
			return nil, fmt.Errorf("unsupported operation: %s. Only 'add' and 'multiply' are supported.", args.Operation)
		}
	}

	// 3. Registreer de tool met een duidelijke beschrijving
	return function.NewFunctionTool(
		fn,
		function.WithName("calculator"),
		function.WithDescription("Perform basic arithmetic operations. Use this tool for each step of a calculation."),
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
