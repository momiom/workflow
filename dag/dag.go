// DAG（有向非巡回グラフ）を表すパッケージ
package dag

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/trace"
	"sync"

	"github.com/momiom/workflow/node"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type NodeID string

type NodeStatus string

const (
	Pending   NodeStatus = "Pending"
	Running   NodeStatus = "Running"
	Completed NodeStatus = "Completed"
	Error     NodeStatus = "Error"
)

type NodeState struct {
	ID     NodeID
	Status NodeStatus
}

type NodeIO struct {
	ID      NodeID
	Inputs  []string
	Outputs []string
}

type DAG struct {
	graph         *simple.DirectedGraph
	nodes         map[NodeID]graph.Node
	nodeMap       map[NodeID]node.Node
	inDegree      map[NodeID]int
	nodeStatus    map[NodeID]NodeStatus
	statusMu      sync.Mutex
	statusChan    chan NodeState
	ioChan        chan NodeIO
	maxConcurrent int
}

func NewDAG(maxConcurrent int) *DAG {
	return &DAG{
		graph:         simple.NewDirectedGraph(),
		nodes:         make(map[NodeID]graph.Node),
		nodeMap:       make(map[NodeID]node.Node),
		inDegree:      make(map[NodeID]int),
		nodeStatus:    make(map[NodeID]NodeStatus),
		statusChan:    make(chan NodeState),
		ioChan:        make(chan NodeIO),
		maxConcurrent: maxConcurrent,
	}
}

// ノードをDAGに追加するメソッド
func (dag *DAG) AddNode(id NodeID, n node.Node) {
	slog.Debug("Adding node", "id", id, "node", n)
	node := dag.graph.NewNode()
	dag.graph.AddNode(node)
	dag.nodes[id] = node
	dag.nodeMap[id] = n
	dag.inDegree[id] = 0
	dag.nodeStatus[id] = Pending
}

func (dag *DAG) GetStatusChan() <-chan NodeState {
	return dag.statusChan
}

func (dag *DAG) GetIOChan() <-chan NodeIO {
	return dag.ioChan
}

// エッジ（依存関係）をDAGに追加するメソッド
func (dag *DAG) AddEdge(from NodeID, to NodeID) error {
	slog.Debug("Adding edge", "from", from, "to", to)

	fromNode, ok := dag.nodes[from]
	if !ok {
		return fmt.Errorf("node %s does not exist", from)
	}
	toNode, ok := dag.nodes[to]
	if !ok {
		return fmt.Errorf("node %s does not exist", to)
	}

	dag.graph.SetEdge(dag.graph.NewEdge(fromNode, toNode))
	dag.inDegree[to]++
	return nil
}

// 出次数が0のノード（リーフノード）を取得するメソッド
func (dag *DAG) GetLeafNodes() []NodeID {
	slog.Debug("Getting leaf nodes")

	var leafNodes []NodeID
	nodes := dag.graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		if len(graph.NodesOf(dag.graph.From(node.ID()))) == 0 {
			for id, n := range dag.nodes {
				if n.ID() == node.ID() {
					leafNodes = append(leafNodes, id)
					break
				}
			}
		}
	}
	return leafNodes
}

func (dag *DAG) updateNodeStatus(id NodeID, status NodeStatus) {
	dag.statusMu.Lock()
	defer dag.statusMu.Unlock()
	dag.nodeStatus[id] = status
	dag.statusChan <- NodeState{ID: id, Status: status}
}

func (dag *DAG) notifyNodeIO(id NodeID, inputs, outputs []string) {
	dag.ioChan <- NodeIO{ID: id, Inputs: inputs, Outputs: outputs}
}

