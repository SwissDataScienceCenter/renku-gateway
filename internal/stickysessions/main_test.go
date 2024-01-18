package stickysessions

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

type MockInformer struct {
	store        []*discoveryV1.EndpointSlice
	eventHandler cache.ResourceEventHandler
	mutex        sync.RWMutex
}

func (m *MockInformer) List(labels.Selector) ([]*discoveryV1.EndpointSlice, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.store, nil
}

func (m *MockInformer) TriggerAdd(obj *discoveryV1.EndpointSlice) {
	func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		m.store = append(m.store, obj)
	}()
	m.eventHandler.OnAdd(obj, true)
}

func (m *MockInformer) TriggerDelete(obj *discoveryV1.EndpointSlice) {
	for i, islice := range m.store {
		if islice.Endpoints[0].TargetRef.UID == obj.Endpoints[0].TargetRef.UID {
			func() {
				m.mutex.Lock()
				defer m.mutex.Unlock()
				newStore := []*discoveryV1.EndpointSlice{}
				newStore = append(newStore, m.store[:i]...)
				if i < len(m.store)-1 {
					newStore = append(newStore, m.store[i+1:]...)
				}
				m.store = newStore
			}()
			m.eventHandler.OnDelete(obj)
			return
		}
	}
}

func (m *MockInformer) TriggerUpdate(oldObj, newObj *discoveryV1.EndpointSlice) {
	for i, islice := range m.store {
		if islice.Endpoints[0].TargetRef.UID == oldObj.Endpoints[0].TargetRef.UID {
			func() {
				m.mutex.Lock()
				defer m.mutex.Unlock()
				m.store[i] = newObj
			}()
			m.eventHandler.OnUpdate(oldObj, newObj)
			return
		}
	}
}

func (m *MockInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.eventHandler = handler
	return nil, nil
}

func setupTestUpstream(ID string) (*httptest.Server, *url.URL, discoveryV1.EndpointSlice) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, ID)
	}))
	url, err := url.Parse(srv.URL)
	if err != nil {
		log.Fatal(err)
	}
	trueVal := true
	falseVal := false
	portName := "http"
	portNumber, err := strconv.ParseInt(url.Port(), 10, 32)
	if err != nil {
		log.Fatal(err)
	}
	portNumber32 := int32(portNumber)
	endpointSlice := discoveryV1.EndpointSlice{
		Endpoints: []discoveryV1.Endpoint{
			{
				Addresses: []string{url.Hostname()},
				Conditions: discoveryV1.EndpointConditions{
					Ready:       &trueVal,
					Serving:     &trueVal,
					Terminating: &falseVal,
				},
				TargetRef: &coreV1.ObjectReference{
					UID: types.UID(ID),
				},
			},
		},
		Ports: []discoveryV1.EndpointPort{{Name: &portName, Port: &portNumber32}},
	}
	return srv, url, endpointSlice
}

