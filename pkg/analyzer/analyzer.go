package analyzer

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// PodStatus 表示 Pod 的状态分类
type PodStatus string

const (
	StatusHealthy PodStatus = "Healthy"
	StatusWarning PodStatus = "Warning"
	StatusError   PodStatus = "Error"
	StatusPending PodStatus = "Pending"
	StatusUnknown PodStatus = "Unknown"
)

// ConfigIssue 表示配置问题
type ConfigIssue string

const (
	IssueMissingRequests ConfigIssue = "Missing resource requests"
	IssueMissingLimits   ConfigIssue = "Missing resource limits"
	IssueNoProbe         ConfigIssue = "Missing health probe"
)

// ECI 相关的标签和注解
const (
	// 阿里云 ECI Pod 的标识
	ECINodeLabelKey    = "type"
	ECINodeLabelValue  = "virtual-kubelet"
	ECIPodAnnotation   = "k8s.aliyun.com/eci-instance-id"
	ECINodeNamePrefix  = "virtual-kubelet"
	VirtualKubeletType = "virtual-kubelet"
)

// PodAnalysis 包含单个 Pod 的分析结果
type PodAnalysis struct {
	Name          string
	Namespace     string
	Status        PodStatus
	Phase         corev1.PodPhase
	Ready         string // "2/2" 格式
	Restarts      int32
	Age           string
	RunningTime   string        // Pod 实际运行时间（从 Running 开始计算）
	Reason        string        // 如果有问题，说明原因
	ConfigIssues  []ConfigIssue // 配置问题列表
	ContainerInfo []ContainerAnalysis
	IsECI         bool   // 是否运行在 ECI 上
	ECIInstanceID string // ECI 实例 ID
	NodeName      string // 节点名称
}

// ContainerAnalysis 包含容器级别的分析
type ContainerAnalysis struct {
	Name            string
	Ready           bool
	RestartCount    int32
	LastTermination string // 上次终止原因
	HasRequests     bool
	HasLimits       bool
	HasProbe        bool
}

// AnalysisResult 包含整体分析结果
type AnalysisResult struct {
	Pods             []PodAnalysis
	TotalPods        int
	HealthyPods      int
	WarningPods      int
	ErrorPods        int
	PendingPods      int
	TotalRestarts    int32
	ConfigIssueCount int
	ECIPodCount      int // ECI Pod 数量
}

// HasIssues 检查是否有任何问题
func (r *AnalysisResult) HasIssues() bool {
	return r.ErrorPods > 0 || r.WarningPods > 0 || r.ConfigIssueCount > 0
}

// AnalyzePods 分析 Pod 列表
func AnalyzePods(pods *corev1.PodList, checkConfig bool) *AnalysisResult {
	result := &AnalysisResult{
		Pods:      make([]PodAnalysis, 0, len(pods.Items)),
		TotalPods: len(pods.Items),
	}

	for _, pod := range pods.Items {
		analysis := analyzeSinglePod(&pod, checkConfig)
		result.Pods = append(result.Pods, analysis)

		// 更新统计
		result.TotalRestarts += analysis.Restarts
		if analysis.IsECI {
			result.ECIPodCount++
		}
		switch analysis.Status {
		case StatusHealthy:
			result.HealthyPods++
		case StatusWarning:
			result.WarningPods++
		case StatusError:
			result.ErrorPods++
		case StatusPending:
			result.PendingPods++
		}
		result.ConfigIssueCount += len(analysis.ConfigIssues)
	}

	return result
}

// analyzeSinglePod 分析单个 Pod
func analyzeSinglePod(pod *corev1.Pod, checkConfig bool) PodAnalysis {
	analysis := PodAnalysis{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Phase:     pod.Status.Phase,
		Age:       formatAge(pod.CreationTimestamp.Time),
		NodeName:  pod.Spec.NodeName,
	}

	// 检测是否是 ECI Pod
	analysis.IsECI, analysis.ECIInstanceID = detectECI(pod)

	// 计算运行时间（从容器实际开始运行算起）
	analysis.RunningTime = calculateRunningTime(pod)

	// 分析容器状态
	readyCount := 0
	totalCount := len(pod.Spec.Containers)
	var totalRestarts int32 = 0

	for i, container := range pod.Spec.Containers {
		containerAnalysis := analyzeContainer(&container, pod, i, checkConfig)
		analysis.ContainerInfo = append(analysis.ContainerInfo, containerAnalysis)

		if containerAnalysis.Ready {
			readyCount++
		}
		totalRestarts += containerAnalysis.RestartCount

		// 收集配置问题
		if checkConfig {
			if !containerAnalysis.HasRequests {
				analysis.ConfigIssues = appendIfNotExists(analysis.ConfigIssues, IssueMissingRequests)
			}
			if !containerAnalysis.HasLimits {
				analysis.ConfigIssues = appendIfNotExists(analysis.ConfigIssues, IssueMissingLimits)
			}
			if !containerAnalysis.HasProbe {
				analysis.ConfigIssues = appendIfNotExists(analysis.ConfigIssues, IssueNoProbe)
			}
		}
	}

	analysis.Ready = fmt.Sprintf("%d/%d", readyCount, totalCount)
	analysis.Restarts = totalRestarts

	// 确定整体状态
	analysis.Status, analysis.Reason = determinePodStatus(pod, readyCount, totalCount, totalRestarts)

	return analysis
}

