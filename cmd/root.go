package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/FishPie-HQ/kubectl-podview/pkg/analyzer"
	"github.com/FishPie-HQ/kubectl-podview/pkg/client"
	"github.com/FishPie-HQ/kubectl-podview/pkg/printer"
)

var (
	namespace   string
	kubeconfig  string
	showAll     bool
	checkConfig bool
)

// rootCmd 是根命令
var rootCmd = &cobra.Command{
	Use:   "kubectl-podview",
	Short: "A kubectl plugin to view pod status and resource configuration",
	Long: `kubectl-podview is a kubectl plugin that provides a comprehensive view
of pods in a namespace, including:
  - Pod status and conditions
  - Container restart counts and reasons
  - Resource requests/limits configuration check
  - Summary statistics

Examples:
  # View pods in default namespace
  kubectl podview

  # View pods in a specific namespace
  kubectl podview -n test-gatekeeper

  # Show all pods including healthy ones
  kubectl podview -n test-gatekeeper --all

  # Check resource configuration issues
  kubectl podview -n test-gatekeeper --check-config`,

	RunE: runPodView,
}

func init() {
	// 添加命令行参数
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace to inspect")
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: ~/.kube/config)")
	rootCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all pods, including healthy ones")
	rootCmd.Flags().BoolVar(&checkConfig, "check-config", false, "Check and highlight resource configuration issues")
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

// runPodView 是主要的执行逻辑
func runPodView(cmd *cobra.Command, args []string) error {
	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 创建 Kubernetes 客户端
	fmt.Printf("🔗 Connecting to cluster...\n")
	k8sClient, err := client.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 2. 获取 Pod 列表
	fmt.Printf("📦 Fetching pods in namespace '%s'...\n", namespace)
	pods, err := k8sClient.GetPods(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		fmt.Printf("⚠️  No pods found in namespace '%s'\n", namespace)
		return nil
	}

	// 3. 分析 Pod 状态
	fmt.Printf("🔍 Analyzing %d pods...\n\n", len(pods.Items))
	results := analyzer.AnalyzePods(pods, checkConfig)

	// 4. 打印结果
	p := printer.NewPrinter(os.Stdout)
	p.PrintPodTable(results, showAll)
	p.PrintSummary(results)

	// 5. 如果有问题，打印建议
	if results.HasIssues() {
		p.PrintRecommendations(results)
	}

	return nil
}
