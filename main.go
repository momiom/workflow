package main

import (
	"context"
	"fmt"
	"github.com/momiom/workflow/dag"
	"github.com/momiom/workflow/node"
	"log/slog"
	"os"
	"runtime/trace"
	"time"
)

type MockLLMClient struct{}

func (c *MockLLMClient) GenerateResponse(prompt string) (string, error) {
	return "Mock LLM output 1: Input prompt is \"" + prompt + "\"", nil
}

type MockLLMClient2 struct{}

func (c *MockLLMClient2) GenerateResponse(prompt string) (string, error) {
	return "Mock LLM output 2: Input prompt is \"" + prompt + "\"", nil
}

func main() {
	// 現在時刻を取得
	start := time.Now()

	// トレースファイルの作成
	f, err := os.Create("trace.out")
	if err != nil {
		slog.Error("Failed to create trace output file", "error", err)
		return
	}
	defer f.Close()

	// トレースの開始
	if err := trace.Start(f); err != nil {
		slog.Error("Failed to start trace", "error", err)
		return
	}
	defer trace.Stop()

	ctx := context.Background()

	var logLevel = new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// ノードの作成
	textProcessor := func(inputs []string) (string, error) {
		time.Sleep(3 * time.Second)
		return inputs[0] + " " + inputs[1], nil
	}
	textProcessor2 := func(inputs []string) (string, error) {
		time.Sleep(3 * time.Second)
		return inputs[0] + " " + inputs[1], nil
	}
	textProcessor3 := func(inputs []string) (string, error) {
		time.Sleep(3 * time.Second)
		return inputs[0] + " " + inputs[1], nil
	}

	textNode := node.NewTextNode("textNode", textProcessor)
	llmClient := &MockLLMClient{}
	llmNode := node.NewLLMNode("llmNode", llmClient)
	textNode2 := node.NewTextNode("textNode2", textProcessor2)
	llmClient2 := &MockLLMClient2{}
	llmNode2 := node.NewLLMNode("llmNode2", llmClient2)
	textNode3 := node.NewTextNode("textNode3", textProcessor3)

	// DAGの作成
	workflow := dag.NewDAG(2) // 最大同時実行数を2に設定
	workflow.AddNode("textNode", textNode)
	workflow.AddNode("llmNode", llmNode)
	workflow.AddNode("textNode2", textNode2)
	workflow.AddNode("llmNode2", llmNode2)
	workflow.AddNode("textNode3", textNode3)

	// エッジの設定
	workflow.AddEdge("textNode", "llmNode")
	workflow.AddEdge("textNode2", "llmNode2")
	workflow.AddEdge("llmNode", "textNode3")
	workflow.AddEdge("llmNode2", "textNode3")

	// 入力の設定
	inputs := map[dag.NodeID][]string{
		"textNode":  {"hello", "world"},
		"textNode2": {"goodbye", "world"},
	}

	// 状態変更と入出力を監視するゴルーチンを起動
	go func() {
		for state := range workflow.GetStatusChan() {
			fmt.Printf("Node %s is now %s\n", state.ID, state.Status)
		}
	}()

	go func() {
		for io := range workflow.GetIOChan() {
			fmt.Printf("Node %s inputs: %v outputs: %v\n", io.ID, io.Inputs, io.Outputs)
		}
	}()

	// ワークフローの実行
	nodeOutputs, finalOutputs, err := workflow.Execute(ctx, inputs)
	if err != nil {
		slog.Error("Error executing workflow", "error", err)
		return
	}

	// 各ノードの出力を表示
	for id, output := range nodeOutputs {
		slog.Info("Node Outputs", "node", id, "output", output)
	}

	// ワークフロー全体の最終出力を表示
	for _, output := range finalOutputs {
		slog.Info("Final Outputs", "output", output)
	}

	// 経過時間を表示
	slog.Info("Elapsed time", "time", time.Since(start).Seconds())
}
