# EVE SDE Server - Go SDK

Official Go client library for the EVE SDE Server API.

## Installation

```bash
go get github.com/ilya/eve-sde-server/sdk/go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    evesde "github.com/ilya/eve-sde-server/sdk/go"
)

func main() {
    // Create client
    client := evesde.NewClient("http://localhost:8080", "")

    // Get item
    item, err := client.GetItem(34) // Tritanium
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found: %s\n", item.Name)
}
```

## Usage

### Initialize Client

```go
// Without authentication
client := evesde.NewClient("http://localhost:8080", "")

// With API key
client := evesde.NewClient("http://localhost:8080", "esk_your_api_key_here")
```

### Get Item by ID

```go
item, err := client.GetItem(34)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Name: %s\n", item.Name)
fmt.Printf("Volume: %.2f m³\n", item.Volume)
fmt.Printf("Description: %s\n", item.Description)
```

### Search Items

```go
results, err := client.Search("tritanium", 10)
if err != nil {
    log.Fatal(err)
}

for _, item := range results.Data {
    fmt.Printf("- %s (ID: %d)\n", item.Name, item.TypeID)
}
```

### List Items with Pagination

```go
// Get first 100 items
items, err := client.ListItems(100, 0)
if err != nil {
    log.Fatal(err)
}

// Get next 100 items
items, err = client.ListItems(100, 100)
```

### Health Check

```go
healthy, err := client.Health()
if err != nil {
    log.Fatal(err)
}

if healthy {
    fmt.Println("Server is healthy")
}
```

## API Reference

### Types

#### `Item`
```go
type Item struct {
    TypeID      int     `json:"type_id"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Volume      float64 `json:"volume"`
    GroupID     int     `json:"group_id,omitempty"`
    CategoryID  int     `json:"category_id,omitempty"`
}
```

#### `SearchResult`
```go
type SearchResult struct {
    Data []Item
    Meta struct {
        Total  int
        Limit  int
        Offset int
    }
}
```

### Methods

#### `NewClient(baseURL, apiKey string) *Client`
Creates a new API client.

#### `GetItem(typeID int) (*Item, error)`
Retrieves an item by its type ID.

#### `ListItems(limit, offset int) ([]Item, error)`
Lists items with pagination.

#### `Search(query string, limit int) (*SearchResult, error)`
Searches for items by name or description.

#### `Health() (bool, error)`
Checks server health.

## Examples

See the [examples](./examples/) directory for complete examples:

- `basic/` - Basic usage examples
- `advanced/` - Advanced patterns (caching, retries, etc.)

## Error Handling

All methods return errors that should be checked:

```go
item, err := client.GetItem(34)
if err != nil {
    log.Printf("Error fetching item: %v", err)
    return
}
```

## Timeouts

Default timeout is 30 seconds. You can customize it:

```go
client := evesde.NewClient("http://localhost:8080", "")
client.HTTPClient.Timeout = 60 * time.Second
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License