func TestUpstreamSelection(t *testing.T) {
	type TestCase struct {
		RequestCookies          []http.Cookie
		UpstreamPortName        string
		UpstreamIDs             []string
		SessionCookieName       string
		ExpectedResponseStatus  int
		ExpectedResponseContent string
		ExpectedResponseCookies map[string]string
	}

	runTest := func(t *testing.T, testCase TestCase) {
		endpointItems := []EndpointStoreItem{}
		endpointSlices := []*discoveryV1.EndpointSlice{}
		for _, i := range testCase.UpstreamIDs {
			server, _, endpoint := setupTestUpstream(i)
			defer server.Close()
			endpointSlices = append(endpointSlices, &endpoint)
			endpointItems = append(endpointItems, NewEndpointStoreItems(&endpoint, testCase.UpstreamPortName)...)
		}
		store := NewEndpointStoreFromEndpointItems(endpointItems, false)
		mockInformer := MockInformer{
			mutex: sync.RWMutex{},
			store: endpointSlices,
		}
		balancer := StickySessionBalancer{
			ContainerPortName: testCase.UpstreamPortName,
			CookieName:        testCase.SessionCookieName,
			cache:             &mockInformer,
			watcher:           &mockInformer,
			store:             store,
		}
		e := echo.New()
		proxy := middleware.ProxyWithConfig(middleware.ProxyConfig{
			Balancer: &balancer,
		})
		proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		proxyPort := proxyListener.Addr().(*net.TCPAddr).Port
		e.Listener = proxyListener
		defer proxyListener.Close()
		e.Use(middleware.Logger())
		e.Group("/").Use(proxy)
		go e.Start(fmt.Sprintf(":%d", proxyPort))
		defer e.Close()
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", proxyPort), nil)
		assert.NoError(t, err)
		for _, c := range testCase.RequestCookies {
			aCookie := c
			req.AddCookie(&aCookie)
		}
		res, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, testCase.ExpectedResponseStatus, res.StatusCode)
		resContent, err := io.ReadAll(res.Body)
		defer res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, testCase.ExpectedResponseContent, string(resContent))
		assert.Equal(t, len(testCase.ExpectedResponseCookies), len(res.Cookies()))
		cookiesMatched := 0
		for expectedCookieName, expectedCookieVal := range testCase.ExpectedResponseCookies {
			for _, actualCookie := range res.Cookies() {
				if actualCookie.Name == expectedCookieName {
					assert.Equal(t, expectedCookieVal, actualCookie.Value)
					cookiesMatched++
				}
			}
		}
		assert.Equal(t, len(testCase.ExpectedResponseCookies), cookiesMatched)
	}

	testCases := []TestCase{
		{
			UpstreamPortName:        "http",
			UpstreamIDs:             []string{"host1"},
			RequestCookies:          []http.Cookie{},
			SessionCookieName:       "session-cookie",
			ExpectedResponseStatus:  http.StatusOK,
			ExpectedResponseContent: "host1",
			ExpectedResponseCookies: map[string]string{"session-cookie": "host1"},
		},
		{
			UpstreamPortName:        "http",
			UpstreamIDs:             []string{"host1", "host2"},
			RequestCookies:          []http.Cookie{{Name: "session-cookie", Value: "host2"}},
			SessionCookieName:       "session-cookie",
			ExpectedResponseStatus:  http.StatusOK,
			ExpectedResponseContent: "host2",
			ExpectedResponseCookies: map[string]string{},
		},
		{
			UpstreamPortName:        "http",
			UpstreamIDs:             []string{"host1"},
			RequestCookies:          []http.Cookie{{Name: "session-cookie", Value: "host2"}},
			SessionCookieName:       "session-cookie",
			ExpectedResponseStatus:  http.StatusOK,
			ExpectedResponseContent: "host1",
			ExpectedResponseCookies: map[string]string{"session-cookie": "host1"},
		},
		{
			UpstreamPortName:        "http",
			UpstreamIDs:             []string{},
			RequestCookies:          []http.Cookie{},
			SessionCookieName:       "session-cookie",
			ExpectedResponseStatus:  http.StatusBadGateway,
			ExpectedResponseContent: "no upstream servers are available",
			ExpectedResponseCookies: map[string]string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase), func(t *testing.T) {
			runTest(t, testCase)
		})
	}
}

