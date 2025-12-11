package client

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client 封装了 Kubernetes 客户端操作
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient 创建一个新的 Kubernetes 客户端
// 优先级: 指定的 kubeconfig > KUBECONFIG 环境变量 > ~/.kube/config > in-cluster config
func NewClient(kubeconfigPath string) (*Client, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{clientset: clientset}, nil
}

// buildConfig 构建 Kubernetes 配置
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	// 1. 如果指定了 kubeconfig 路径，使用它
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// 2. 检查 KUBECONFIG 环境变量
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// 3. 尝试默认的 ~/.kube/config
	if home, err := os.UserHomeDir(); err == nil {
		kubeconfig := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err == nil {
			return clientcmd.BuildConfigFromFlags("", kubeconfig)
		}
	}

	// 4. 尝试 in-cluster 配置（在 Pod 内运行时）
	return rest.InClusterConfig()
}

// GetPods 获取指定命名空间的所有 Pod
func (c *Client) GetPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

// GetPod 获取单个 Pod
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetEvents 获取指定 Pod 的事件
func (c *Client) GetEvents(ctx context.Context, namespace, podName string) (*corev1.EventList, error) {
	fieldSelector := "involvedObject.name=" + podName
	return c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}
