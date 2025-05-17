package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

// CreateMultiplePortRules creates security rules for multiple ports in a Network Security Group
func CreateMultiplePortRules(ctx context.Context, cred *azidentity.DefaultAzureCredential) error {
	// Retrieve environment variables
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	nsgName := os.Getenv("NSG_NAME")

	// Create SecurityRulesClient
	securityRulesClient, err := armnetwork.NewSecurityRulesClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create SecurityRules client: %v", err)
	}

	// Define ports to create rules for
	ports := []int{80, 443, 51820, 22}
	fmt.Printf("Processing rules for standard ports: %v\n", ports)

	// Iterate over ports to create rules
	for i, port := range ports {
		fmt.Printf("\nCreating rule for port %d...\n", port)
		ruleName := fmt.Sprintf("Allow-Port-%d", port)
		priority := int32(100 + (i * 50)) // 100, 150, 200, 250, etc.

		// Define the security rule
		securityRule := armnetwork.SecurityRule{
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				Description:              to.Ptr(fmt.Sprintf("Allow inbound traffic on port %d", port)),
				Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolTCP),
				SourceAddressPrefix:      to.Ptr("*"),
				SourcePortRange:          to.Ptr("*"),
				DestinationAddressPrefix: to.Ptr("*"),
				DestinationPortRange:     to.Ptr(fmt.Sprintf("%d", port)),
				Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
				Priority:                 to.Ptr(priority),
				Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
			},
		}

		// Create or update the security rule
		poller, err := securityRulesClient.BeginCreateOrUpdate(
			ctx,
			resourceGroupName,
			nsgName,
			ruleName,
			securityRule,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to begin creating rule for port %d: %v", port, err)
		}

		// Wait for the operation to complete
		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete rule creation for port %d: %v", port, err)
		}

		fmt.Printf("Successfully created rule for port %d\n", port)
	}

	fmt.Printf("All rules created successfully for NSG %s\n", nsgName)
	return nil
}
