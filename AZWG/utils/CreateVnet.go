package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

func CreateVnet(
	ctx context.Context,
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resourceGroupName string,
	location string,
	vnetName string,
	addressPrefix string,
) (armnetwork.VirtualNetworksClientCreateOrUpdateResponse, error) {
	InfoLogger.Printf("Creating virtual network %s in %s", vnetName, location)
	InfoLogger.Printf("Using address prefix: %s", addressPrefix)

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create virtual network client: %v", err)
		return armnetwork.VirtualNetworksClientCreateOrUpdateResponse{}, err
	}

	InfoLogger.Printf("Initiating virtual network creation")
	vnetPoller, err := vnetClient.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: &location,
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{&addressPrefix},
			},
		},
	}, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to begin virtual network creation: %v", err)
		return armnetwork.VirtualNetworksClientCreateOrUpdateResponse{}, fmt.Errorf("failed to allocate virtual network: %v", err)
	}

	InfoLogger.Printf("Waiting for virtual network creation to complete...")
	vnetResult, err := vnetPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 6 * time.Second,
	})
	if err != nil {
		ErrorLogger.Printf("Failed to complete virtual network creation: %v", err)
		return armnetwork.VirtualNetworksClientCreateOrUpdateResponse{}, err
	}

	InfoLogger.Printf("Virtual network %s created successfully", vnetName)
	return vnetResult, nil
}
