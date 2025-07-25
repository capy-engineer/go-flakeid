# GoFlakeID - Distributed ID Generator Library

A high-performance, decentralized ID generator library for Go applications. Generate unique, sortable IDs across distributed systems without central coordination.

## Features

- 🚀 **High Performance**: Generate 1000+ IDs per millisecond per machine
- 🔄 **Distributed**: No central coordination required
- 📈 **Sortable**: Lexicographically sortable by generation time
- 🛡️ **Thread-Safe**: Safe for concurrent use across multiple goroutines
- 🎯 **Flexible**: Configurable entity types, encoders, and bit layouts
- 🔧 **Pluggable**: Custom encoders, time providers, and configurations
- 📦 **Zero Dependencies**: Pure Go implementation

## Installation

```bash
go get github.com/hxuan190/goflakeid
```

## Quick Start

### 1. Define Your Entity Types

```go
package main

import "github.com/hxuan190/goflakeid"

// Define entity types for your domain
const (
    EntityUser     goflakeid.EntityType = 0
    EntityProduct  goflakeid.EntityType = 1
    EntityOrder    goflakeid.EntityType = 2
    EntityPayment  goflakeid.EntityType = 3
)
```

### 2. Configure and Create Generator

```go
func main() {
    // Configure generator with your entity types
    config := goflakeid.NewConfig(1, 5). // datacenter=1, machine=5
        AddEntityType(EntityUser, "user", "u").
        AddEntityType(EntityProduct, "product", "p").
        AddEntityType(EntityOrder, "order", "o").
        AddEntityType(EntityPayment, "payment", "pay")
    
    generator, err := goflakeid.NewGenerator(*config)
    if err != nil {
        panic(err)
    }
    
    // Generate IDs
    userID, _ := generator.GeneratePublic(EntityUser)       // "u_9q81j1zBf"
    productID, _ := generator.GeneratePublic(EntityProduct) // "p_8k72m3nXe"
    
    fmt.Printf("User ID: %s\n", userID)
    fmt.Printf("Product ID: %s\n", productID)
}
```

## ID Structure

The library generates 64-bit IDs with the following structure:

| Component | Bits | Description |
|-----------|------|-------------|
| Timestamp | 41   | Millisecond precision (69+ years) |
| Entity Type | 5  | Up to 32 entity types |
| Datacenter | 2   | Up to 4 datacenters |
| Machine | 6      | Up to 64 machines per datacenter |
| Sequence | 10     | Up to 1024 IDs/ms per machine |

## Usage Examples

### Basic Usage

```go
// Create generator
config := goflakeid.NewConfig(1, 5).
    AddEntityType(EntityUser, "user", "u").
    AddEntityType(EntityPost, "post", "p")

generator, _ := goflakeid.NewGenerator(*config)

// Generate raw ID
rawID, _ := generator.Generate(EntityUser)
fmt.Printf("Raw ID: %d\n", rawID) // 1234567890123456

// Generate public ID
publicID, _ := generator.GeneratePublic(EntityUser)
fmt.Printf("Public ID: %s\n", publicID) // "u_9q81j1zBf"
```

### Parsing IDs

```go
// Parse public ID back to components
rawID, entityType, _ := generator.DecodePublic("u_9q81j1zBf")

// Extract all components
parsed := generator.Parse(rawID)
fmt.Printf("Generated at: %s\n", parsed.Timestamp)
fmt.Printf("Entity type: %d\n", parsed.EntityType)
fmt.Printf("Datacenter: %d\n", parsed.DatacenterID)
fmt.Printf("Machine: %d\n", parsed.MachineID)
fmt.Printf("Sequence: %d\n", parsed.Sequence)
```

### Music Platform Example

```go
// Define music platform entities
const (
    EntityTrack    goflakeid.EntityType = 0
    EntityArtist   goflakeid.EntityType = 1
    EntityAlbum    goflakeid.EntityType = 2
    EntityPlaylist goflakeid.EntityType = 3
    EntityUser     goflakeid.EntityType = 4
    EntityConcert  goflakeid.EntityType = 5
)

func NewMusicgoflakeiderator(datacenter, machine uint8) *goflakeid.goflakeiderator {
    config := goflakeid.NewConfig(datacenter, machine).
        AddEntityType(EntityTrack, "track", "t", "Music track").
        AddEntityType(EntityArtist, "artist", "a", "Recording artist").
        AddEntityType(EntityAlbum, "album", "al", "Music album").
        AddEntityType(EntityPlaylist, "playlist", "p", "User playlist").
        AddEntityType(EntityUser, "user", "u", "Platform user").
        AddEntityType(EntityConcert, "concert", "c", "Live concert")
    
    generator, _ := goflakeid.NewGenerator(*config)
    return generator
}

// Usage
musicGen := NewMusicgoflakeiderator(1, 5)
trackID, _ := musicGen.GeneratePublic(EntityTrack)    // "t_9q81j1zBf"
artistID, _ := musicGen.GeneratePublic(EntityArtist)  // "a_8k72m3nXe"
```

### E-commerce Platform Example

