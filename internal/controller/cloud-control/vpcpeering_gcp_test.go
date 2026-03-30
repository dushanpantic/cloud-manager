package cloudcontrol

import (
	"fmt"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	kcpnetwork "github.com/kyma-project/cloud-manager/pkg/kcp/network"
	kcpscope "github.com/kyma-project/cloud-manager/pkg/kcp/scope"
	. "github.com/kyma-project/cloud-manager/pkg/testinfra/dsl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

var _ = Describe("Feature: KCP VpcPeering", func() {
	It("Scenario: KCP GCP VpcPeering is created", func() {
		const (
			kymaName          = "7e829442-f92e-4205-9d36-0d622a422d74"
			kymaNetworkName   = kymaName + "--kyma"
			kymaVpc           = "shoot-12345-abc"
			remoteNetworkName = "f5331c29-bb1a-439c-8376-94be50232eb4"
			remotePeeringName = "peering-sap-gcp-skr-dev-cust-00002-to-sap-sc-learn"
			remoteVpc         = "default"
			remoteRefName     = "skr-gcp-vpcpeering"
		)

		gcpMock := infra.GcpMock2().NewSubscription("vpc-peering-create")
		defer gcpMock.Delete()

		gcpProject := gcpMock.ProjectId()

		scope := &cloudcontrolv1beta1.Scope{}

		By("Given Scope exists", func() {
			kcpscope.Ignore.AddName(kymaName)

			Eventually(CreateScopeGcp2).
				WithArguments(infra.Ctx(), infra, scope, gcpProject, WithName(kymaName)).
				Should(Succeed())
		})

		By("And Given GCP kyma VPC network exists", func() {
			op, err := gcpMock.InsertNetwork(infra.Ctx(), &computepb.InsertNetworkRequest{
				Project: gcpProject,
				NetworkResource: &computepb.Network{
					Name: ptr.To(kymaVpc),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(op.Wait(infra.Ctx())).To(Succeed())
		})

		By("And Given GCP remote VPC network exists", func() {
			op, err := gcpMock.InsertNetwork(infra.Ctx(), &computepb.InsertNetworkRequest{
				Project: gcpProject,
				NetworkResource: &computepb.Network{
					Name: ptr.To(remoteVpc),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(op.Wait(infra.Ctx())).To(Succeed())
		})

		By("And Given the remote network is tagged", func() {
			tagKeyOp, err := gcpMock.CreateTagKey(infra.Ctx(), &resourcemanagerpb.CreateTagKeyRequest{
				TagKey: &resourcemanagerpb.TagKey{
					Parent:    fmt.Sprintf("projects/%s", gcpProject),
					ShortName: kymaName,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			tagKey, err := tagKeyOp.Wait(infra.Ctx())
			Expect(err).ToNot(HaveOccurred())

			tagValueOp, err := gcpMock.CreateTagValue(infra.Ctx(), &resourcemanagerpb.CreateTagValueRequest{
				TagValue: &resourcemanagerpb.TagValue{
					Parent:    tagKey.Name,
					ShortName: "allowed",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			tagValue, err := tagValueOp.Wait(infra.Ctx())
			Expect(err).ToNot(HaveOccurred())

			remoteNet, err := gcpMock.GetNetwork(infra.Ctx(), &computepb.GetNetworkRequest{
				Project: gcpProject,
				Network: remoteVpc,
			})
			Expect(err).ToNot(HaveOccurred())

			bindingParent := fmt.Sprintf("//compute.googleapis.com/projects/%s/global/networks/%d", gcpProject, remoteNet.GetId())
			_, err = gcpMock.CreateTagBinding(infra.Ctx(), &resourcemanagerpb.CreateTagBindingRequest{
				TagBinding: &resourcemanagerpb.TagBinding{
					Parent:   bindingParent,
					TagValue: tagValue.Name,
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})

		kymaNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: kymaVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeKyma,
			},
		}

		By("And Given Kyma Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(kymaNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), kymaNetwork, WithName(kymaNetworkName), WithScope(scope.Name)).
				Should(Succeed())
		})

		remoteNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: remoteVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeExternal,
			},
		}

		By("And Given Remote Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(remoteNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), remoteNetwork, WithName(remoteNetworkName), WithScope(scope.Name), WithState("Ready")).
				Should(Succeed())
		})

		By("When KCP KymaNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					kymaNetwork,
					WithNetworkStatusNetwork(kymaNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		By("And When KCP RemoteNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					remoteNetwork,
					WithNetworkStatusNetwork(remoteNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		vpcpeering := &cloudcontrolv1beta1.VpcPeering{
			Spec: cloudcontrolv1beta1.VpcPeeringSpec{
				Details: &cloudcontrolv1beta1.VpcPeeringDetails{
					LocalNetwork: klog.ObjectRef{
						Name:      kymaNetwork.Name,
						Namespace: kymaNetwork.Namespace,
					},
					RemoteNetwork: klog.ObjectRef{
						Name:      remoteNetwork.Name,
						Namespace: remoteNetwork.Namespace,
					},
					PeeringName:      remotePeeringName,
					LocalPeeringName: "cm-" + remoteNetworkName,
				},
			},
		}

		By("And When the KCP VpcPeering is created", func() {
			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					WithName(remoteNetworkName),
					WithRemoteRef(remoteRefName),
					WithScope(kymaName),
				).
				Should(Succeed())
		})

		By("Then GCP VpcPeering is created on remote side", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingVpcPeeringStatusRemoteId(),
				).
				Should(Succeed())
		})

		By("And Then GCP VpcPeering is created on kyma side", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingVpcPeeringStatusId(),
				).
				Should(Succeed())
		})

		// In mock2, AddPeering automatically sets both peerings to ACTIVE when
		// peerings exist on both sides, so no manual state setting is needed.

		By("Then KCP VpcPeering has Ready condition", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingConditionTrue(cloudcontrolv1beta1.ConditionTypeReady),
				).
				Should(Succeed())
		})

		// DELETE
		By("When KCP VpcPeering is deleted", func() {
			Eventually(Delete).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "Error deleting VPC Peering")
		})

		By("Then VpcPeering does not exist", func() {
			Eventually(IsDeleted).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "VPC Peering was not deleted")
		})
	})

	It("Scenario: KCP GCP VpcPeering can be deleted due to issues on the remote network", func() {
		const (
			kymaName          = "ec697362-8f63-4423-b34f-8a99c0460d46"
			kymaNetworkName   = kymaName + "--kyma"
			kymaVpc           = "shoot-12345-abc"
			remoteNetworkName = "0ab0eca3-3094-4842-9834-7492aaa0639d"
			remotePeeringName = "peering-with-permission-error"
			remoteVpc         = "remote-vpc"
			remoteRefName     = "skr-gcp-vpcpeering"
		)

		gcpMock := infra.GcpMock2().NewSubscription("vpc-peering-error")
		defer gcpMock.Delete()

		gcpProject := gcpMock.ProjectId()

		scope := &cloudcontrolv1beta1.Scope{}

		By("Given Scope exists", func() {
			kcpscope.Ignore.AddName(kymaName)

			Eventually(CreateScopeGcp2).
				WithArguments(infra.Ctx(), infra, scope, gcpProject, WithName(kymaName)).
				Should(Succeed())
		})

		By("And Given GCP kyma VPC network exists", func() {
			op, err := gcpMock.InsertNetwork(infra.Ctx(), &computepb.InsertNetworkRequest{
				Project: gcpProject,
				NetworkResource: &computepb.Network{
					Name: ptr.To(kymaVpc),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(op.Wait(infra.Ctx())).To(Succeed())
		})

		// No remote VPC network is created in the mock, so GetNetwork for
		// the remote VPC will fail with "not found", triggering the error path.

		kymaNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: kymaVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeKyma,
			},
		}

		By("And Given Kyma Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(kymaNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), kymaNetwork, WithName(kymaNetworkName), WithScope(scope.Name)).
				Should(Succeed())
		})

		remoteNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: remoteVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeExternal,
			},
		}

		By("And Given Remote Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(remoteNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), remoteNetwork, WithName(remoteNetworkName), WithScope(scope.Name), WithState("Ready")).
				Should(Succeed())
		})

		By("And Given KCP KymaNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					kymaNetwork,
					WithNetworkStatusNetwork(kymaNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		By("And Given KCP RemoteNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					remoteNetwork,
					WithNetworkStatusNetwork(remoteNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		vpcpeering := &cloudcontrolv1beta1.VpcPeering{
			Spec: cloudcontrolv1beta1.VpcPeeringSpec{
				Details: &cloudcontrolv1beta1.VpcPeeringDetails{
					LocalNetwork: klog.ObjectRef{
						Name:      kymaNetwork.Name,
						Namespace: kymaNetwork.Namespace,
					},
					RemoteNetwork: klog.ObjectRef{
						Name:      remoteNetwork.Name,
						Namespace: remoteNetwork.Namespace,
					},
					PeeringName:      remotePeeringName,
					LocalPeeringName: "cm-" + remoteNetworkName,
				},
			},
		}

		By("And Given KCP VpcPeering is created", func() {
			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					WithName(remoteNetworkName),
					WithRemoteRef(remoteRefName),
					WithScope(kymaName),
				).
				Should(Succeed())
		})

		By("Then KCP VpcPeering has Error condition", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingConditionTrue(cloudcontrolv1beta1.ConditionTypeError),
				).
				Should(Succeed())
		})

		By("When KCP VpcPeering in error state is deleted", func() {
			Eventually(Delete).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "Error deleting VPC Peering")
		})

		By("Then VpcPeering does not exist", func() {
			Eventually(IsDeleted).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "VPC Peering was not deleted")
		})
	})

	It("Scenario: KCP GCP VpcPeering can be deleted when Networks are not in Ready state", func() {
		const (
			kymaName          = "21445c56-35fa-423a-a8d3-7bd9f3ed4976"
			kymaNetworkName   = kymaName + "--kyma"
			kymaVpc           = "shoot-12345-abc-357"
			remoteNetworkName = "2d10d06f-81f5-4155-adae-1922a9d2dd08"
			remotePeeringName = "peering-with-permission-deleting-with-warning"
			remoteVpc         = "remote-vpc-warning-test"
			remoteRefName     = "skr-gcp-vpcpeering-45"
		)

		gcpMock := infra.GcpMock2().NewSubscription("vpc-peering-warning")
		defer gcpMock.Delete()

		gcpProject := gcpMock.ProjectId()

		scope := &cloudcontrolv1beta1.Scope{}

		By("Given Scope exists", func() {
			kcpscope.Ignore.AddName(kymaName)

			Eventually(CreateScopeGcp2).
				WithArguments(infra.Ctx(), infra, scope, gcpProject, WithName(kymaName)).
				Should(Succeed())
		})

		By("And Given GCP kyma VPC network exists", func() {
			op, err := gcpMock.InsertNetwork(infra.Ctx(), &computepb.InsertNetworkRequest{
				Project: gcpProject,
				NetworkResource: &computepb.Network{
					Name: ptr.To(kymaVpc),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(op.Wait(infra.Ctx())).To(Succeed())
		})

		By("And Given GCP remote VPC network exists", func() {
			op, err := gcpMock.InsertNetwork(infra.Ctx(), &computepb.InsertNetworkRequest{
				Project: gcpProject,
				NetworkResource: &computepb.Network{
					Name: ptr.To(remoteVpc),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(op.Wait(infra.Ctx())).To(Succeed())
		})

		By("And Given the remote network is tagged", func() {
			tagKeyOp, err := gcpMock.CreateTagKey(infra.Ctx(), &resourcemanagerpb.CreateTagKeyRequest{
				TagKey: &resourcemanagerpb.TagKey{
					Parent:    fmt.Sprintf("projects/%s", gcpProject),
					ShortName: kymaName,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			tagKey, err := tagKeyOp.Wait(infra.Ctx())
			Expect(err).ToNot(HaveOccurred())

			tagValueOp, err := gcpMock.CreateTagValue(infra.Ctx(), &resourcemanagerpb.CreateTagValueRequest{
				TagValue: &resourcemanagerpb.TagValue{
					Parent:    tagKey.Name,
					ShortName: "allowed",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			tagValue, err := tagValueOp.Wait(infra.Ctx())
			Expect(err).ToNot(HaveOccurred())

			remoteNet, err := gcpMock.GetNetwork(infra.Ctx(), &computepb.GetNetworkRequest{
				Project: gcpProject,
				Network: remoteVpc,
			})
			Expect(err).ToNot(HaveOccurred())

			bindingParent := fmt.Sprintf("//compute.googleapis.com/projects/%s/global/networks/%d", gcpProject, remoteNet.GetId())
			_, err = gcpMock.CreateTagBinding(infra.Ctx(), &resourcemanagerpb.CreateTagBindingRequest{
				TagBinding: &resourcemanagerpb.TagBinding{
					Parent:   bindingParent,
					TagValue: tagValue.Name,
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})

		kymaNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: kymaVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeKyma,
			},
		}

		By("And Given Kyma Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(kymaNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), kymaNetwork, WithName(kymaNetworkName), WithScope(scope.Name)).
				Should(Succeed())
		})

		remoteNetwork := &cloudcontrolv1beta1.Network{
			Spec: cloudcontrolv1beta1.NetworkSpec{
				Network: cloudcontrolv1beta1.NetworkInfo{
					Reference: &cloudcontrolv1beta1.NetworkReference{
						Gcp: &cloudcontrolv1beta1.GcpNetworkReference{
							GcpProject:  gcpProject,
							NetworkName: remoteVpc,
						},
					},
				},
				Type: cloudcontrolv1beta1.NetworkTypeExternal,
			},
		}

		By("And Given Remote Network exists in KCP", func() {
			kcpnetwork.Ignore.AddName(remoteNetworkName)

			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), remoteNetwork, WithName(remoteNetworkName), WithScope(scope.Name), WithState("Ready")).
				Should(Succeed())
		})

		By("And Given KCP KymaNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					kymaNetwork,
					WithNetworkStatusNetwork(kymaNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		By("And Given KCP RemoteNetwork is Ready", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					remoteNetwork,
					WithNetworkStatusNetwork(remoteNetwork.Spec.Network.Reference),
					WithState("Ready"),
					WithConditions(KcpReadyCondition())).
				Should(Succeed())
		})

		vpcpeering := &cloudcontrolv1beta1.VpcPeering{
			Spec: cloudcontrolv1beta1.VpcPeeringSpec{
				Details: &cloudcontrolv1beta1.VpcPeeringDetails{
					LocalNetwork: klog.ObjectRef{
						Name:      kymaNetwork.Name,
						Namespace: kymaNetwork.Namespace,
					},
					RemoteNetwork: klog.ObjectRef{
						Name:      remoteNetwork.Name,
						Namespace: remoteNetwork.Namespace,
					},
					PeeringName:      remotePeeringName,
					LocalPeeringName: "cm-" + remoteNetworkName,
				},
			},
		}

		By("And Given KCP VpcPeering is created", func() {
			Eventually(CreateObj).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					WithName(remoteNetworkName),
					WithRemoteRef(remoteRefName),
					WithScope(kymaName),
				).
				Should(Succeed())
		})

		By("And Given GCP VpcPeering is created on remote side", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingVpcPeeringStatusRemoteId(),
				).
				Should(Succeed())
		})

		By("And Given GCP VpcPeering is created on kyma side", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingVpcPeeringStatusId(),
				).
				Should(Succeed())
		})

		By("And Given KCP VpcPeering has Ready condition", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering,
					NewObjActions(),
					HavingConditionTrue(cloudcontrolv1beta1.ConditionTypeReady),
				).
				Should(Succeed())
		})

		By("And Given KCP KymaNetwork has Warning state", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					kymaNetwork,
					WithoutConditions("Ready"),
					WithState("Warning"),
					WithConditions(KcpWarningCondition())).
				Should(Succeed())
		})

		By("And Given KCP RemoteNetwork has Warning state", func() {
			Eventually(UpdateStatus).
				WithArguments(infra.Ctx(),
					infra.KCP().Client(),
					remoteNetwork,
					WithoutConditions("Ready"),
					WithState("Warning"),
					WithConditions(KcpWarningCondition())).
				Should(Succeed())
		})

		By("When KCP VpcPeering is deleted", func() {
			Eventually(Delete).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "Error deleting VPC Peering")
		})

		By("Then VpcPeering does not exist", func() {
			Eventually(IsDeleted).
				WithArguments(infra.Ctx(), infra.KCP().Client(), vpcpeering).
				Should(Succeed(), "VPC Peering was not deleted")
		})
	})
})
