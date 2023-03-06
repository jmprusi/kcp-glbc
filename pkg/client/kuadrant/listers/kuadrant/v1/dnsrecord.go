
//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by kcp code-generator. DO NOT EDIT.

package v1

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"	
	"github.com/kcp-dev/logicalcluster/v3"
	
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"

	kuadrantv1 "github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1"
	)

// DNSRecordClusterLister can list DNSRecords across all workspaces, or scope down to a DNSRecordLister for one workspace.
// All objects returned here must be treated as read-only.
type DNSRecordClusterLister interface {
	// List lists all DNSRecords in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error)
	// Cluster returns a lister that can list and get DNSRecords in one workspace.
Cluster(clusterName logicalcluster.Name) DNSRecordLister
DNSRecordClusterListerExpansion
}

type dNSRecordClusterLister struct {
	indexer cache.Indexer
}

// NewDNSRecordClusterLister returns a new DNSRecordClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewDNSRecordClusterLister(indexer cache.Indexer) *dNSRecordClusterLister {
	return &dNSRecordClusterLister{indexer: indexer}
}

// List lists all DNSRecords in the indexer across all workspaces.
func (s *dNSRecordClusterLister) List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*kuadrantv1.DNSRecord))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get DNSRecords.
func (s *dNSRecordClusterLister) Cluster(clusterName logicalcluster.Name) DNSRecordLister {
return &dNSRecordLister{indexer: s.indexer, clusterName: clusterName}
}

// DNSRecordLister can list DNSRecords across all namespaces, or scope down to a DNSRecordNamespaceLister for one namespace.
// All objects returned here must be treated as read-only.
type DNSRecordLister interface {
	// List lists all DNSRecords in the workspace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error)
// DNSRecords returns a lister that can list and get DNSRecords in one workspace and namespace.
	DNSRecords(namespace string) DNSRecordNamespaceLister
DNSRecordListerExpansion
}
// dNSRecordLister can list all DNSRecords inside a workspace or scope down to a DNSRecordLister for one namespace.
type dNSRecordLister struct {
	indexer cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all DNSRecords in the indexer for a workspace.
func (s *dNSRecordLister) List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*kuadrantv1.DNSRecord))
	})
	return ret, err
}

// DNSRecords returns an object that can list and get DNSRecords in one namespace.
func (s *dNSRecordLister) DNSRecords(namespace string) DNSRecordNamespaceLister {
return &dNSRecordNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// dNSRecordNamespaceLister helps list and get DNSRecords.
// All objects returned here must be treated as read-only.
type DNSRecordNamespaceLister interface {
	// List lists all DNSRecords in the workspace and namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error)
	// Get retrieves the DNSRecord from the indexer for a given workspace, namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*kuadrantv1.DNSRecord, error)
	DNSRecordNamespaceListerExpansion
}
// dNSRecordNamespaceLister helps list and get DNSRecords.
// All objects returned here must be treated as read-only.
type dNSRecordNamespaceLister struct {
	indexer   cache.Indexer
	clusterName   logicalcluster.Name
	namespace string
}

// List lists all DNSRecords in the indexer for a given workspace and namespace.
func (s *dNSRecordNamespaceLister) List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*kuadrantv1.DNSRecord))
	})
	return ret, err
}

// Get retrieves the DNSRecord from the indexer for a given workspace, namespace and name.
func (s *dNSRecordNamespaceLister) Get(name string) (*kuadrantv1.DNSRecord, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(kuadrantv1.Resource("DNSRecord"), name)
	}
	return obj.(*kuadrantv1.DNSRecord), nil
}
// NewDNSRecordLister returns a new DNSRecordLister.
// We assume that the indexer:
// - is fed by a workspace-scoped LIST+WATCH
// - uses cache.MetaNamespaceKeyFunc as the key function
// - has the cache.NamespaceIndex as an index
func NewDNSRecordLister(indexer cache.Indexer) *dNSRecordScopedLister {
	return &dNSRecordScopedLister{indexer: indexer}
}

// dNSRecordScopedLister can list all DNSRecords inside a workspace or scope down to a DNSRecordLister for one namespace.
type dNSRecordScopedLister struct {
	indexer cache.Indexer
}

// List lists all DNSRecords in the indexer for a workspace.
func (s *dNSRecordScopedLister) List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error) {
	err = cache.ListAll(s.indexer, selector, func(i interface{}) {
		ret = append(ret, i.(*kuadrantv1.DNSRecord))
	})
	return ret, err
}

// DNSRecords returns an object that can list and get DNSRecords in one namespace.
func (s *dNSRecordScopedLister) DNSRecords(namespace string) DNSRecordNamespaceLister {
	return &dNSRecordScopedNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// dNSRecordScopedNamespaceLister helps list and get DNSRecords.
type dNSRecordScopedNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DNSRecords in the indexer for a given workspace and namespace.
func (s *dNSRecordScopedNamespaceLister) List(selector labels.Selector) (ret []*kuadrantv1.DNSRecord, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*kuadrantv1.DNSRecord))
	})
	return ret, err
}

// Get retrieves the DNSRecord from the indexer for a given workspace, namespace and name.
func (s *dNSRecordScopedNamespaceLister) Get(name string) (*kuadrantv1.DNSRecord, error) {
	key := s.namespace + "/" + name
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(kuadrantv1.Resource("DNSRecord"), name)
	}
	return obj.(*kuadrantv1.DNSRecord), nil
}
