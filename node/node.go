// ノードのインターフェースを定義します。
package node

// Nodeインターフェースは全てのノードが実装すべきメソッドを定義します。
type Node interface {
	// Executeはノードのメインの処理を実行します。
	Execute() error

	// Nameはノードの名前を返します。
	Name() string

	// SetInputsはノードの入力を設定します。
	SetInputs(inputs []string)

	// GetOutputsはノードの出力を返します。
	GetOutputs() []string
}
