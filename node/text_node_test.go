package node_test

import (
	"fmt"
	"testing"

	"github.com/momiom/workflow/node"
)

func TestTextNode(t *testing.T) {
	processor := func(inputs []string) (string, error) {
		if len(inputs) != 2 {
			return "", fmt.Errorf("expected 2 inputs, got %d", len(inputs))
		}
		return inputs[0] + " " + inputs[1], nil
	}

	tests := []struct {
		name           string
		inputs         []string
		expectedOutput string
		expectError    bool
	}{
		{"Concatenate hello and world", []string{"hello", "world"}, "hello world", false},
		{"Concatenate foo and bar", []string{"foo", "bar"}, "foo bar", false},
		{"Single input", []string{"single"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := node.NewTextNode("textNode", processor)
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
