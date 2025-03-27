package main

import (
	"context"
	"fmt"
	"log"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	nc "cloud.google.com/go/networkconnectivity/apiv1"
	"cloud.google.com/go/networkconnectivity/apiv1/networkconnectivitypb"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"k8s.io/utils/ptr"
)

func main() {
	ctx := context.Background()

	isDeleteFlow := true

	idempotenceId := uuid.NewString()

	projectId := "sap-sc-learn"
	region := "us-central1"
	parent := fmt.Sprintf("projects/%s/locations/%s", projectId, region)
	networkName := "dule-vpc"
	networkNameFull := fmt.Sprintf("projects/%s/global/networks/%s", projectId, networkName)
	subnetName := "duletest234234"
	subnetNameFull := fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", projectId, region, subnetName)
	subnetRange := "10.250.8.0/29"

	// format: projects/project-id/locations/us-central1/serviceConnectionPolicies/policy-1
	connectionPolicyNameShort := "dulerediscluster"
	connectionPolicyNameFull := fmt.Sprintf("%s/serviceConnectionPolicies/%s", parent, connectionPolicyNameShort)

	computeClient, err := compute.NewSubnetworksRESTClient(ctx, option.WithCredentialsFile("/Users/i517577/Documents/gcpcred/sap-sc-learn-redis-sa.json"))
	if err != nil {
		log.Fatalf("Failed to create network client: %v", err)
	}
	defer computeClient.Close()

	ncClient, err := nc.NewCrossNetworkAutomationClient(ctx, option.WithCredentialsFile("/Users/i517577/Documents/gcpcred/sap-sc-learn-redis-sa.json"))
	if err != nil {
		log.Fatalf("Failed to create network client: %v", err)
	}
	defer ncClient.Close()

	connectionPolicy, err := ncClient.GetServiceConnectionPolicy(ctx, &networkconnectivitypb.GetServiceConnectionPolicyRequest{
		Name: connectionPolicyNameFull,
	})
	if err != nil {
		log.Printf("Failed to get connection policy: %v", err)
	}

	subnet, err := computeClient.Get(ctx, &computepb.GetSubnetworkRequest{
		Project:    projectId,
		Region:     region,
		Subnetwork: subnetName,
	})
	if err != nil {
		log.Printf("Failed to get network: %v", err)
	}

	if isDeleteFlow {
		if connectionPolicy != nil {
			log.Printf("Deleting connection policy")
			_, err = ncClient.DeleteServiceConnectionPolicy(ctx, &networkconnectivitypb.DeleteServiceConnectionPolicyRequest{
				Name:      connectionPolicyNameFull,
				RequestId: idempotenceId,
			})
			if err != nil {
				log.Fatalf("Failed to connection policy: %v", err)
			}
		}

		if subnet != nil {
			log.Printf("Deleting subnet...")
			_, err = computeClient.Delete(ctx, &computepb.DeleteSubnetworkRequest{
				Project:    projectId,
				Region:     region,
				Subnetwork: subnetName,
				RequestId:  &idempotenceId,
			})
			if err != nil {
				log.Fatalf("Failed to delete network: %v", err)
			}
		}

	} else {
		if subnet == nil {
			log.Printf("Creating Subnet...")
			_, err = computeClient.Insert(ctx, &computepb.InsertSubnetworkRequest{
				Project: projectId,
				Region:  region,
				SubnetworkResource: &computepb.Subnetwork{
					IpCidrRange:           &subnetRange,
					Name:                  &subnetName,
					Network:               &networkNameFull,
					PrivateIpGoogleAccess: ptr.To(true),
					Purpose:               ptr.To("PRIVATE"),
				},
				RequestId: &idempotenceId,
			})
			if err != nil {
				log.Fatalf("Failed to create network: %v", err)
			}
		}

		if connectionPolicy == nil {
			log.Printf("Creating connection policy...")
			_, err = ncClient.CreateServiceConnectionPolicy(ctx, &networkconnectivitypb.CreateServiceConnectionPolicyRequest{
				Parent:                    parent,
				ServiceConnectionPolicyId: connectionPolicyNameShort,
				ServiceConnectionPolicy: &networkconnectivitypb.ServiceConnectionPolicy{
					Name:         connectionPolicyNameFull,
					Network:      networkNameFull,
					ServiceClass: "gcp-memorystore-redis",
					Description:  "for redis cluster",
					PscConfig: &networkconnectivitypb.ServiceConnectionPolicy_PscConfig{
						Subnetworks:              []string{subnetNameFull},
						ProducerInstanceLocation: networkconnectivitypb.ServiceConnectionPolicy_PscConfig_PRODUCER_INSTANCE_LOCATION_UNSPECIFIED,
					},
				},
				RequestId: idempotenceId,
			})
			if err != nil {
				log.Fatalf("Failed to create connection policy: %v", err)
			}
		}
	}

	log.Printf("Done")
}
