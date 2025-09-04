/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package network

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverGenericNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	clusterNetwork, err := discoverNetwork(ctx, client)
	if err != nil {
		return nil, err
	}

	if clusterNetwork != nil {
		clusterNetwork.NetworkPlugin = cni.Generic
		return clusterNetwork, nil
	}

	return nil, nil
}

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	clusterNetwork := &ClusterNetwork{}

	podIPRange, err := findPodIPRange(ctx, client)
	if err != nil {
		return nil, err
	}

	if podIPRange != "" {
		clusterNetwork.PodCIDRs = []string{podIPRange}
	}

	clusterNetwork.ServiceCIDRs, err = discoverGenericServiceCIDRs(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(clusterNetwork.PodCIDRs) > 0 || len(clusterNetwork.ServiceCIDRs) > 0 {
		return clusterNetwork, nil
	}

	return nil, nil
}

func discoverGenericServiceCIDRs(ctx context.Context, client controllerClient.Client) ([]string, error) {
	serviceCIDR := &networkingv1.ServiceCIDR{}

	err := client.Get(ctx, controllerClient.ObjectKey{Name: "kubernetes"}, serviceCIDR)
	if err == nil {
		return serviceCIDR.Spec.CIDRs, nil
	}

	if !resource.IsNotFoundErr(err) {
		return nil, errors.Wrap(err, "error retrieving kubernetes ServiceCIDR")
	}

	clusterIPRange, err := findClusterIPRangeFromApiserver(ctx, client)
	if err == nil && clusterIPRange == "" {
		clusterIPRange, err = findClusterIPRangeFromKubeController(ctx, client)
	}

	if err != nil || clusterIPRange == "" {
		return nil, err
	}

	return []string{clusterIPRange}, nil
}

func findClusterIPRangeFromApiserver(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-apiserver", "--service-cluster-ip-range")
}

func findClusterIPRangeFromKubeController(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-controller-manager", "--service-cluster-ip-range")
}

func findPodIPRange(ctx context.Context, client controllerClient.Client) (string, error) {
	podIPRange, err := findPodIPRangeFromKubeController(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromKubeProxy(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromNodeSpec(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	return "", nil
}

func findPodIPRangeFromKubeController(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-controller-manager", "--cluster-cidr")
}

func findPodIPRangeFromKubeProxy(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "k8s-app=kube-proxy", "--cluster-cidr")
}

func findPodIPRangeFromNodeSpec(ctx context.Context, client controllerClient.Client) (string, error) {
	nodes := &corev1.NodeList{}

	err := client.List(ctx, nodes)
	if err != nil {
		return "", errors.WithMessagef(err, "error listing nodes")
	}

	return parseToPodCidr(nodes.Items)
}

func parseToPodCidr(nodes []corev1.Node) (string, error) {
	// In K8s, each node is typically assigned a unique PodCIDR range for the pods that run on that node.
	// Each node's PodCIDR is used to allocate IP addresses to the pods scheduled on that node. Only if
	// the cluster is a single node deployment, we should rely on the node.Spec.PodCIDR as podCIDR of the cluster.
	if len(nodes) == 1 {
		return nodes[0].Spec.PodCIDR, nil
	}

	return "", nil
}
