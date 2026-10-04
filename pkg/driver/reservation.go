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
	"context"
	"fmt"
	"time"

	"github.com/kubernetes-sigs/dra-driver-cpu/internal/ctxlog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/cpuset"
)

const reservationCheckTimeout = 30 * time.Second

// cpuReservationMismatch returns a description of the mismatch between
// reservedCPUs and the kubelet's CPU reservation (capacity - allocatable),
// or an empty string if they match.
func cpuReservationMismatch(node *v1.Node, reserved cpuset.CPUSet) string {
	capacity, ok := node.Status.Capacity[v1.ResourceCPU]
	if !ok {
		return ""
	}
	allocatable, ok := node.Status.Allocatable[v1.ResourceCPU]
	if !ok {
		return ""
	}
	kubeletReservedMilli := capacity.MilliValue() - allocatable.MilliValue()
	driverReservedMilli := int64(reserved.Size()) * 1000
	if kubeletReservedMilli == driverReservedMilli {
		return ""
	}
	return fmt.Sprintf("the kubelet reserves %dm CPU (capacity %s, allocatable %s) but the driver reserves %d CPUs (%q); "+
		"set the kubelet's reservation (kubeReserved + systemReserved, or reservedSystemCPUs) to match reservedCPUs",
		kubeletReservedMilli, capacity.String(), allocatable.String(), reserved.Size(), reserved.String())
}

// checkCPUReservation logs a warning if reservedCPUs and the kubelet's CPU
// reservation disagree.
func (cp *CPUDriver) checkCPUReservation(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)
	if cp.kubeClient == nil || cp.nodeName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, reservationCheckTimeout)
	defer cancel()
	node, err := cp.kubeClient.CoreV1().Nodes().Get(ctx, cp.nodeName, metav1.GetOptions{})
	if err != nil {
		logger.V(2).Info("cannot check the CPU reservation against the kubelet", "node", cp.nodeName, "err", err)
		return
	}
	if msg := cpuReservationMismatch(node, cp.topology.ReservedCPUs); msg != "" {
		logger.Info("WARNING: CPU reservation mismatch", "node", cp.nodeName, "detail", msg)
	}
}
