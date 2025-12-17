package http

import (
	"encoding/json"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/graph"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

// UIEvent is het envelope dat je naar de frontend streamt.
// Bevat alles wat je nodig hebt om per node/graph te monitoren.
type UIEvent struct {
	// Event / routing info
	Type               string `json:"type"`    // afgeleid van ev.Object
	Object             string `json:"object"`  // ruwe event object type (graph.node.execution, model.response, ...)
	EventID            string `json:"eventId"` // ev.ID
	Author             string `json:"author,omitempty"`
	Timestamp          string `json:"timestamp"` // RFC3339Nano string
	RequestID          string `json:"requestId,omitempty"`
	InvocationID       string `json:"invocationId,omitempty"`
	ParentInvocationID string `json:"parentInvocationId,omitempty"`
	FilterKey          string `json:"filterKey,omitempty"`
	RunnerCompletion   bool   `json:"runnerCompletion,omitempty"`
	GraphCompletion    bool   `json:"graphCompletion,omitempty"` // evt. toekomstige uitbreiding

	// Model / LLM streaming
	ContentDelta   string           `json:"contentDelta,omitempty"`   // streaming delta voor huidige node
	Content        string           `json:"content,omitempty"`        // final content (per node / per model call)
	ToolCallsDelta []model.ToolCall `json:"toolCallsDelta,omitempty"` // streaming tool_call delta
	ToolCalls      []model.ToolCall `json:"toolCalls,omitempty"`      // final tool_calls

	// Token usage per model call (alleen gevuld op non-partial events)
	Usage *model.Usage `json:"usage,omitempty"`

	// Metadata uit StateDelta
	ModelMetadata   *graph.ModelExecutionMetadata `json:"modelMetadata,omitempty"`
	NodeMetadata    *graph.NodeExecutionMetadata  `json:"nodeMetadata,omitempty"`
	PregelMetadata  *graph.PregelStepMetadata     `json:"pregelMetadata,omitempty"`
	ChannelMetadata *graph.ChannelUpdateMetadata  `json:"channelMetadata,omitempty"`
	StateMetadata   *graph.StateUpdateMetadata    `json:"stateMetadata,omitempty"`

	// Ruwe error info (LLM/tool/flow error)
	Error *model.ResponseError `json:"error,omitempty"`

	// Optioneel: je kunt hier nog raw event toevoegen voor debug view
	// Raw *event.Event `json:"raw,omitempty"`
}

// BuildUIEvent projecteert een *event.Event naar een UIEvent.
func BuildUIEvent(ev *event.Event) *UIEvent {
	if ev == nil {
		return nil
	}

	ui := &UIEvent{
		// Belangrijk: event.Event heeft geen Type-veld; gebruik Object als type
		Type:    ev.Object,
		Object:  ev.Object,
		EventID: ev.ID,
		Author:  ev.Author,
		// Timestamp is time.Time → naar string
		Timestamp:          ev.Timestamp.Format(time.RFC3339Nano),
		RequestID:          ev.RequestID,
		InvocationID:       ev.InvocationID,
		ParentInvocationID: ev.ParentInvocationID,
		FilterKey:          ev.FilterKey,
	}

	// Error direct doorgeven (ResponseError komt uit embedded *model.Response)
	if ev.Error != nil {
		ui.Error = ev.Error
	}

	// Runner completion flag – handig om in UI "workflow klaar" te tonen.
	if ev.IsRunnerCompletion() {
		ui.RunnerCompletion = true
	}

	// Model/LLM streaming + usage
	if ev.Response != nil {
		resp := ev.Response

		if len(resp.Choices) > 0 {
			choice := resp.Choices[0]

			// Streaming deltas (IsPartial = true)
			if resp.IsPartial {
				if choice.Delta.Content != "" {
					ui.ContentDelta = choice.Delta.Content
				}
				if len(choice.Delta.ToolCalls) > 0 {
					ui.ToolCallsDelta = choice.Delta.ToolCalls
				}
			} else {
				// Final non-partial message per model call
				if choice.Message.Content != "" {
					ui.Content = choice.Message.Content
				}
				if len(choice.Message.ToolCalls) > 0 {
					ui.ToolCalls = choice.Message.ToolCalls
				}
			}
		}

		// Token usage alleen op non-partial events (resp.IsPartial == false)
		if !resp.IsPartial && resp.Usage != nil {
			ui.Usage = resp.Usage
		}
	}

	// Metadata uit StateDelta (Event Metadata / StateDelta section in Graph docs)
	if len(ev.StateDelta) > 0 {
		// ModelExecutionMetadata (_model_metadata)
		if b, ok := ev.StateDelta[graph.MetadataKeyModel]; ok {
			var md graph.ModelExecutionMetadata
			if err := json.Unmarshal(b, &md); err == nil {
				ui.ModelMetadata = &md
			}
		}

		// NodeExecutionMetadata (_node_metadata)
		if b, ok := ev.StateDelta[graph.MetadataKeyNode]; ok {
			var md graph.NodeExecutionMetadata
			if err := json.Unmarshal(b, &md); err == nil {
				ui.NodeMetadata = &md
			}
		}

		// PregelStepMetadata (_pregel_metadata) – bevat stepNumber, phase, activeNodes etc.
		if b, ok := ev.StateDelta[graph.MetadataKeyPregel]; ok {
			var md graph.PregelStepMetadata
			if err := json.Unmarshal(b, &md); err == nil {
				ui.PregelMetadata = &md
			}
		}

		// ChannelUpdateMetadata (_channel_metadata) – welke channels zijn geüpdatet, triggered nodes, etc.
		if b, ok := ev.StateDelta[graph.MetadataKeyChannel]; ok {
			var md graph.ChannelUpdateMetadata
			if err := json.Unmarshal(b, &md); err == nil {
				ui.ChannelMetadata = &md
			}
		}

		// StateUpdateMetadata (_state_metadata) – updatedKeys, removedKeys, stateSize.
		if b, ok := ev.StateDelta[graph.MetadataKeyState]; ok {
			var md graph.StateUpdateMetadata
			if err := json.Unmarshal(b, &md); err == nil {
				ui.StateMetadata = &md
			}
		}
	}

	return ui
}
