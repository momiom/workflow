package dag_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/momiom/workflow/dag"
	"github.com/momiom/workflow/node"
)

type MockLLMClient struct{}

func (c *MockLLMClient) GenerateResponse(prompt string) (string, error) {
	return "mock response: " + prompt, nil
}

func TestDAG(t *testing.T) {
	// テキストプロセッサ関数
	textProcessor := func(inputs []string) (string, error) {
		if len(inputs) == 0 {
			return "", fmt.Errorf("input must not be empty")
		}
		r := strings.Join(inputs, " ")
		return r, nil
	}

	// テストケース
	tests := []struct {
		name                 string
		maxConcurrent        int
		nodes                map[dag.NodeID]node.Node
		edges                [][]dag.NodeID
		inputs               map[dag.NodeID][]string
		expectedOutputs      map[dag.NodeID][]string
		expectedFinalOutputs map[dag.NodeID][]string
		expectError          bool
	}{
		{
			name:          "text -> llm",
			maxConcurrent: 2,
			nodes: map[dag.NodeID]node.Node{
				"textNode": node.NewTextNode("textNode", textProcessor),
				"llmNode":  node.NewLLMNode("llmNode", &MockLLMClient{}),
			},
			edges: [][]dag.NodeID{
				{"textNode", "llmNode"},
			},
			inputs: map[dag.NodeID][]string{
				"textNode": {"hello", "world"},
			},
			expectedOutputs: map[dag.NodeID][]string{
				"textNode": {"hello world"},
				"llmNode":  {"mock response: hello world"},
			},
			expectedFinalOutputs: map[dag.NodeID][]string{
				"llmNode": {"mock response: hello world"},
			},
			expectError: false,
		},
		{
			name:          "text -> llm, text2 -> llm2",
			maxConcurrent: 2,
			nodes: map[dag.NodeID]node.Node{
				"textNode1": node.NewTextNode("textNode1", textProcessor),
				"textNode2": node.NewTextNode("textNode2", textProcessor),
				"llmNode":   node.NewLLMNode("llmNode", &MockLLMClient{}),
				"llmNode2":  node.NewLLMNode("llmNode2", &MockLLMClient{}),
			},
			edges: [][]dag.NodeID{
				{"textNode1", "llmNode"},
				{"textNode2", "llmNode2"},
			},
			inputs: map[dag.NodeID][]string{
				"textNode1": {"hello", "world"},
				"textNode2": {"goodbye", "world"},
			},
			expectedOutputs: map[dag.NodeID][]string{
				"textNode1": {"hello world"},
				"textNode2": {"goodbye world"},
			},
			expectedFinalOutputs: map[dag.NodeID][]string{
				"llmNode":  {"mock response: hello world"},
				"llmNode2": {"mock response: goodbye world"},
			},
			expectError: false,
		},
		{
			name:          "text -> llm, text2 -> llm2, llm -> text3, llm2 -> text3",
			maxConcurrent: 1,
			nodes: map[dag.NodeID]node.Node{
				"textNode1": node.NewTextNode("textNode1", textProcessor),
				"textNode2": node.NewTextNode("textNode2", textProcessor),
				"llmNode":   node.NewLLMNode("llmNode", &MockLLMClient{}),
				"llmNode2":  node.NewLLMNode("llmNode2", &MockLLMClient{}),
				"textNode3": node.NewTextNode("textNode3", textProcessor),
			},
			edges: [][]dag.NodeID{
				{"textNode1", "llmNode"},
				{"textNode2", "llmNode2"},
				{"llmNode", "textNode3"},
				{"llmNode2", "textNode3"},
			},
			inputs: map[dag.NodeID][]string{
				"textNode1": {"hello", "world"},
				"textNode2": {"goodbye", "world"},
			},
			expectedOutputs: map[dag.NodeID][]string{
				"textNode1": {"hello world"},
				"textNode2": {"goodbye world"},
				"textNode3": {"mock response: hello world mock response: goodbye world"},
			},
			expectedFinalOutputs: map[dag.NodeID][]string{
				"textNode3": {"mock response: hello world mock response: goodbye world"},
			},
			expectError: false,
		},
		{
			name:          "DAG with missing inputs",
			maxConcurrent: 2,
			nodes: map[dag.NodeID]node.Node{
				"textNode": node.NewTextNode("textNode", textProcessor),
				"llmNode":  node.NewLLMNode("llmNode", &MockLLMClient{}),
			},
			edges: [][]dag.NodeID{
				{"textNode", "llmNode"},
			},
			inputs: map[dag.NodeID][]string{
				"textNode": {""},
			},
			expectedOutputs: map[dag.NodeID][]string{
				"textNode": {""},
			},
			expectedFinalOutputs: nil,
			expectError:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			workflow := dag.NewDAG(tt.maxConcurrent)
			for id, n := range tt.nodes {
				workflow.AddNode(id, n)
			}
			for _, edge := range tt.edges {
				err := workflow.AddEdge(edge[0], edge[1])
				if err != nil {
					t.Fatalf("failed to add edge: %v", err)
				}
			}

			nodeOutputs, finalOutputs, err := workflow.Execute(ctx, tt.inputs)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				for id, expectedOutput := range tt.expectedOutputs {
					output, exists := nodeOutputs[id]
					if !exists {
						t.Fatalf("missing output for node %s", id)
					}
					if len(output) != len(expectedOutput) {
						t.Fatalf("expected output length %d for node %s, got %d", len(expectedOutput), id, len(output))
					}
					for i := range output {
						if !slices.Contains(expectedOutput, output[i]) {
							t.Fatalf("expected %v, got %v", expectedOutput, output)
						}
					}
				}

				if len(finalOutputs) != len(tt.expectedFinalOutputs) {
					t.Fatalf("expected final outputs length %d, got %d", len(tt.expectedFinalOutputs), len(finalOutputs))
				}
				for k, v := range finalOutputs {
					o, ok := tt.expectedFinalOutputs[k]
					if !ok {
						t.Fatalf("expected final output for node %s, but not found", k)
					}
					if v[0] != o[0] {
						t.Fatalf("expected final output %v, got %v", o, v)
					}
				}
			}
		})
	}
}
