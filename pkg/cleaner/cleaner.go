package cleaner

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	AnnotationJanitorTTL   = "janitor/ttl"
	AnnotationIQStudioTTL  = "iqstudio.dev/ttl"
)

// NamespaceClient lists and deletes namespaces.
type NamespaceClient interface {
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) error
}

type k8sClient struct {
	client kubernetes.Interface
}

func NewK8sClient(client kubernetes.Interface) NamespaceClient {
	return &k8sClient{client: client}
}

func (k *k8sClient) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	list, err := k.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *k8sClient) DeleteNamespace(ctx context.Context, name string) error {
	return k.client.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

// Result describes a namespace cleanup action.
type Result struct {
	Name      string
	Reason    string
	Deleted   bool
	DryRun    bool
}

// Cleaner evaluates namespaces and optionally deletes expired ones.
type Cleaner struct {
	Client  NamespaceClient
	Now     time.Time
	DryRun  bool
	Logger  func(format string, args ...interface{})
}

func (c *Cleaner) Run(ctx context.Context) ([]Result, error) {
	namespaces, err := c.Client.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	var results []Result
	for _, ns := range namespaces {
		if isSystemNamespace(ns.Name) {
			continue
		}

		expired, reason, err := isExpired(ns, c.Now)
		if err != nil {
			results = append(results, Result{Name: ns.Name, Reason: fmt.Sprintf("skip: %v", err)})
			continue
		}
		if !expired {
			continue
		}

		result := Result{Name: ns.Name, Reason: reason, DryRun: c.DryRun}
		if c.DryRun {
			c.log("dry-run: would delete namespace %s (%s)", ns.Name, reason)
		} else {
			if err := c.Client.DeleteNamespace(ctx, ns.Name); err != nil {
				result.Reason = fmt.Sprintf("delete failed: %v", err)
			} else {
				result.Deleted = true
				c.log("deleted namespace %s (%s)", ns.Name, reason)
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func (c *Cleaner) log(format string, args ...interface{}) {
	if c.Logger != nil {
		c.Logger(format, args...)
	}
}

func isSystemNamespace(name string) bool {
	switch name {
	case "default", "kube-system", "kube-public", "kube-node-lease":
		return true
	}
	return false
}

func isExpired(ns corev1.Namespace, now time.Time) (bool, string, error) {
	annotations := ns.Annotations
	if annotations == nil {
		return false, "", nil
	}

	if ttl, ok := annotations[AnnotationJanitorTTL]; ok {
		expiry, err := time.Parse(time.RFC3339, ttl)
		if err != nil {
			return false, "", fmt.Errorf("invalid %s: %w", AnnotationJanitorTTL, err)
		}
		if now.After(expiry) {
			return true, fmt.Sprintf("%s expired at %s", AnnotationJanitorTTL, expiry.Format(time.RFC3339)), nil
		}
		return false, "", nil
	}

	if durationStr, ok := annotations[AnnotationIQStudioTTL]; ok {
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return false, "", fmt.Errorf("invalid %s: %w", AnnotationIQStudioTTL, err)
		}
		created := ns.CreationTimestamp.Time
		expiry := created.Add(duration)
		if now.After(expiry) {
			return true, fmt.Sprintf("%s=%s expired at %s", AnnotationIQStudioTTL, durationStr, expiry.Format(time.RFC3339)), nil
		}
	}

	return false, "", nil
}
