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

func CreateNIC(
	ctx context.Context,
	subscriptionID string,
	cred *azidentity.DefaultAzureCredential,
	subnetID *string,
	publicIPID *string,
	nsgID *string,
	location string,
	resourceGroupName string,
) (
	nicResult *runtime.Poller[armnetwork.InterfacesClientCreateOrUpdateResponse],
	err error) {

	nicName := os.Getenv("NIC_NAME")
	InfoLogger.Printf("Creating network interface %s in %s", nicName, location)
	InfoLogger.Printf("Using subnet ID: %s", *subnetID)
	InfoLogger.Printf("Using public IP ID: %s", *publicIPID)
	InfoLogger.Printf("Using NSG ID: %s", *nsgID)

	nicParams := armnetwork.Interface{
		Location: to.Ptr(location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr(nicName),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(*subnetID),
						},
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: to.Ptr(*publicIPID),
						},
					},
				},
			},
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: to.Ptr(*nsgID),
			},
		},
	}

	// Create or update NIC
	nicClient, err := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create network interface client: %v", err)
		return nil, fmt.Errorf("failed to create network interfaces client: %v", err)
	}

	InfoLogger.Printf("Beginning network interface creation...")
	nicPoller, err := nicClient.BeginCreateOrUpdate(ctx, resourceGroupName, nicName, nicParams, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to begin NIC creation: %v", err)
		return nil, fmt.Errorf("failed to begin NIC creation: %v", err)
	}

	InfoLogger.Printf("Network interface creation initiated successfully")
	return nicPoller, nil
}
