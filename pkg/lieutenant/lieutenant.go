package lieutenant

import (
	"context"
	"fmt"

	lieutenantv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	Client    client.Client
	Namespace string
}

type Config struct {
	Host      string
	Token     string
	Namespace string
}

func NewLieutenantClient(config Config) (*Client, error) {
	scheme := runtime.NewScheme()
	err := lieutenantv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("could not create new lieutenant client: %w", err)
	}

	conf := &rest.Config{
		Host:        config.Host, // yes this is the correct field, host accepts a url
		BearerToken: config.Token,
	}
	c, err := client.New(conf, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		panic(err)
	}
	return &Client{
		Client:    c,
		Namespace: config.Namespace,
	}, nil
}

func (l *Client) GetClusterFacts(ctx context.Context, cluster_id string) (map[string]string, error) {
	var cluster lieutenantv1alpha1.Cluster

	nsn := client.ObjectKey{
		Namespace: l.Namespace,
		Name:      cluster_id,
	}

	if err := l.Client.Get(ctx, nsn, &cluster); err != nil {
		return nil, err
	}

	return cluster.Spec.Facts, nil
}