func TestEventHandling(t *testing.T) {
	type EventType string
	const (
		Add    EventType = "add"
		Delete EventType = "delete"
		Update EventType = "update"
	)
	type Event struct {
		Old  *discoveryV1.EndpointSlice
		New  *discoveryV1.EndpointSlice
		Type EventType
	}
	type TestCase struct {
		UpstreamIDs       []string
		Events            []Event
		ExpectedEndpoints []EndpointStoreItem
	}

	runTest := func(t *testing.T, testCase TestCase) {
		endpointItems := []EndpointStoreItem{}
		endpointSlices := []*discoveryV1.EndpointSlice{}
		for _, i := range testCase.UpstreamIDs {
			server, _, endpoint := setupTestUpstream(i)
			defer server.Close()
			endpointSlices = append(endpointSlices, &endpoint)
			endpointItems = append(endpointItems, NewEndpointStoreItems(&endpoint, "http")...)
		}
		store := NewEndpointStoreFromEndpointItems(endpointItems, false)
		mockInformer := MockInformer{
			mutex: sync.RWMutex{},
			store: endpointSlices,
		}
		balancer := StickySessionBalancer{
			ContainerPortName: "http",
			cache:             &mockInformer,
			watcher:           &mockInformer,
			store:             store,
		}
		balancer.watcher.AddEventHandler(balancer)
		for _, event := range testCase.Events {
			switch event.Type {
			case Add:
				mockInformer.TriggerAdd(event.New)
			case Delete:
				mockInformer.TriggerDelete(event.Old)
			case Update:
				mockInformer.TriggerUpdate(event.Old, event.New)
			}
		}
		assert.Equal(t, len(testCase.ExpectedEndpoints), len(balancer.store.List()))
		for _, expectedEndpoint := range testCase.ExpectedEndpoints {
			actualEndpoint, found := balancer.store.Get(expectedEndpoint.UID)
			assert.True(t, found)
			assert.Equal(t, expectedEndpoint.UID, actualEndpoint.UID)
			assert.Equal(t, expectedEndpoint.Sessions, actualEndpoint.Sessions)
		}
	}

	trueVal := true
	falseVal := false
	testCases := []TestCase{
		{
			UpstreamIDs:       []string{},
			ExpectedEndpoints: []EndpointStoreItem{},
		},
		{
			UpstreamIDs:       []string{"host1"},
			ExpectedEndpoints: []EndpointStoreItem{{UID: "host1", Sessions: 0}},
		},
		{
			UpstreamIDs:       []string{},
			ExpectedEndpoints: []EndpointStoreItem{{UID: "host2", Sessions: 0}},
			Events: []Event{
				{
					Type: Add,
					New: &discoveryV1.EndpointSlice{
						Endpoints: []discoveryV1.Endpoint{
							{
								Addresses: []string{"host2"},
								TargetRef: &coreV1.ObjectReference{UID: types.UID("host2")},
								Conditions: discoveryV1.EndpointConditions{
									Ready:       &trueVal,
									Serving:     &trueVal,
									Terminating: &falseVal,
								},
							},
						},
					},
				},
			},
		},
		{
			UpstreamIDs:       []string{"host1"},
			ExpectedEndpoints: []EndpointStoreItem{{UID: "host2", Sessions: 0}, {UID: "host1", Sessions: 0}},
			Events: []Event{
				{
					Type: Add,
					New: &discoveryV1.EndpointSlice{
						Endpoints: []discoveryV1.Endpoint{
							{
								Addresses: []string{"host2"},
								TargetRef: &coreV1.ObjectReference{UID: types.UID("host2")},
								Conditions: discoveryV1.EndpointConditions{
									Ready:       &trueVal,
									Serving:     &trueVal,
									Terminating: &falseVal,
								},
							},
						},
					},
				},
			},
		},
		{
			UpstreamIDs:       []string{"host1"},
			ExpectedEndpoints: []EndpointStoreItem{},
			Events: []Event{
				{
					Type: Delete,
					Old: &discoveryV1.EndpointSlice{
						Endpoints: []discoveryV1.Endpoint{
							{
								Addresses: []string{"host1"},
								TargetRef: &coreV1.ObjectReference{UID: types.UID("host1")},
								Conditions: discoveryV1.EndpointConditions{
									Ready:       &trueVal,
									Serving:     &trueVal,
									Terminating: &falseVal,
								},
							},
						},
					},
				},
			},
		},
		{
			UpstreamIDs:       []string{"host1"},
			ExpectedEndpoints: []EndpointStoreItem{},
			Events: []Event{
				{
					Type: Update,
					Old: &discoveryV1.EndpointSlice{
						Endpoints: []discoveryV1.Endpoint{
							{
								Addresses: []string{"host1"},
								TargetRef: &coreV1.ObjectReference{UID: types.UID("host1")},
								Conditions: discoveryV1.EndpointConditions{
									Ready:       &trueVal,
									Serving:     &trueVal,
									Terminating: &falseVal,
								},
							},
						},
					},
					New: &discoveryV1.EndpointSlice{
						Endpoints: []discoveryV1.Endpoint{
							{
								Addresses: []string{"host1"},
								TargetRef: &coreV1.ObjectReference{UID: types.UID("host1")},
								Conditions: discoveryV1.EndpointConditions{
									Ready:       &falseVal,
									Serving:     &trueVal,
									Terminating: &falseVal,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase), func(t *testing.T) {
			runTest(t, testCase)
		})
	}
}
