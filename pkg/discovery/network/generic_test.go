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

package network_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testPodCIDR     = "1.2.3.4/16"
	testServiceCIDR = "4.5.6.7/16"
)

var _ = Describe("Generic Network", func() {
	var clusterNet *network.ClusterNetwork

	When("there is a kube-controller-manager pod with the expected pod and service CIDR parameters", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeKubeControllerManagerPod(),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with both CIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

		It("should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("there is a kube-controller-manager pod with the pod and service CIDR parameters passed as Args", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakePodWithArg("kube-controller-manager", []string{"kube-controller-manager"},
					[]string{"--cluster-cidr=" + testPodCIDR, "--service-cluster-ip-range=" + testServiceCIDR}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with both CIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

		It("should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("there is a kube-proxy pod with the expected pod CIDR parameter", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeKubeProxyPod(),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with the pod CIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(BeEmpty())
		})

		It("should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("there is a kube-apiserver pod with the expected service CIDR parameter", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeKubeAPIServerPod(),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(BeEmpty())
		})

		It("should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("there are kube-proxy and kube-apiserver pods with the expected CIDR parameters", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeKubeProxyPod(),
				fakeKubeAPIServerPod(),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with both CIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("should identify the network plugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("the kubernetes ServiceCIDR resource exists", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(ctx, newServiceCIDR())
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})

		It("should return a ClusterNetwork with the service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

	When("there is a kube-proxy pod but without the expected pod CIDR parameter", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakePod("kube-proxy", []string{"kube-proxy", "--cluster-ABCD=1.2.3.4"}, []corev1.EnvVar{}),
			)
		})

		It("should return no ClusterNetwork", func() {
			Expect(clusterNet).To(BeNil())
		})
	})

	When("there is a kube-controller-manager pod but without the expected pod and service CIDR parameters", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakePod("kube-controller-manager", []string{"kube-controller-manager", "--foo=bar"}, []corev1.EnvVar{}),
			)
		})

		It("should return no ClusterNetwork", func() {
			Expect(clusterNet).To(BeNil())
		})
	})

	When("there is a kube-apiserver pod but without the expected service CIDR parameter", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakePod("kube-apiserver", []string{"kube-apiserver", "--foo=bar"}, []corev1.EnvVar{}),
			)
		})

		It("should return no ClusterNetwork", func() {
			Expect(clusterNet).To(BeNil())
		})
	})

	When("a pod parameter has comma-separated CIDRs", func() {
		const testPodCIDR2 = "5.6.7.8/16"

		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx, fakeKubeControllerManagerPod(testPodCIDR+","+testPodCIDR2),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with all the CIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR, testPodCIDR2}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

	When("pod CIDR information exists on a single node cluster", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeNode("node1", testPodCIDR),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with the pod CIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo(cni.Generic))
		})
	})

	When("no pod CIDR information exists on any node", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeNode("node1", ""),
				fakeNode("node2", ""),
			)
		})

		It("should return no ClusterNetwork", func() {
			Expect(clusterNet).To(BeNil())
		})
	})

	When("pod CIDR information exists on a multi-node cluster", func() {
		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(
				ctx,
				fakeNode("node1", testPodCIDR),
				fakeNode("node2", testPodCIDR),
			)
		})

		It("should return no ClusterNetwork", func() {
			Expect(clusterNet).To(BeNil())
		})
	})

	When("the Submariner resource exists", func() {
		const globalCIDR = "242.112.0.0/24"
		const clustersetIPCIDR = "243.110.0.0/20"

		BeforeEach(func(ctx SpecContext) {
			clusterNet = testDiscoverGenericWith(ctx, &v1alpha1.Submariner{
				ObjectMeta: metav1.ObjectMeta{
					Name: names.SubmarinerCrName,
				},
				Spec: v1alpha1.SubmarinerSpec{
					GlobalCIDR:       globalCIDR,
					ClustersetIPCIDR: clustersetIPCIDR,
				},
			}, newServiceCIDR())
			Expect(clusterNet).NotTo(BeNil())
		})

		It("should return a ClusterNetwork with the global CIDR", func() {
			Expect(clusterNet.GlobalCIDR).To(Equal(globalCIDR))
			clusterNet.Show()
		})

		It("should return a ClusterNetwork with the clustersetIP CIDR", func() {
			Expect(clusterNet.ClustersetIPCIDR).To(Equal(clustersetIPCIDR))
			clusterNet.Show()
		})
	})
})

func testDiscoverGenericWith(ctx context.Context, objects ...controllerClient.Object) *network.ClusterNetwork {
	client := newTestClient(objects...)
	clusterNet, err := network.Discover(ctx, client, "")
	Expect(err).NotTo(HaveOccurred())

	return clusterNet
}

func newTestClient(objects ...controllerClient.Object) controllerClient.Client {
	return fakeClient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
