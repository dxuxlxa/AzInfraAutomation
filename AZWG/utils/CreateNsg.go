package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

func CreateNsg(
	ctx context.Context,
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resourceGroupName string,
	location string,
) (*runtime.Poller[armnetwork.SecurityGroupsClientCreateOrUpdateResponse], error) {
	nsgName := os.Getenv("NSG_NAME")
	InfoLogger.Printf("Creating Network Security Group %s in %s", nsgName, location)

	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create NSG client: %v", err)
		return nil, fmt.Errorf("failed to create NSG client: %v", err)
	}

	InfoLogger.Printf("Initiating NSG creation...")
	nsgPoller, err := nsgClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		nsgName,
		armnetwork.SecurityGroup{
			Location: &location,
		},
		nil,
	)
	if err != nil {
		ErrorLogger.Printf("Failed to begin NSG creation: %v", err)
		return nil, fmt.Errorf("failed to begin NSG creation: %v", err)
	}

	InfoLogger.Printf("NSG creation initiated successfully")
	return nsgPoller, nil
}
