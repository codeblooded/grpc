// Copyright 2020 gRPC authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindPools(t *testing.T) {
	// Test that pools can be discovered from listing all kubernetes nodes
	poolNames := []string{"drivers", "workers-8core", "workers-32core"}
	nodes := makeNodesInPools(map[string]int{
		poolNames[0]: 1,
		poolNames[1]: 3,
		poolNames[2]: 3,
	})
	mnl := &mockNodeLister{Nodes: nodes}

	pools, err := FindPools(mnl)
	if err != nil {
		t.Fatalf("unexpected error in find pools using node lister: %v", err)
	}
	for _, poolName := range poolNames {
		if _, ok := pools[poolName]; !ok {
			t.Errorf("pool %v not found", poolName)
		}
	}
}

type mockNodeLister struct {
	Nodes []v1.Node
	Error error
}

func (mnl *mockNodeLister) List(_ metav1.ListOptions) (*v1.NodeList, error) {
	if mnl.Error != nil {
		return nil, mnl.Error
	}

	list := &v1.NodeList{
		Items: mnl.Nodes,
	}

	return list, nil
}

// makeNodesInPools accepts a map with pool names as the string and the number of nodes as the
// value. It returns a slice of fake kubernetes nodes with the appropriate pool labels.
//
// For example, 7 node instances can be generated with labels for 2 pools (3 with A, 4 with B):
//
//	nodes := makeNodesInPools(map[string]int{
//		"A": 3,
//		"B": 4,
//	})
func makeNodesInPools(nodeMap map[string]int) []v1.Node {
	var nodes []v1.Node

	for name, count := range nodeMap {
		for i := 0; i < count; i++ {
			nodes = append(nodes, v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"pool": name,
					},
				},
			})
		}
	}

	return nodes
}
