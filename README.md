# go-flakeid

A fast, minimal, distributed ID generator for Go, inspired by Twitter Snowflake. Generate sortable 64-bit unique IDs with zero dependencies and lock-free performance.

## Features

- **Lock-free ID generation** using atomic operations
- **1M+ IDs per second** performance
- **64-bit unique IDs** sortable by generation time
- **Zero external dependencies**
- **Customizable bit layouts** for different scales
- **Auto machine ID** derivation from hostname/IP
- **Thread-safe** with no mutex locks
- **Configurable epoch** for extended timestamp range

## Installation

```bash
go get github.com/capy-engineer/go-flakeid
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/capy-engineer/go-flakeid"
)

func main() {
    // Create configuration: regionID=1, appID=2, machineID=5
    config := goflakeid.NewConfig(1, 2, 5)
    
    // Create generator
    generator, err := goflakeid.NewGenerator(*config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate ID
    id, err := generator.Generate()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated ID: %d\n", id)
    
    // Decode ID components
    components := generator.Decode(id)
    fmt.Printf("Timestamp: %s\n", components.Timestamp)
    fmt.Printf("Region: %d, App: %d, Machine: %d, Sequence: %d\n",
        components.RegionID, components.AppID, components.MachineID, components.Sequence)
}
```

## Configuration

### Default Bit Layout

The default layout allocates 64 bits as: `42 + 4 + 3 + 5 + 10 = 64`

- **42 bits** for timestamp (≈139 years from epoch)
- **4 bits** for region ID (16 regions)
- **3 bits** for app ID (8 apps per region)
- **5 bits** for machine ID (32 machines per app)
- **10 bits** for sequence (1024 IDs per millisecond)

This supports:
- 16 regions × 8 apps × 32 machines = 4,096 generators
- 1,024 IDs per millisecond per generator
- 4.2 billion IDs per second globally

### Custom Configuration

```go
// Custom epoch (recommended: your service launch date)
config := goflakeid.NewConfig(1, 2, 5).
    WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

// Auto-derive machine ID from environment
config = goflakeid.NewConfig(1, 2, 0).
    WithAutoMachineID()

// Custom bit layout for different scale
customLayout := goflakeid.BitLayout{
    TimestampBits: 41,  // ≈69 years
    RegionBits:    5,   // 32 regions
    AppBits:       5,   // 32 apps
    MachineBits:   8,   // 256 machines
    SequenceBits:  5,   // 32 IDs/ms (lower throughput, more machines)
}

config = goflakeid.NewConfig(1, 2, 5).
    WithBitLayout(customLayout)
```

## Advanced Usage

### Batch Generation

```go
// Generate 100 IDs efficiently
ids, err := generator.GenerateBatch(100)
if err != nil {
    log.Fatal(err)
}
```

### Machine ID Strategies

```go
// Auto-derive from hostname/MAC address
config := goflakeid.NewConfig(1, 2, 0).WithAutoMachineID()

// Derive from IP address
machineID := goflakeid.MachineIDFromIP("192.168.1.100")
config := goflakeid.NewConfig(1, 2, machineID)

// Custom function
config.MachineIDGen = func() uint8 {
    // Your custom logic here
    return uint8(os.Getpid() & 0x1F)
}
```

## Architecture

### ID Structure (64 bits)

```
|-------- 42 bits --------|-- 4 --|-- 3 --|-- 5 --|---- 10 ----|
| timestamp (milliseconds) | region |  app  | machine | sequence |
```

### Lock-Free Algorithm

The generator uses atomic Compare-And-Swap (CAS) operations to ensure thread safety without mutex locks:

1. Pack timestamp and sequence into a single 64-bit state
2. Use atomic CAS to update state
3. Retry on contention (extremely rare)
4. No mutex locks = better performance under load

## Best Practices

1. **Set epoch close to your service launch date** to maximize timestamp range
2. **Use meaningful region/app IDs** for easier debugging
3. **Monitor sequence numbers** in high-throughput scenarios
4. **Use batch generation** for bulk operations
5. **Configure bit layout** based on your scale requirements

## Common Configurations

### Small Scale (Single Region)
```go
// 41 timestamp + 10 machine + 13 sequence = 64 bits
// Supports 1024 machines, 8192 IDs/ms each
layout := goflakeid.BitLayout{
    TimestampBits: 41,
    RegionBits:    0,
    AppBits:       0,
    MachineBits:   10,
    SequenceBits:  13,
}
```

### Multi-Region Scale
```go
// Default layout: 42 + 4 + 3 + 5 + 10 = 64
// 16 regions, 8 apps, 32 machines, 1024 IDs/ms
config := goflakeid.NewConfig(regionID, appID, machineID)
```

### Extreme Scale
```go
// 40 timestamp + 8 region + 8 app + 8 sequence = 64 bits
// 256 regions/apps, only 256 IDs/ms per machine
layout := goflakeid.BitLayout{
    TimestampBits: 40,  // ≈34 years
    RegionBits:    8,   // 256 regions
    AppBits:       8,   // 256 apps
    MachineBits:   0,   // Encoded in app ID
    SequenceBits:  8,   // 256 IDs/ms
}
```

## Contributing

Contributions are welcome! Please ensure:
- Zero external dependencies
- Maintain lock-free performance
- Add tests for new features
- Update documentation