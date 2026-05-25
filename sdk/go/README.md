# EVE SDE Server - Go SDK

Go client library for the EVE SDE Server API.

## Installation

```bash
go get github.com/ilyaux/eve-sde-server/sdk/go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    evesde "github.com/ilyaux/eve-sde-server/sdk/go"
)

func main() {
    client := evesde.NewClient("http://localhost:8080", "")

    item, err := client.GetItem(34)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found: %s\n", item.Name)
}
```

## Usage

```go
client := evesde.NewClient("http://localhost:8080", "")

// Use an API key when AUTH_ENABLED=true.
client = evesde.NewClient("http://localhost:8080", "esk_your_api_key")
```

## Items

```go
item, err := client.GetItem(34)
items, err := client.ListItems(100, 0)
list, err := client.ListItemsWithMeta(100, 0)
results, err := client.Search("tritanium", 10)
```

`ListItems` returns only the item slice. `ListItemsWithMeta` returns pagination
metadata as well.

## Taxonomy

```go
category, err := client.GetCategory(4)
categories, err := client.ListCategories(50, 0)

group, err := client.GetGroup(18)
groups, err := client.ListGroups(6, 50, 0) // categoryID, limit, offset
allGroups, err := client.ListGroups(0, 50, 0)
```

Pass `categoryID <= 0` to `ListGroups` to list groups across all categories.

## SDE Versions

```go
changelog, err := client.Changelog()
diff, err := client.Diff("20250101", "20250201")
```

`Diff` uses version identifiers tracked by the server during imports. When no
historical snapshots are available, the server returns an empty diff with a
note.

## ESI Proxy

```go
typeInfo, err := client.ESITypeInfo(34)
prices, err := client.ESIMarketPrices()
history, err := client.ESIMarketHistory(10000002, 34)
```

These helpers call the server-side cached ESI proxy rather than ESI directly.

## Health

```go
healthy, err := client.Health()
if err != nil {
    log.Fatal(err)
}

health, err := client.HealthStatus()
ready, err := client.Ready()
version, err := client.Version()
```

## Types

```go
type Item struct {
    TypeID      int
    Name        string
    Description string
    Volume      float64
    GroupID     int
    CategoryID  int
}

type Category struct {
    CategoryID int
    Name       string
    Published  bool
}

type Group struct {
    GroupID    int
    CategoryID int
    Name       string
    Published  bool
}
```

## Timeout

The default HTTP timeout is 30 seconds:

```go
client := evesde.NewClient("http://localhost:8080", "")
client.HTTPClient.Timeout = 60 * time.Second
```

## Examples

See [examples/basic](./examples/basic/) for a complete runnable example.