// detectECI 检测 Pod 是否运行在 ECI 上
func detectECI(pod *corev1.Pod) (bool, string) {
	// 方法1: 检查 ECI 实例 ID 注解（最可靠）
	if eciID, ok := pod.Annotations[ECIPodAnnotation]; ok && eciID != "" {
		return true, eciID
	}

	// 方法2: 检查节点名是否包含 virtual-kubelet
	if strings.Contains(strings.ToLower(pod.Spec.NodeName), ECINodeNamePrefix) {
		return true, ""
	}

	// 方法3: 检查其他常见的 ECI 相关注解
	eciAnnotations := []string{
		"k8s.aliyun.com/eci-instance-spec",
		"k8s.aliyun.com/eci-use-specs",
		"alibabacloud.com/eci",
	}
	for _, anno := range eciAnnotations {
		if _, ok := pod.Annotations[anno]; ok {
			return true, ""
		}
	}

	return false, ""
}

// calculateRunningTime 计算 Pod 实际运行时间
func calculateRunningTime(pod *corev1.Pod) string {
	// 如果 Pod 不在 Running 状态，返回 "-"
	if pod.Status.Phase != corev1.PodRunning {
		return "-"
	}

	// 尝试从容器状态获取最早的启动时间
	var earliestStart *time.Time

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil {
			startTime := cs.State.Running.StartedAt.Time
			if earliestStart == nil || startTime.Before(*earliestStart) {
				earliestStart = &startTime
			}
		}
	}

	// 如果没有找到运行中的容器，使用 Pod 的 Ready condition 时间
	if earliestStart == nil {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				earliestStart = &cond.LastTransitionTime.Time
				break
			}
		}
	}

	// 如果还是没有，返回 Age
	if earliestStart == nil {
		return formatAge(pod.CreationTimestamp.Time)
	}

	return formatAge(*earliestStart)
}

// analyzeContainer 分析单个容器
func analyzeContainer(container *corev1.Container, pod *corev1.Pod, index int, checkConfig bool) ContainerAnalysis {
	analysis := ContainerAnalysis{
		Name: container.Name,
	}

	// 查找对应的容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == container.Name {
			analysis.Ready = cs.Ready
			analysis.RestartCount = cs.RestartCount

			// 检查上次终止原因
			if cs.LastTerminationState.Terminated != nil {
				term := cs.LastTerminationState.Terminated
				analysis.LastTermination = fmt.Sprintf("%s (exit: %d)", term.Reason, term.ExitCode)
			}
			break
		}
	}

	// 检查资源配置
	if checkConfig {
		resources := container.Resources
		analysis.HasRequests = len(resources.Requests) > 0
		analysis.HasLimits = len(resources.Limits) > 0
		analysis.HasProbe = container.LivenessProbe != nil || container.ReadinessProbe != nil
	}

	return analysis
}

// determinePodStatus 根据各种条件确定 Pod 状态
func determinePodStatus(pod *corev1.Pod, readyCount, totalCount int, restarts int32) (PodStatus, string) {
	// 检查 Pod Phase
	switch pod.Status.Phase {
	case corev1.PodPending:
		reason := getPendingReason(pod)
		return StatusPending, reason
	case corev1.PodFailed:
		return StatusError, getFailedReason(pod)
	case corev1.PodUnknown:
		return StatusUnknown, "Pod status unknown"
	}

	// Pod 在 Running 状态，检查容器是否都 Ready
	if readyCount < totalCount {
		reason := getNotReadyReason(pod)
		return StatusWarning, reason
	}

	// 检查重启次数
	if restarts > 10 {
		return StatusWarning, fmt.Sprintf("High restart count: %d", restarts)
	}

	// 检查是否有异常的容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return StatusWarning, cs.State.Waiting.Reason
		}
	}

	return StatusHealthy, ""
}

// getPendingReason 获取 Pod Pending 的原因
func getPendingReason(pod *corev1.Pod) string {
	// 检查 Pod Conditions
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
			return fmt.Sprintf("Unschedulable: %s", cond.Message)
		}
	}

	// 检查容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			return cs.State.Waiting.Reason
		}
	}

	// 检查 init 容器
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil {
			return fmt.Sprintf("Init:%s", cs.State.Waiting.Reason)
		}
		if cs.State.Running != nil {
			return fmt.Sprintf("Init:%s running", cs.Name)
		}
	}

	return "Pending"
}

// getFailedReason 获取 Pod 失败的原因
func getFailedReason(pod *corev1.Pod) string {
	if pod.Status.Reason != "" {
		return pod.Status.Reason
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil {
			return fmt.Sprintf("%s (exit: %d)", cs.State.Terminated.Reason, cs.State.Terminated.ExitCode)
		}
	}

	return "Failed"
}

// getNotReadyReason 获取容器未就绪的原因
func getNotReadyReason(pod *corev1.Pod) string {
	var reasons []string

	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				reasons = append(reasons, cs.State.Waiting.Reason)
			} else if cs.State.Running != nil {
				reasons = append(reasons, "NotReady")
			}
		}
	}

	if len(reasons) > 0 {
		return strings.Join(reasons, ", ")
	}
	return "Containers not ready"
}

// formatAge 格式化时间为易读的 age 格式
func formatAge(t time.Time) string {
	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

// appendIfNotExists 如果不存在则追加
func appendIfNotExists(slice []ConfigIssue, item ConfigIssue) []ConfigIssue {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}
