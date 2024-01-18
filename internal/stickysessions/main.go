// Package stickysessions implements a sticky session load balancer for echo's reverse proxy middleware.
package stickysessions

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	discoveryV1 "k8s.io/api/discovery/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8sEndpointSliceWatcher interface {
	AddEventHandler(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

type K8sEndpointSliceLister interface {
	List(labels.Selector) ([]*discoveryV1.EndpointSlice, error)
}

type StickySessionBalancer struct {
	Service           string
	Namespace         string
	ContainerPortName string
	CookieName        string
	CookiePath        string
	store             *EndpointStore
	cache             K8sEndpointSliceLister
	watcher           K8sEndpointSliceWatcher
}

func NewStickySessionBalancer(ctx context.Context, service string, namespace string, containerPortName string, cookiePath string, cookieName string) middleware.ProxyBalancer {
	slog.Info(
		fmt.Sprintf(
			"Setting up sticky session balancer for service %s port %s in namespace %s",
			service,
			containerPortName,
			namespace,
		),
	)
	var clientConfig *rest.Config
	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		slog.Info("Cannot find in-cluster config, looking for kubeconfig file")
		var kubeconfigPath string
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
		flag.Parse()

		clientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			slog.Error("cannot find a k8s config in kubeconfig path", "error", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		slog.Error("cannot make a new clientset for sticky sessions", "error", err)
		os.Exit(1)
	}

	listOptions := func(options *metaV1.ListOptions) {
		options.LabelSelector = fmt.Sprintf("kubernetes.io/service-name=%s", service)
	}
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Second*30, informers.WithNamespace(namespace), informers.WithTweakListOptions(listOptions))
	endpointInformer := factory.Discovery().V1().EndpointSlices()

	cache := endpointInformer.Lister()
	watcher := endpointInformer.Informer()

	factory.Start(ctx.Done())
	synced := factory.WaitForCacheSync(ctx.Done())
	for v, ok := range synced {
		if !ok {
			slog.Error("caches failed to sync", "error", v)
			os.Exit(1)
		}
	}

	slices, err := cache.List(labels.Everything())
	if err != nil {
		slog.Error("listing caches failed", "error", err)
		os.Exit(1)
	}
	slog.Info(fmt.Sprintf("Endpoint slices found: %v", slices))

	endpointStore := NewEndpointStoreFromEndpointSlices(slices, containerPortName)

	balancer := StickySessionBalancer{
		Service:           service,
		Namespace:         namespace,
		ContainerPortName: containerPortName,
		CookieName:        cookieName,
		CookiePath:        cookiePath,
		store:             endpointStore,
		cache:             cache,
		watcher:           watcher,
	}
	balancer.watcher.AddEventHandler(balancer)
	return &balancer
}

// AddTarget is required by the LoadBalancer interface.
// It is a no-op because the list of proxy targets is managed by a k8s event listener.
func (s *StickySessionBalancer) AddTarget(target *middleware.ProxyTarget) bool {
	return true
}

// RemoveTarget is required by the LoadBalancer interface.
// It is a no-op because the list of proxy targets is managed by a k8s event listener.
func (s *StickySessionBalancer) RemoveTarget(name string) bool {
	return true
}

// Next assigns a proxy target to every incoming request that is received by the proxy
// and the load balancer.
func (s *StickySessionBalancer) Next(c echo.Context) *middleware.ProxyTarget {
	var upstream EndpointStoreItem
	var upstreamFound bool
	var cookie *http.Cookie
	cookie, err := c.Request().Cookie(s.CookieName)
	if err == http.ErrNoCookie {
		// The cookie does not exist at all
		upstream, found := s.store.Peek()
		if !found {
			slog.Info(
				fmt.Sprintf(
					"Receiving sticky session request (%s/%s), no cookie found, no new suitable upstream found, failing the request",
					s.Namespace,
					s.Service,
				),
			)
			c.String(http.StatusBadGateway, "no upstream servers are available")
			return &middleware.ProxyTarget{URL: &url.URL{}}
		}
		s.store.IncrementSessions(upstream.UID, 1)
		cookie = &http.Cookie{
			Name:  s.CookieName,
			Path:  s.CookiePath,
			Value: upstream.UID,
		}
		c.SetCookie(cookie)
		slog.Info(
			fmt.Sprintf(
				"Receiving sticky session request (%s/%s), no cookie found, assigned upstream with uid %s",
				s.Namespace, 
				s.Service,
				upstream.UID,
			),
		)
		return upstream.ProxyTarget()
	}
	upstream, upstreamFound = s.store.Get(cookie.Value)
	if !upstreamFound {
		// The cookie exists but is pointing to an upstream that does not exist
		upstream, upstreamFound = s.store.Peek()
		if !upstreamFound {
			slog.Info(
				fmt.Sprintf(
					"Receiving sticky session request (%s/%s), cookie found but not upstream, no new suitable upstream found, failing the request",
					s.Namespace,
					s.Service,
				),
			)
			c.String(http.StatusBadGateway, "no upstream servers are available")
			return &middleware.ProxyTarget{URL: &url.URL{}}
		}
		s.store.IncrementSessions(upstream.UID, 1)
		cookie = &http.Cookie{
			Name:  s.CookieName,
			Path:  s.CookiePath,
			Value: upstream.UID,
		}
		c.SetCookie(cookie)
		slog.Info(
			fmt.Sprintf(
				"Receiving sticky session request (%s/%s), cookie found but not upstream, assigned upstream with uid %s",s.Namespace,
				s.Service,
				upstream.UID,
			),
		)
	}
	return upstream.ProxyTarget()
}

func (s StickySessionBalancer) OnAdd(obj any, isInitialList bool) {
	slice := obj.(*discoveryV1.EndpointSlice)
	endpointItems := NewEndpointStoreItems(slice, s.ContainerPortName)
	for _, endpointItem := range endpointItems {
		if s.store.Exists(endpointItem.UID) || !endpointItem.Ready {
			continue
		}
		s.store.Add(endpointItem)
	}
}

func (s StickySessionBalancer) OnUpdate(oldObj, newObj any) {
	newStore := NewEndpointStoreFromEndpointItems(NewEndpointStoreItems(newObj.(*discoveryV1.EndpointSlice), s.ContainerPortName), true)
	for _, newItem := range newStore.List() {
		if !newItem.Ready {
			// An endpoint is not ready to receive traffic, remove from store
			s.store.Remove(newItem.UID)
			continue
		}
		oldItem, oldItemFound := s.store.Get(newItem.UID)
		if !oldItemFound {
			// A new endpoint has been added the slice
			s.store.Add(newItem)
			continue
		}
		if oldItem.Host != newItem.Host {
			// An existing endpoint has changed its address
			s.store.UpdateHost(newItem.UID, newItem.Host)
		}
	}
	for _, oldItem := range s.store.List() {
		_, found := newStore.Get(oldItem.UID)
		if !found {
			// An old endpoiint has been removed
			s.store.Remove(oldItem.UID)
		}
	}
}

func (s StickySessionBalancer) OnDelete(obj any) {
	removedSlice := obj.(*discoveryV1.EndpointSlice)
	for _, endpoint := range removedSlice.Endpoints {
		uid := string(endpoint.TargetRef.UID)
		s.store.Remove(uid)
	}
}
