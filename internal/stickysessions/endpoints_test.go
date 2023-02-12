package stickysessions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEndpointStoreFromEndpointItems(t *testing.T) {
	type TestCase struct {
		Input          []EndpointStoreItem
		ExpectedOutput []EndpointStoreItem
	}
	testCases := []TestCase{
		{
			Input:          []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 1}, {Host: "host2", index: 1, Sessions: 2}, {Host: "host3", index: 2, Sessions: 3}},
			ExpectedOutput: []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 1}, {Host: "host2", index: 1, Sessions: 2}, {Host: "host3", index: 2, Sessions: 3}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 100}, {Host: "host2", index: 1, Sessions: 5}, {Host: "host3", index: 2, Sessions: 10}},
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", index: 0, Sessions: 5}, {Host: "host3", index: 1, Sessions: 10}, {Host: "host1", index: 2, Sessions: 100}},
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.Input), func(t *testing.T) {
			endpoints := NewEndpointStoreFromEndpointItems(testCase.Input, true)
			assert.Equal(t, testCase.ExpectedOutput, endpoints.List())
		})
	}
}

func TestRemove(t *testing.T) {
	type TestCase struct {
		Input          []EndpointStoreItem
		Remove         string
		ExpectedOutput []EndpointStoreItem
	}
	testCases := []TestCase{
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 1}, {Host: "host2", UID: "host2", index: 1, Sessions: 2}, {Host: "host3", UID: "host3", index: 2, Sessions: 3}},
			Remove:         "host1",
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", UID: "host2", index: 0, Sessions: 2}, {Host: "host3", UID: "host3", index: 1, Sessions: 3}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 100}, {Host: "host2", UID: "host2", index: 1, Sessions: 5}, {Host: "host3", UID: "host3", index: 2, Sessions: 10}},
			Remove:         "host1",
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", UID: "host2", index: 0, Sessions: 5}, {Host: "host3", UID: "host3", index: 1, Sessions: 10}},
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.Input), func(t *testing.T) {
			endpoints := NewEndpointStoreFromEndpointItems(testCase.Input, true)
			endpoints.Remove(testCase.Remove)
			assert.Equal(t, testCase.ExpectedOutput, endpoints.List())
		})
	}
}

func TestAdd(t *testing.T) {
	type TestCase struct {
		Input          []EndpointStoreItem
		Add            EndpointStoreItem
		ExpectedOutput []EndpointStoreItem
	}
	testCases := []TestCase{
		{
			Input:          []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 1}, {Host: "host2", index: 1, Sessions: 2}, {Host: "host3", index: 2, Sessions: 3}},
			Add:            EndpointStoreItem{Host: "host5"},
			ExpectedOutput: []EndpointStoreItem{{Host: "host5", index: 0, Sessions: 0}, {Host: "host1", index: 1, Sessions: 1}, {Host: "host2", index: 2, Sessions: 2}, {Host: "host3", index: 3, Sessions: 3}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 100}, {Host: "host2", index: 1, Sessions: 5}, {Host: "host3", index: 2, Sessions: 10}},
			Add:            EndpointStoreItem{Host: "host5", Sessions: 1000},
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", index: 0, Sessions: 5}, {Host: "host3", index: 1, Sessions: 10}, {Host: "host1", index: 2, Sessions: 100}, {Host: "host5", Sessions: 1000, index: 3}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", index: 0, Sessions: 100}, {Host: "host2", index: 1, Sessions: 5}, {Host: "host3", index: 2, Sessions: 10}},
			Add:            EndpointStoreItem{Host: "host5", Sessions: 6},
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", index: 0, Sessions: 5}, {Host: "host5", Sessions: 6, index: 1}, {Host: "host3", index: 2, Sessions: 10}, {Host: "host1", index: 3, Sessions: 100}},
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.Input), func(t *testing.T) {
			endpoints := NewEndpointStoreFromEndpointItems(testCase.Input, true)
			endpoints.Add(testCase.Add)
			assert.Equal(t, testCase.ExpectedOutput, endpoints.List())
		})
	}
}

func TestUpdate(t *testing.T) {
	type TestCase struct {
		Input          []EndpointStoreItem
		ExpectedOutput []EndpointStoreItem
		UpdateName     string
		UpdateValue    int
	}
	testCases := []TestCase{
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 1}, {Host: "host2", UID: "host2", index: 1, Sessions: 2}, {Host: "host3", UID: "host3", index: 2, Sessions: 3}},
			UpdateName:     "host2",
			UpdateValue:    20,
			ExpectedOutput: []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 1}, {Host: "host3", UID: "host3", index: 1, Sessions: 3}, {Host: "host2", UID: "host2", index: 2, Sessions: 22}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 100}, {Host: "host2", UID: "host2", index: 1, Sessions: 5}, {Host: "host3", UID: "host3", index: 2, Sessions: 10}},
			UpdateName:     "host2",
			UpdateValue:    2,
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", UID: "host2", index: 0, Sessions: 7}, {Host: "host3", UID: "host3", index: 1, Sessions: 10}, {Host: "host1", UID: "host1", index: 2, Sessions: 100}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 100}, {Host: "host2", UID: "host2", index: 1, Sessions: 5}, {Host: "host3", UID: "host3", index: 2, Sessions: 10}},
			UpdateName:     "host1",
			UpdateValue:    200,
			ExpectedOutput: []EndpointStoreItem{{Host: "host2", UID: "host2", index: 0, Sessions: 5}, {Host: "host3", UID: "host3", index: 1, Sessions: 10}, {Host: "host1", UID: "host1", index: 2, Sessions: 300}},
		},
		{
			Input:          []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 100}, {Host: "host2", UID: "host2", index: 1, Sessions: 5}, {Host: "host3", UID: "host3", index: 2, Sessions: 10}},
			UpdateName:     "host1",
			UpdateValue:    -100,
			ExpectedOutput: []EndpointStoreItem{{Host: "host1", UID: "host1", index: 0, Sessions: 0}, {Host: "host2", UID: "host2", index: 1, Sessions: 5}, {Host: "host3", UID: "host3", index: 2, Sessions: 10}},
		},
	}
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.Input), func(t *testing.T) {
			endpoints := NewEndpointStoreFromEndpointItems(testCase.Input, true)
			assert.True(t, endpoints.IncrementSessions(testCase.UpdateName, testCase.UpdateValue))
			assert.Equal(t, testCase.ExpectedOutput, endpoints.List())
		})
	}
}
