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
	namespace     string
	allNamespaces bool
	kubeconfig    string
	showAll       bool
	checkConfig   bool
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
  - ECI (Elastic Container Instance) pod identification
  - Summary statistics

Examples:
  # View pods in default namespace
  kubectl podview

  # View pods in a specific namespace
  kubectl podview -n test-gatekeeper

  # View pods across all namespaces
  kubectl podview -A

  # Show all pods including healthy ones
  kubectl podview -n test-gatekeeper --all

  # Check resource configuration issues
  kubectl podview -n test-gatekeeper --check-config`,

	RunE: runPodView,
}

func init() {
	// 添加命令行参数
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace to inspect")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Query all namespaces")
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
	// 创建带超时的 context，全命名空间查询需要更长时间
	timeout := 30 * time.Second
	if allNamespaces {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 1. 创建 Kubernetes 客户端
	fmt.Printf("🔗 Connecting to cluster...\n")
	k8sClient, err := client.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 2. 确定查询范围
	queryNamespace := namespace
	if allNamespaces {
		queryNamespace = "" // 空字符串表示所有命名空间
		fmt.Printf("📦 Fetching pods across all namespaces...\n")
	} else {
		fmt.Printf("📦 Fetching pods in namespace '%s'...\n", namespace)
	}

	// 3. 获取 Pod 列表
	pods, err := k8sClient.GetPods(ctx, queryNamespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		if allNamespaces {
			fmt.Printf("⚠️  No pods found in the cluster\n")
		} else {
			fmt.Printf("⚠️  No pods found in namespace '%s'\n", namespace)
		}
		return nil
	}

	// 4. 分析 Pod 状态
	fmt.Printf("🔍 Analyzing %d pods...\n\n", len(pods.Items))
	results := analyzer.AnalyzePods(pods, checkConfig)

	// 5. 打印结果
	p := printer.NewPrinter(os.Stdout)
	p.PrintPodTable(results, showAll, allNamespaces)
	p.PrintSummary(results)

	// 6. 如果有问题，打印建议
	if results.HasIssues() {
		p.PrintRecommendations(results)
	}

	return nil
}
