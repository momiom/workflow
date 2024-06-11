package node

import (
	"fmt"
)

// LLMNodeはLLM（大規模言語モデル）を利用するノードです。
type LLMNode struct {
	name      string
	inputs    []string
	outputs   []string
	llmClient LLMClient
}

// LLMClientはLLMサービスと通信するためのインターフェースです。
type LLMClient interface {
	GenerateResponse(prompt string) (string, error)
}

// NewLLMNodeは新しいLLMNodeを作成します。
func NewLLMNode(name string, client LLMClient) *LLMNode {
	return &LLMNode{name: name, llmClient: client}
}

// ExecuteはLLMにテキストを送り、応答を受け取ります。
func (n *LLMNode) Execute() error {
	if len(n.inputs) != 1 {
		return fmt.Errorf("input must be exactly 1, got %d", len(n.inputs))
	}
	if len(n.inputs[0]) == 0 {
		return fmt.Errorf("input must not be empty")
	}

	response, err := n.llmClient.GenerateResponse(n.inputs[0])
	if err != nil {
		return err
	}

	n.outputs = []string{response}
	return nil
}

// Nameはノードの名前を返します。
func (n *LLMNode) Name() string {
	return n.name
}

// SetInputsはノードの入力を設定します。
func (n *LLMNode) SetInputs(inputs []string) {
	n.inputs = inputs
}

// GetOutputsはノードの出力を返します。
func (n *LLMNode) GetOutputs() []string {
	return n.outputs
}
