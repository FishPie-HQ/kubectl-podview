package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/FishPie-HQ/kubectl-podview/pkg/analyzer"
)

// 终端颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Printer 负责格式化输出
type Printer struct {
	out io.Writer
}

// NewPrinter 创建一个新的 Printer
func NewPrinter(out io.Writer) *Printer {
	return &Printer{out: out}
}

// PrintPodTable 打印 Pod 表格
func (p *Printer) PrintPodTable(result *analyzer.AnalysisResult, showAll bool) {
	// 表头
	header := fmt.Sprintf("%-40s %-10s %-8s %-10s %-10s %-s",
		"NAME", "STATUS", "READY", "RESTARTS", "AGE", "REASON")
	fmt.Fprintln(p.out, colorBold+header+colorReset)
	fmt.Fprintln(p.out, strings.Repeat("-", 100))

	displayedCount := 0
	for _, pod := range result.Pods {
		// 如果不是 showAll 模式，跳过健康的 Pod
		if !showAll && pod.Status == analyzer.StatusHealthy && len(pod.ConfigIssues) == 0 {
			continue
		}

		displayedCount++
		p.printPodRow(pod)
	}

	if displayedCount == 0 {
		fmt.Fprintln(p.out, colorGreen+"  ✓ All pods are healthy!"+colorReset)
	}
	fmt.Fprintln(p.out)
}

// printPodRow 打印单行 Pod 信息
func (p *Printer) printPodRow(pod analyzer.PodAnalysis) {
	// 状态颜色
	statusColor := p.getStatusColor(pod.Status)
	
	// 状态图标
	statusIcon := p.getStatusIcon(pod.Status)
	
	// 格式化 reason，如果太长就截断
	reason := pod.Reason
	if len(reason) > 30 {
		reason = reason[:27] + "..."
	}

	// 配置问题标记
	configMark := ""
	if len(pod.ConfigIssues) > 0 {
		configMark = colorYellow + " ⚙" + colorReset
	}

	// 打印主行
	fmt.Fprintf(p.out, "%-40s %s%-10s%s %-8s %-10d %-10s %s%s\n",
		truncate(pod.Name, 40),
		statusColor,
		statusIcon+string(pod.Status),
		colorReset,
		pod.Ready,
		pod.Restarts,
		pod.Age,
		reason,
		configMark,
	)

	// 如果有配置问题，打印详情
	if len(pod.ConfigIssues) > 0 {
		for _, issue := range pod.ConfigIssues {
			fmt.Fprintf(p.out, "  %s└─ %s%s\n", colorYellow, issue, colorReset)
		}
	}
}

// PrintSummary 打印汇总统计
func (p *Printer) PrintSummary(result *analyzer.AnalysisResult) {
	fmt.Fprintln(p.out, colorBold+"📊 Summary"+colorReset)
	fmt.Fprintln(p.out, strings.Repeat("-", 40))
	
	fmt.Fprintf(p.out, "Total Pods:     %d\n", result.TotalPods)
	
	// 健康的用绿色
	if result.HealthyPods > 0 {
		fmt.Fprintf(p.out, "%sHealthy:        %d%s\n", colorGreen, result.HealthyPods, colorReset)
	}
	
	// Pending 用蓝色
	if result.PendingPods > 0 {
		fmt.Fprintf(p.out, "%sPending:        %d%s\n", colorBlue, result.PendingPods, colorReset)
	}
	
	// Warning 用黄色
	if result.WarningPods > 0 {
		fmt.Fprintf(p.out, "%sWarning:        %d%s\n", colorYellow, result.WarningPods, colorReset)
	}
	
	// Error 用红色
	if result.ErrorPods > 0 {
		fmt.Fprintf(p.out, "%sError:          %d%s\n", colorRed, result.ErrorPods, colorReset)
	}
	
	fmt.Fprintf(p.out, "Total Restarts: %d\n", result.TotalRestarts)
	
	if result.ConfigIssueCount > 0 {
		fmt.Fprintf(p.out, "%sConfig Issues:  %d%s\n", colorYellow, result.ConfigIssueCount, colorReset)
	}
	
	fmt.Fprintln(p.out)
}

// PrintRecommendations 打印改进建议
func (p *Printer) PrintRecommendations(result *analyzer.AnalysisResult) {
	fmt.Fprintln(p.out, colorBold+"💡 Recommendations"+colorReset)
	fmt.Fprintln(p.out, strings.Repeat("-", 40))

	recommendations := make(map[string]bool)

	for _, pod := range result.Pods {
		// 基于状态的建议
		switch pod.Status {
		case analyzer.StatusError:
			recommendations["Check pod events: kubectl describe pod "+pod.Name] = true
		case analyzer.StatusPending:
			if strings.Contains(pod.Reason, "Unschedulable") {
				recommendations["Check node resources and taints"] = true
			}
			if strings.Contains(pod.Reason, "ImagePull") {
				recommendations["Verify image name and pull secrets"] = true
			}
		case analyzer.StatusWarning:
			if pod.Restarts > 10 {
				recommendations["Investigate high restart count - check logs: kubectl logs "+pod.Name+" --previous"] = true
			}
			if strings.Contains(pod.Reason, "CrashLoopBackOff") {
				recommendations["Container keeps crashing - check application logs and resource limits"] = true
			}
		}

		// 基于配置问题的建议
		for _, issue := range pod.ConfigIssues {
			switch issue {
			case analyzer.IssueMissingRequests:
				recommendations["Set resource requests to enable proper scheduling"] = true
			case analyzer.IssueMissingLimits:
				recommendations["Set resource limits to prevent resource exhaustion"] = true
			case analyzer.IssueNoProbe:
				recommendations["Add liveness/readiness probes for better health checking"] = true
			}
		}
	}

	if len(recommendations) == 0 {
		fmt.Fprintln(p.out, colorGreen+"  ✓ No specific recommendations"+colorReset)
	} else {
		for rec := range recommendations {
			fmt.Fprintf(p.out, "  • %s\n", rec)
		}
	}
	fmt.Fprintln(p.out)
}

// getStatusColor 返回状态对应的颜色代码
func (p *Printer) getStatusColor(status analyzer.PodStatus) string {
	switch status {
	case analyzer.StatusHealthy:
		return colorGreen
	case analyzer.StatusWarning:
		return colorYellow
	case analyzer.StatusError:
		return colorRed
	case analyzer.StatusPending:
		return colorBlue
	default:
		return colorReset
	}
}

// getStatusIcon 返回状态对应的图标
func (p *Printer) getStatusIcon(status analyzer.PodStatus) string {
	switch status {
	case analyzer.StatusHealthy:
		return "✓ "
	case analyzer.StatusWarning:
		return "⚠ "
	case analyzer.StatusError:
		return "✗ "
	case analyzer.StatusPending:
		return "◷ "
	default:
		return "? "
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
