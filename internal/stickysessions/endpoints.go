package stickysessions

import (
	"fmt"
	"net/url"
	"sort"
	"sync"

	"github.com/labstack/echo/v4/middleware"
	discoveryV1 "k8s.io/api/discovery/v1"
)

// EndpointStoreItem defines a single endpoint that requests are proxied to
type EndpointStoreItem struct {
	UID      string
	Host     string
	Sessions int
	Ready    bool
	index    int
}

// ProxyTarget returns a proxy target ready to be use in the echo reverse proxy middleware
func (e *EndpointStoreItem) ProxyTarget() *middleware.ProxyTarget {
	url := url.URL{Host: e.Host, Scheme: "http"}
	return &middleware.ProxyTarget{
		Name: e.UID,
		URL:  &url,
	}
}

func NewEndpointStoreItem(input discoveryV1.Endpoint, containerPort int) EndpointStoreItem {
	ready := *input.Conditions.Ready && *input.Conditions.Serving && !*input.Conditions.Terminating
	return EndpointStoreItem{
		Host: fmt.Sprintf("%s:%d", input.Addresses[0], containerPort), UID: string(input.TargetRef.UID), Ready: ready,
	}
}

func NewEndpointStoreItems(input *discoveryV1.EndpointSlice, containerPortName string) []EndpointStoreItem {
	output := []EndpointStoreItem{}
	var portNumber = 80
	for _, port := range input.Ports {
		if *port.Name == containerPortName {
			portNumber = int(*port.Port)
			break
		}
	}
	for _, endpoint := range input.Endpoints {
		item := NewEndpointStoreItem(endpoint, portNumber)
		output = append(output, item)
	}
	return output
}

// EndpointStore is a collection of many EndpointStoreItems
type EndpointStore struct {
	list  []*EndpointStoreItem
	index map[string]*EndpointStoreItem
	mutex *sync.RWMutex
}

// NewEndpointStoreFromEndpointItems creates an endpoint store from a slice of items
// The items' index fields will be overwritten accordingly.
func NewEndpointStoreFromEndpointItems(input []EndpointStoreItem, includeNonReady bool) *EndpointStore {
	endpoints := &EndpointStore{
		list:  make([]*EndpointStoreItem, 0, len(input)),
		index: make(map[string]*EndpointStoreItem, len(input)),
		mutex: &sync.RWMutex{},
	}
	endpoints.mutex.Lock()
	defer endpoints.mutex.Unlock()
	for _, endpoint := range input {
		if !includeNonReady && !endpoint.Ready {
			continue
		}
		// NOTE: Make a copy of loop variable, golang uses the same pointer for the whole loop
		newEndpoint := endpoint
		endpoints.list = append(endpoints.list, &newEndpoint)
		endpoints.index[endpoint.UID] = &newEndpoint
	}
	sort.Sort(endpoints)
	return endpoints
}

// NewEndpointStoreFromEndpointSlices creates an endpoint store from a list of k8s EndpointSlices
// The items' index fields will be overwritten accordingly.
func NewEndpointStoreFromEndpointSlices(input []*discoveryV1.EndpointSlice, containerPortName string) *EndpointStore {
	items := []EndpointStoreItem{}
	for _, endpointSlice := range input {
		es := endpointSlice
		items = append(items, NewEndpointStoreItems(es, containerPortName)...)
	}
	return NewEndpointStoreFromEndpointItems(items, false)
}

// Peek shows the Endpoint with the least sticky sessions assignments
func (e *EndpointStore) Peek() (EndpointStoreItem, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.Len() == 0 {
		return EndpointStoreItem{}, false
	}
	return *e.list[0], true
}

// List returns a list of all endpoints
func (e *EndpointStore) List() []EndpointStoreItem {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	output := make([]EndpointStoreItem, len(e.list))
	for i, endpoint := range e.list {
		output[i] = *endpoint
	}
	return output
}

// Remove an endpoint that is no longer available
func (e *EndpointStore) Remove(name string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	itemsLen := len(e.list)
	if itemsLen == 0 {
		return
	}
	endpoint, found := e.index[name]
	if !found {
		return
	}
	ind := endpoint.index
	oldList := e.list
	if ind == itemsLen-1 {
		e.list = append([]*EndpointStoreItem{}, oldList[:ind]...)
	} else {
		e.list = append([]*EndpointStoreItem{}, oldList[:ind]...)
		e.list = append(e.list, oldList[ind+1:]...)
	}
	delete(e.index, name)
	for i := ind; i < itemsLen-1; i++ {
		e.list[i].index = e.list[i].index - 1
	}
}

// Add an endpoint that has just become available
func (e *EndpointStore) Add(endpoint EndpointStoreItem) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	itemsLen := len(e.list)
	endpoint.index = itemsLen
	e.list = append(e.list, &endpoint)
	e.index[endpoint.UID] = &endpoint
	sort.Sort(e)
}

// IncrementSessions increments the number of sessions for an endpoint that has just become available
func (e *EndpointStore) IncrementSessions(name string, inc int) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	endpoint, found := e.index[name]
	if !found {
		return false
	}
	endpoint.Sessions += inc
	sort.Sort(e)
	return true
}

// UpdateHost updates the Host of an existing endpoint item in the store over an existing one
func (e *EndpointStore) UpdateHost(name string, host string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	endpoint, found := e.index[name]
	if !found {
		return false
	}
	endpoint.Host = host
	return true
}

// Exists check if an items in the store exists based on the name
func (e *EndpointStore) Exists(name string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	_, found := e.index[name]
	return found
}

// Get returns a copy of an item from the store if it exists based on the name
func (e *EndpointStore) Get(name string) (EndpointStoreItem, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	item, found := e.index[name]
	if !found {
		return EndpointStoreItem{}, false
	}
	return *item, found
}

// UIDs returns all UIDs in the store
func (e *EndpointStore) UIDs() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	keys := make([]string, len(e.index))
	i := 0
	for k := range e.index {
		keys[i] = k
		i++
	}
	return keys
}

// Not thread safe, locking has to be handled outside of Len
func (e EndpointStore) Len() int {
	if e.list == nil {
		return 0
	}
	return len(e.list)
}

// Not thread safe, locking has to be handled outside of Swap
func (e EndpointStore) Swap(i, j int) {
	if e.list == nil {
		return
	}
	e.list[i], e.list[j] = e.list[j], e.list[i]
	e.list[i].index = i
	e.list[j].index = j
}

// Not thread safe, locking has to be handled outside of Less
func (e EndpointStore) Less(i, j int) bool {
	if e.list == nil {
		return false
	}
	return e.list[i].Sessions < e.list[j].Sessions
}
