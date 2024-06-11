package node

// TextNodeはテキストを処理するノードです。
type TextNode struct {
	name      string
	inputs    []string
	outputs   []string
	processor func([]string) (string, error)
}

// NewTextNodeは新しいTextNodeを作成します。
func NewTextNode(name string, processor func([]string) (string, error)) *TextNode {
	return &TextNode{name: name, processor: processor}
}

// Executeは入力を処理する関数を使用してテキストを処理します。
func (n *TextNode) Execute() error {
	output, err := n.processor(n.inputs)
	if err != nil {
		return err
	}
	n.outputs = []string{output}
	return nil
}

// Nameはノードの名前を返します。
func (n *TextNode) Name() string {
	return n.name
}

// SetInputsはノードの入力を設定します。
func (n *TextNode) SetInputs(inputs []string) {
	n.inputs = inputs
}

// GetOutputsはノードの出力を返します。
func (n *TextNode) GetOutputs() []string {
	return n.outputs
}