// DAGを実行するメソッド
func (dag *DAG) Execute(ctx context.Context, inputs map[NodeID][]string) (map[NodeID][]string, map[NodeID][]string, error) {
	slog.Debug("Executing DAG")

	// トポロジカルソートでノードの実行順序を決定
	sorted, err := topo.Sort(dag.graph)
	if err != nil {
		return nil, nil, err
	}

	outputs := make(map[NodeID][]string)      // ノードの出力を保持するマップ
	finalOutputs := make(map[NodeID][]string) // 最終出力を保持するマップ
	var mu sync.Mutex                         // 同期用のミューテックス
	var wg sync.WaitGroup                     // 並列処理の待機グループ
	var execErr error                         // 実行エラーを保持する変数

	sem := make(chan struct{}, dag.maxConcurrent) // セマフォとしてチャネルを使用

	var execNode func(ctx context.Context, id NodeID)

	// ノードを実行する関数
	execNode = func(ctx context.Context, id NodeID) {
		slog.Debug("Start execNode", "id", id)
		defer slog.Debug("End execNode", "id", id)

		defer wg.Done()
		sem <- struct{}{} // セマフォのロックを取得
		defer func() {
			<-sem // セマフォのロックを解放
		}()

		// ノードの状態を更新
		dag.updateNodeStatus(id, Running)

		// ノードごとにトレースイベントを開始
		trace.WithRegion(ctx, fmt.Sprintf("Node %s", id), func() {
			n := dag.nodeMap[id]

			// 初期入力と依存ノードからの入力を収集
			var nodeInputs []string
			if input, exists := inputs[id]; exists {
				nodeInputs = append(nodeInputs, input...)
			}

			for _, fromNode := range graph.NodesOf(dag.graph.To(dag.nodes[id].ID())) {
				for fromID, n := range dag.nodes {
					if n.ID() == fromNode.ID() {
						if output, exists := outputs[fromID]; exists {
							nodeInputs = append(nodeInputs, output...)
						}
						break
					}
				}
			}
			n.SetInputs(nodeInputs)
			slog.Debug("Node inputs", "id", id, "inputs", nodeInputs)

			// ノードを実行
			slog.Debug("Executing node", "id", id)
			if err := n.Execute(); err != nil {
				slog.Debug("Error executing node", "id", id, "error", err)
				mu.Lock()
				execErr = err
				mu.Unlock()
				dag.updateNodeStatus(id, Error)
				trace.Log(ctx, "error", err.Error())
				return
			}

			// ノードの出力を収集
			nodeOutputs := n.GetOutputs()
			mu.Lock()
			outputs[id] = nodeOutputs
			mu.Unlock()
			slog.Debug("Node outputs", "id", id, "outputs", nodeOutputs)

			// ノードの状態と入出力を更新
			dag.updateNodeStatus(id, Completed)
			dag.notifyNodeIO(id, nodeInputs, nodeOutputs)

			// 依存先ノードの入力次数を更新し、実行可能になったノードを実行
			mu.Lock()
			for _, toNode := range graph.NodesOf(dag.graph.From(dag.nodes[id].ID())) {
				for toID, n := range dag.nodes {
					if n.ID() == toNode.ID() {
						dag.inDegree[toID]--
						if dag.inDegree[toID] == 0 {
							wg.Add(1)
							go execNode(ctx, toID)
						}
						break
					}
				}
			}
			mu.Unlock()
		})
	}

	// トレースタスクを作成して実行を開始
	ctx, task := trace.NewTask(ctx, "DAG Execution")
	defer task.End()

	// 入力次数が0のノード（実行可能なノード）から実行を開始
	for _, n := range sorted {
		for id, node := range dag.nodes {
			if node.ID() == n.ID() && dag.inDegree[id] == 0 {
				wg.Add(1)
				go execNode(ctx, id)
			}
		}
	}

	wg.Wait()

	close(dag.statusChan)
	close(dag.ioChan)

	if execErr != nil {
		return nil, nil, execErr
	}

	// リーフノードの出力を収集
	for _, id := range dag.GetLeafNodes() {
		finalOutputs[id] = outputs[id]
	}

	return outputs, finalOutputs, nil
}
