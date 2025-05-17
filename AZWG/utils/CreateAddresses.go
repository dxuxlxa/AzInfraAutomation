package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

// CreateAddresses creates both the subnet and public IP address
// Returns pollers for both operations to allow the caller to control polling frequency
func CreateAddresses(
	ctx context.Context,
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resourceGroupName string,
	location string,
	vnetName string,
) (
	*runtime.Poller[armnetwork.SubnetsClientCreateOrUpdateResponse],
	*runtime.Poller[armnetwork.PublicIPAddressesClientCreateOrUpdateResponse],
	error,
) {
	// Create Subnet
	subnetName := os.Getenv("SUBNET_NAME")
	subnetPrefix := os.Getenv("SUBNET_PREFIX")
	InfoLogger.Printf("Creating subnet %s with prefix %s", subnetName, subnetPrefix)

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create subnet client: %v", err)
		return nil, nil, fmt.Errorf("failed to create subnet client: %v", err)
	}

	InfoLogger.Printf("Initiating subnet creation in VNet %s...", vnetName)
	subnetPoller, err := subnetClient.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, subnetName, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.Ptr(subnetPrefix),
		},
	}, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to begin subnet creation: %v", err)
		return nil, nil, fmt.Errorf("failed to begin subnet creation: %v", err)
	}

	// Create Public IP Address
	publicIPName := os.Getenv("PUBLIC_IP_NAME")
	InfoLogger.Printf("Creating public IP address %s in %s", publicIPName, location)

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create public IP client: %v", err)
		return nil, nil, fmt.Errorf("failed to create public IP client: %v", err)
	}

	InfoLogger.Printf("Initiating public IP creation with static allocation...")
	publicIPPoller, err := publicIPClient.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPName, armnetwork.PublicIPAddress{
		Location: &location,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: to.Ptr(armnetwork.PublicIPAddressSKUNameStandard),
		},
	}, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to begin public IP creation: %v", err)
		return nil, nil, fmt.Errorf("failed to begin public IP creation: %v", err)
	}

	InfoLogger.Printf("Network address creation initiated successfully")
	return subnetPoller, publicIPPoller, nil
}
