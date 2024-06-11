package node_test

import (
	"testing"

	"github.com/momiom/workflow/node"
)

type MockLLMClient struct{}

func (c *MockLLMClient) GenerateResponse(prompt string) (string, error) {
	return "mock response: " + prompt, nil
}

func TestLLMNode(t *testing.T) {
	client := &MockLLMClient{}

	tests := []struct {
		name           string
		inputs         []string
		expectedOutput string
		expectError    bool
	}{
		{"Generate response for hello world", []string{"hello world"}, "mock response: hello world", false},
		{"Empty input", []string{""}, "", true},
		{"Multiple inputs", []string{"first input", "second input"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := node.NewLLMNode("llmNode", client)
			n.SetInputs(tt.inputs)

			err := n.Execute()
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				outputs := n.GetOutputs()
				if len(outputs) != 1 || outputs[0] != tt.expectedOutput {
					t.Fatalf("expected %v, got %v", tt.expectedOutput, outputs)
				}
			}
		})
	}
}
