package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/iftekarqureshi/k8s-namespace-cleaner/pkg/cleaner"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	dryRun := false
	if len(os.Args) > 1 && (os.Args[1] == "--dry-run" || os.Args[1] == "-d") {
		dryRun = true
	}

	config, err := restConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	c := &cleaner.Cleaner{
		Client: cleaner.NewK8sClient(clientset),
		Now:    time.Now(),
		DryRun: dryRun,
		Logger: func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		},
	}

	results, err := c.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	deleted := 0
	for _, r := range results {
		if r.Deleted || (r.DryRun && r.Reason != "" && !r.Deleted) {
			if r.Deleted {
				deleted++
			}
		}
	}
	fmt.Printf("done: %d namespace(s) processed, %d deleted\n", len(results), deleted)
}

func restConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeconfig.ClientConfig()
}
