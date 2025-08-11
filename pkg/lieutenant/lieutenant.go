package lieutenant

import (
	"context"
	"fmt"

	lieutenantv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type client struct {
	Client    k8sClient.Client
	Namespace string
}

type Config struct {
	Host      string
	Token     string
	Namespace string
}

func NewLieutenantClient(config Config) (*client, error) {
	scheme := runtime.NewScheme()
	err := lieutenantv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("could not create new lieutenant client: %w", err)
	}

	conf := &rest.Config{
		Host:        config.Host, // yes this is the correct field, host accepts a url
		BearerToken: config.Token,
	}
	c, err := k8sClient.New(conf, k8sClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	return &client{
		Client:    c,
		Namespace: config.Namespace,
	}, nil
}

func (l *client) GetClusterFacts(ctx context.Context, cluster_id string) (map[string]string, error) {
	var cluster lieutenantv1alpha1.Cluster

	nsn := k8sClient.ObjectKey{
		Namespace: l.Namespace,
		Name:      cluster_id,
	}

	if err := l.Client.Get(ctx, nsn, &cluster); err != nil {
		return nil, err
	}

	return cluster.Spec.Facts, nil
}
