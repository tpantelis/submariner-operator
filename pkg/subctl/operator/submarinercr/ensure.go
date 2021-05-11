/*
© 2019 Red Hat, Inc. and others.

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

package submarinercr

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	submarinerClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
)

const (
	SubmarinerName = "submariner"
)

func Ensure(config *rest.Config, namespace string, submarinerSpec submariner.SubmarinerSpec) error {
	submarinerCR := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerName,
		},
		Spec: submarinerSpec,
	}

	client, err := submarinerClientset.NewForConfig(config)
	if err != nil {
		return err
	}

	propagationPolicy := metav1.DeletePropagationForeground

	return createAnew(context.TODO(), &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Submariners(namespace).Get(ctx, name, options)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			return client.SubmarinerV1alpha1().Submariners(namespace).Create(ctx, obj.(*submariner.Submariner), options)
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			return client.SubmarinerV1alpha1().Submariners(namespace).Delete(ctx, name, options)
		},
	}, submarinerCR, metav1.CreateOptions{}, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
}

func createAnew(ctx context.Context, client resource.Interface, obj runtime.Object,
	createOptions metav1.CreateOptions, deleteOptions metav1.DeleteOptions) error {

	name := resource.ToMeta(obj).GetName()

	backOff := wait.Backoff{
		Steps:    10,
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Cap:      10 * time.Minute,
	}

	return wait.ExponentialBackoff(backOff, func() (bool, error) {
		_, err := client.Create(ctx, obj, createOptions)
		klog.Infof("**Create: %v", err)

		if !apierrors.IsAlreadyExists(err) {
			return true, err
		}

		err = client.Delete(ctx, name, deleteOptions)
		klog.Infof("**Delete: %v", err)
		if apierrors.IsNotFound(err) {
			err = nil
		}

		return false, errors.WithMessagef(err, "failed to delete pre-existing instance %q", name)
	})
}
