package cleaner

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsExpiredJanitorTTL(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dev-test",
			Annotations: map[string]string{
				AnnotationJanitorTTL: "2026-06-16T12:00:00Z",
			},
		},
	}
	expired, reason, err := isExpired(ns, now)
	if err != nil {
		t.Fatal(err)
	}
	if !expired {
		t.Fatal("expected expired")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestIsExpiredIQStudioTTL(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "feature-branch",
			CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
			Annotations: map[string]string{
				AnnotationIQStudioTTL: "1h",
			},
		},
	}
	expired, _, err := isExpired(ns, now)
	if err != nil {
		t.Fatal(err)
	}
	if !expired {
		t.Fatal("expected expired")
	}
}

func TestCleanerDryRun(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "stale-ns",
				Annotations: map[string]string{
					AnnotationJanitorTTL: "2026-06-01T00:00:00Z",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
		},
	)

	c := &Cleaner{
		Client: NewK8sClient(client),
		Now:    now,
		DryRun: true,
	}
	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "stale-ns" {
		t.Fatalf("unexpected namespace: %s", results[0].Name)
	}
	if results[0].Deleted {
		t.Fatal("dry-run should not delete")
	}

	// Namespace should still exist
	_, err = client.CoreV1().Namespaces().Get(context.Background(), "stale-ns", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("namespace should still exist: %v", err)
	}
}

func TestCleanerDeletesExpired(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "stale-ns",
				Annotations: map[string]string{
					AnnotationJanitorTTL: "2026-06-01T00:00:00Z",
				},
			},
		},
	)

	c := &Cleaner{
		Client: NewK8sClient(client),
		Now:    now,
		DryRun: false,
	}
	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Deleted {
		t.Fatalf("expected deleted result, got %+v", results)
	}
}