```go
const (
    EntityProduct   goflakeid.EntityType = 0
    EntityCustomer  goflakeid.EntityType = 1
    EntityOrder     goflakeid.EntityType = 2
    EntityPayment   goflakeid.EntityType = 3
    EntityShipment  goflakeid.EntityType = 4
    EntityReview    goflakeid.EntityType = 5
)

config := goflakeid.NewConfig(2, 10).
    AddEntityType(EntityProduct, "product", "prd").
    AddEntityType(EntityCustomer, "customer", "cst").
    AddEntityType(EntityOrder, "order", "ord").
    AddEntityType(EntityPayment, "payment", "pay").
    AddEntityType(EntityShipment, "shipment", "shp").
    AddEntityType(EntityReview, "review", "rev")

ecommerceGen, _ := goflakeid.NewGenerator(*config)

// Generate IDs
productID, _ := ecommerceGen.GeneratePublic(EntityProduct)   // "prd_9q81j1zBf"
orderID, _ := ecommerceGen.GeneratePublic(EntityOrder)       // "ord_8k72m3nXe"
```

## Advanced Configuration

### Custom Epoch

```go
customEpoch := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

config := goflakeid.NewConfig(1, 5).
    SetEpoch(customEpoch).
    AddEntityType(EntityUser, "user", "u")
```

### Custom Bit Layout

```go
// More entity types, less timestamp precision
customLayout := goflakeid.BitLayout{
    TimestampBits:  38,  // ~8.7 years from epoch
    EntityTypeBits: 8,   // 256 entity types
    DatacenterBits: 2,   // 4 datacenters
    MachineBits:    6,   // 64 machines
    SequenceBits:   10,  // 1024 IDs/ms
}

config := goflakeid.NewConfig(1, 5).
    SetBitLayout(customLayout).
    AddEntityType(EntityUser, "user", "u")
```

### Custom Encoder

```go
// Use hexadecimal encoding instead of Base62
config := goflakeid.NewConfig(1, 5).
    SetEncoder(goflakeid.HexEncoder{}).
    AddEntityType(EntityUser, "user", "u")

generator, _ := goflakeid.NewGenerator(*config)
userID, _ := generator.GeneratePublic(EntityUser) // "u_a1b2c3d4e5f6"
```

### Custom Base62 Alphabet

```go
// Custom alphabet (e.g., URL-safe)
customAlphabet := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
encoder := goflakeid.NewBase62Encoder(customAlphabet)

config := goflakeid.NewConfig(1, 5).
    SetEncoder(encoder).
    AddEntityType(EntityUser, "user", "u")
```

## Batch Generation

For high-throughput scenarios:

```go
// Generate 100 IDs in a batch
userIDs, err := generator.GenerateBatch(EntityUser, 100)
if err != nil {
    panic(err)
}

fmt.Printf("Generated %d user IDs\n", len(userIDs))
```


## Distributed Deployment

### Multiple Datacenters

```go
// Datacenter 1
dc1Generator, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 5).AddEntityType(...))

// Datacenter 2  
dc2Generator, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(2, 5).AddEntityType(...))

// IDs from different datacenters are guaranteed unique
```

### Multiple Machines per Datacenter

```go
// Machine 1 in datacenter 1
machine1, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 1).AddEntityType(...))

// Machine 2 in datacenter 1
machine2, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 2).AddEntityType(...))

// Each machine generates unique IDs
```

## Best Practices

### 1. Entity Type Management

```go
// Keep entity types in a separate package
package entities

import "github.com/hxuan190/goflakeid"

const (
    User     goflakeid.EntityType = 0
    Product  goflakeid.EntityType = 1
    Order    goflakeid.EntityType = 2
    // ... add new types sequentially
)

// Provide factory function
func Newgoflakeiderator(datacenter, machine uint8) *goflakeid.goflakeiderator {
    config := goflakeid.NewConfig(datacenter, machine).
        AddEntityType(User, "user", "u").
        AddEntityType(Product, "product", "p").
        AddEntityType(Order, "order", "o")
    
    generator, _ := goflakeid.NewGenerator(*config)
    return generator
}
```

### 2. Singleton Pattern

```go
var (
    generator *goflakeid.goflakeiderator
    once      sync.Once
)

func Getgoflakeiderator() *goflakeid.goflakeiderator {
    once.Do(func() {
        // Initialize from config
        datacenterID := getDatacenterID() // from env/config
        machineID := getMachineID()       // from env/config
        
        generator = Newgoflakeiderator(datacenterID, machineID)
    })
    return generator
}
```

### 3. Error Handling

```go
func CreateUser(name string) (string, error) {
    userID, err := generator.GeneratePublic(EntityUser)
    if err != nil {
        return "", fmt.Errorf("failed to generate user ID: %w", err)
    }
    
    // Use userID...
    return userID, nil
}
```

### 4. Configuration Management

```go
// config.yaml
id_generator:
  datacenter_id: 1
  machine_id: 5
  epoch: "2023-01-01T00:00:00Z"
  encoder: "base62"

// Load from config
type Config struct {
    goflakeiderator struct {
        DatacenterID uint8     `yaml:"datacenter_id"`
        MachineID    uint8     `yaml:"machine_id"`
        Epoch        time.Time `yaml:"epoch"`
        Encoder      string    `yaml:"encoder"`
    } `yaml:"id_generator"`
}
```

## Troubleshooting

### Clock Issues

```go
// Handle clock backwards gracefully
id, err := generator.Generate(EntityUser)
if err != nil && strings.Contains(err.Error(), "clock moved backwards") {
    // Log warning and retry
    log.Warn("Clock moved backwards, retrying...")
    time.Sleep(time.Millisecond)
    id, err = generator.Generate(EntityUser)
}
```

### High Load Scenarios

```go
// Use batch generation for bulk operations
const batchSize = 100
userIDs, err := generator.GenerateBatch(EntityUser, batchSize)
if err != nil {
    // Handle error
}
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- Inspired by Twitter's Snowflake ID generation algorithm
- Designed for modern distributed systems and microservices
- Built with performance and scalability in mind