package dsl

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultNfsInstanceHost             = "nfs.instance.local"
	DefaultNfsInstancePath             = "/path"
	DefaultGcpNfsInstanceFileShareName = "vol1"
	DefaultGcpNfsInstanceCapacityGb    = 1024
	DefaultGcpNfsInstanceConnectMode   = "PRIVATE_SERVICE_ACCESS"
	DefaultGcpNfsInstanceTier          = "ZONAL"
)

func WithNfsInstanceStatusHost(host string) ObjStatusAction {
	return &objStatusAction{
		f: func(obj client.Object) {
			if host == "" {
				host = DefaultNfsInstanceHost
			}
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				if len(x.Status.Hosts) == 0 {
					x.Status.Hosts = []string{host}
					x.Status.Host = host
				}
			}
		},
	}
}

func WithNfsInstanceStatusCapacity(capacity resource.Quantity) ObjStatusAction {
	return &objStatusAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				if !x.Status.Capacity.Equal(capacity) {
					x.Status.Capacity = capacity
				}
			}
		},
	}
}

func WithNfsInstanceStatusPath(path string) ObjStatusAction {
	return &objStatusAction{
		f: func(obj client.Object) {
			if path == "" {
				path = DefaultNfsInstancePath
			}
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				if x.Status.Path == "" {
					x.Status.Path = path
				}
			}
		},
	}
}

func WithNfsInstanceStatusId(id string) ObjStatusAction {
	return &objStatusAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				if len(x.Status.Id) == 0 {
					x.Status.Id = id
				}
			}
		},
	}
}

func WithNfsInstanceCapacity(v resource.Quantity) ObjStatusAction {
	return &objStatusAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				if x.Status.Capacity.IsZero() {
					x.Status.Capacity = v
				}
			}
		},
	}
}

func WithNfsInstanceAws() ObjAction {
	return &objAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				x.Spec.Instance.Aws = &cloudcontrolv1beta1.NfsInstanceAws{}
			}
		},
	}
}

func WithNfsInstanceSap(sizeGb int) ObjAction {
	return &objAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				x.Spec.Instance.OpenStack = &cloudcontrolv1beta1.NfsInstanceOpenStack{
					SizeGb: sizeGb,
				}
			}
		},
	}
}

func WithNfsInstanceGcp(location string) ObjAction {
	return &objAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				x.Spec.Instance.Gcp = &cloudcontrolv1beta1.NfsInstanceGcp{}
				x.Spec.Instance.Gcp.ConnectMode = DefaultGcpNfsInstanceConnectMode
				x.Spec.Instance.Gcp.CapacityGb = DefaultGcpNfsInstanceCapacityGb
				x.Spec.Instance.Gcp.FileShareName = DefaultGcpNfsInstanceFileShareName
				x.Spec.Instance.Gcp.Location = location
				x.Spec.Instance.Gcp.Tier = DefaultGcpNfsInstanceTier
			}
		},
	}
}

func WithSourceBackup(backupPath string) ObjAction {
	return &objAction{
		f: func(obj client.Object) {
			if x, ok := obj.(*cloudcontrolv1beta1.NfsInstance); ok {
				x.Spec.Instance.Gcp.SourceBackup = backupPath
			}
		},
	}
}

func CreateNfsInstance(ctx context.Context, clnt client.Client, obj *cloudcontrolv1beta1.NfsInstance, opts ...ObjAction) error {
	if obj == nil {
		obj = &cloudcontrolv1beta1.NfsInstance{}
	}
	NewObjActions(opts...).
		Append(
			WithNamespace(DefaultKcpNamespace),
		).
		ApplyOnObject(obj)

	if obj.Name == "" {
		return errors.New("the KCP NfsInstance must have name set")
	}

	err := clnt.Create(ctx, obj)
	return err
}

func GivenNfsInstanceExists(ctx context.Context, clnt client.Client, obj *cloudcontrolv1beta1.NfsInstance, opts ...ObjAction) error {
	if obj == nil {
		obj = &cloudcontrolv1beta1.NfsInstance{}
	}
	NewObjActions(opts...).
		Append(
			WithNamespace(DefaultKcpNamespace),
		).
		ApplyOnObject(obj)

	if obj.Name == "" {
		return errors.New("the KCP NfsInstance must have name set")
	}

	err := clnt.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	if apierrors.IsNotFound(err) {
		err = clnt.Create(ctx, obj)
	} else {
		err = clnt.Update(ctx, obj)
	}
	return err
}

func UpdateNfsInstance(ctx context.Context, clnt client.Client, obj *cloudcontrolv1beta1.NfsInstance, opts ...ObjAction) error {
	if obj == nil {
		return errors.New("for updating the KCP NfsInstance, the object must be provided")
	}
	obj.Spec.Instance.Gcp.CapacityGb = 2 * DefaultGcpNfsInstanceCapacityGb
	NewObjActions(opts...).
		Append(
			WithNamespace(DefaultKcpNamespace),
		).
		ApplyOnObject(obj)

	if obj.Name == "" {
		return errors.New("the KCP NfsInstance must have name set")
	}

	err := clnt.Update(ctx, obj)
	return err
}

func DeleteNfsInstance(ctx context.Context, clnt client.Client, obj *cloudcontrolv1beta1.NfsInstance, opts ...ObjAction) error {
	if obj == nil {
		return errors.New("for deleting the KCP NfsInstance, the object must be provided")
	}
	NewObjActions(opts...).
		Append(
			WithNamespace(DefaultKcpNamespace),
		).
		ApplyOnObject(obj)

	if obj.Name == "" {
		return errors.New("the KCP NfsInstance must have name set")
	}

	err := clnt.Delete(ctx, obj)
	return err
}

func HavingNfsInstanceStatusId() ObjAssertion {
	return func(obj client.Object) error {
		x, ok := obj.(*cloudcontrolv1beta1.NfsInstance)
		if !ok {
			return fmt.Errorf("the object %T is not KCP NfsInstance", obj)
		}
		if x.Status.Id == "" {
			return errors.New("the KCP NfsInstance status.id is not set")
		}
		return nil
	}
}

func HavingNfsInstanceStatusCapacity(capacity string) ObjAssertion {
	return func(obj client.Object) error {
		x, ok := obj.(*cloudcontrolv1beta1.NfsInstance)
		if !ok {
			return fmt.Errorf("the object %T is not KCP NfsInstance", obj)
		}
		if x.Status.Capacity.IsZero() {
			return errors.New("the KCP NfsInstance status.capacity is not set")
		}
		if x.Status.Capacity.String() != capacity {
			return fmt.Errorf("the KCP NfsInstance status.capacity is %s, but expected %s", x.Status.Capacity.String(), capacity)
		}
		return nil
	}
}
