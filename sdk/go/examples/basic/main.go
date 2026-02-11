package main

import (
	"fmt"
	"log"

	evesde "github.com/ilya/eve-sde-server/sdk/go"
)

func main() {
	// Create client (with optional API key)
	client := evesde.NewClient("http://localhost:8080", "")

	// You can also use an API key if authentication is enabled
	// client := evesde.NewClient("http://localhost:8080", "esk_your_api_key_here")

	// Check server health
	fmt.Println("=== Health Check ===")
	healthy, err := client.Health()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server healthy: %v\n\n", healthy)

	// Get a specific item (Tritanium)
	fmt.Println("=== Get Item (Tritanium) ===")
	item, err := client.GetItem(34)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID: %d\n", item.TypeID)
	fmt.Printf("Name: %s\n", item.Name)
	fmt.Printf("Volume: %.2f m³\n", item.Volume)
	fmt.Printf("Description: %.100s...\n\n", item.Description)

	// Search for items
	fmt.Println("=== Search for 'shield' ===")
	results, err := client.Search("shield", 5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d items (showing %d):\n", results.Meta.Total, len(results.Data))
	for i, item := range results.Data {
		fmt.Printf("%d. %s (ID: %d)\n", i+1, item.Name, item.TypeID)
	}
	fmt.Println()

	// List items with pagination
	fmt.Println("=== List First 10 Items ===")
	items, err := client.ListItems(10, 0)
	if err != nil {
		log.Fatal(err)
	}
	for i, item := range items {
		fmt.Printf("%d. %s (ID: %d, Volume: %.2f m³)\n", i+1, item.Name, item.TypeID, item.Volume)
	}
}
