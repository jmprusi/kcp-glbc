package secret

import (
	"context"

	kcpinformers "github.com/kcp-dev/client-go/informers"
	"github.com/kcp-dev/client-go/kubernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kcpcorev1listers "github.com/kcp-dev/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/go-logr/logr"
	"github.com/kcp-dev/logicalcluster/v3"

	"github.com/kuadrant/kcp-glbc/pkg/migration/workload"
	"github.com/kuadrant/kcp-glbc/pkg/reconciler"
	basereconciler "github.com/kuadrant/kcp-glbc/pkg/reconciler"
)

const defaultControllerName = "kcp-glbc-secret"

// NewController returns a new Controller which reconciles Secrets.
func NewController(config *ControllerConfig) (*Controller, error) {
	controllerName := config.GetName(defaultControllerName)
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName)
	c := &Controller{
		Controller:            basereconciler.NewController(controllerName, queue),
		coreClient:            config.SecretsClient,
		sharedInformerFactory: config.SharedInformerFactory,
	}
	c.Process = c.process
	c.migrationHandler = workload.Migrate

	c.sharedInformerFactory.Core().V1().Secrets().Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			s := obj.(*v1.Secret)
			if s.Labels != nil {
				if _, hcgmanaged := s.Labels[basereconciler.LABEL_HCG_MANAGED]; hcgmanaged {
					return true
				}
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { c.Enqueue(obj) },
			UpdateFunc: func(_, obj interface{}) { c.Enqueue(obj) },
			DeleteFunc: func(obj interface{}) { c.Enqueue(obj) },
		},
	})

	c.indexer = c.sharedInformerFactory.Core().V1().Secrets().Informer().GetIndexer()
	c.secretLister = c.sharedInformerFactory.Core().V1().Secrets().Lister()

	return c, nil
}

type ControllerConfig struct {
	*reconciler.ControllerConfig
	SecretsClient         kubernetes.ClusterInterface
	SharedInformerFactory kcpinformers.SharedInformerFactory
}

type Controller struct {
	*reconciler.Controller
	sharedInformerFactory kcpinformers.SharedInformerFactory
	coreClient            kubernetes.ClusterInterface
	indexer               cache.Indexer
	secretLister          kcpcorev1listers.SecretClusterLister
	migrationHandler      func(obj metav1.Object, queue workqueue.RateLimitingInterface, logger logr.Logger)
}

func (c *Controller) process(ctx context.Context, key string) error {
	object, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		c.Logger.Info("Secret was deleted", "key", key)
		return nil
	}

	current := object.(*corev1.Secret)
	target := current.DeepCopy()

	if err = c.reconcile(ctx, target); err != nil {
		return err
	}

	if !equality.Semantic.DeepEqual(target, current) {
		_, err := c.coreClient.Cluster(logicalcluster.From(target).Path()).CoreV1().Secrets(target.Namespace).Update(ctx, target, metav1.UpdateOptions{})
		return err
	}

	return nil
}
