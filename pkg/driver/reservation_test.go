/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/cpuset"
)

func nodeWithCPU(capacity, allocatable string) *v1.Node {
	node := &v1.Node{}
	if capacity != "" {
		node.Status.Capacity = v1.ResourceList{v1.ResourceCPU: resource.MustParse(capacity)}
	}
	if allocatable != "" {
		node.Status.Allocatable = v1.ResourceList{v1.ResourceCPU: resource.MustParse(allocatable)}
	}
	return node
}

func TestCPUReservationMismatch(t *testing.T) {
	tests := []struct {
		name     string
		node     *v1.Node
		reserved cpuset.CPUSet
		mismatch bool
	}{
		{
			name:     "match",
			node:     nodeWithCPU("16", "14"),
			reserved: cpuset.New(0, 1),
		},
		{
			name:     "match with nothing reserved",
			node:     nodeWithCPU("16", "16"),
			reserved: cpuset.New(),
		},
		{
			name:     "kubelet reserves a fraction, driver a whole CPU",
			node:     nodeWithCPU("16", "15900m"),
			reserved: cpuset.New(0),
			mismatch: true,
		},
		{
			name:     "kubelet reserves less than the driver",
			node:     nodeWithCPU("16", "15"),
			reserved: cpuset.New(0, 1),
			mismatch: true,
		},
		{
			name:     "kubelet reserves more than the driver",
			node:     nodeWithCPU("16", "13"),
			reserved: cpuset.New(0, 1),
			mismatch: true,
		},
		{
			name:     "node reports no cpu capacity",
			node:     nodeWithCPU("", ""),
			reserved: cpuset.New(0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := cpuReservationMismatch(tc.node, tc.reserved)
			if tc.mismatch {
				require.NotEmpty(t, msg)
			} else {
				require.Empty(t, msg)
			}
		})
	}
}
