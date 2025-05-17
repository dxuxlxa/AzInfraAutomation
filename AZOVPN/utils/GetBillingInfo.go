package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"
)

func GetBillingInfo() {
	InfoLogger.Println("Fetching billing information...")

	ctx := context.Background()
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		ErrorLogger.Printf("Failed to get credentials: %v", err)
		return
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")

	// Create cost management client
	costClient, err := armcostmanagement.NewQueryClient(cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create cost management client: %v", err)
		return
	}

	// Get current time and 30 days ago
	now := time.Now().UTC()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Format dates in YYYY-MM-DD format
	// timePeriod := &armcostmanagement.QueryTimePeriod{
	// 	From: &thirtyDaysAgo,
	// 	To:   &now,
	// }

	// Create query for daily costs
	scope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, resourceGroupName)

	var exportType armcostmanagement.ExportType = armcostmanagement.ExportTypeActualCost
	var granularityType armcostmanagement.GranularityType = armcostmanagement.GranularityTypeDaily
	var functionType armcostmanagement.FunctionType = armcostmanagement.FunctionTypeSum

	query := armcostmanagement.QueryDefinition{
		Type: &exportType,
		// TimePeriod: timePeriod, Time period must not be present if Time frame is not Custom
		Dataset: &armcostmanagement.QueryDataset{
			Granularity: &granularityType,
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"preTaxCost": {
					Function: &functionType,
				},
				"totalCost": {
					Function: &functionType,
				},
			},
		},
	}

	InfoLogger.Printf("Querying costs for resource group %s for the last 30 days...", resourceGroupName)
	result, err := costClient.Usage(ctx, scope, query, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to get cost data: %v", err)
		return
	}

	// Print the billing information
	fmt.Printf("\nBilling Information for Resource Group: %s\n", resourceGroupName)
	fmt.Println("============================================")
	fmt.Printf("Time Period: %s to %s\n\n", thirtyDaysAgo.Format("2006-01-02"), now.Format("2006-01-02"))

	if result.Properties != nil && result.Properties.Rows != nil {
		fmt.Println("Cost Breakdown by Resource Type:")
		fmt.Println("--------------------------------------------")

		for _, row := range result.Properties.Rows {
			if len(row) >= 3 {
				resourceType := row[0].(string)
				cost := row[1].(float64)
				currency := row[2].(string)
				fmt.Printf("%-30s: %.2f %s\n", resourceType, cost, currency)
			}
		}
	} else {
		fmt.Println("No cost data available for the specified time period.")
	}

	fmt.Println("\nNote: Costs shown are for the last 30 days and may not include very recent charges.")
}

// func toPtr(s string) *string {
// 	return &s
// }
