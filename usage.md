# Complete Usage Guide for go-flakeid

## Table of Contents
1. [Installation](#installation)
2. [Basic Usage](#basic-usage)
3. [Configuration Options](#configuration-options)
4. [Real-World Examples](#real-world-examples)
5. [Integration Patterns](#integration-patterns)
6. [Testing & Debugging](#testing--debugging)
7. [Performance Tuning](#performance-tuning)
8. [Troubleshooting](#troubleshooting)

## Installation

```bash
# Install the package
go get github.com/yourusername/goflakeid

# Import in your code
import "github.com/yourusername/goflakeid"
```

## Basic Usage

### 1. Simple ID Generation

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/goflakeid"
)

func main() {
    // Create generator with region=1, app=2, machine=5
    config := goflakeid.NewConfig(1, 2, 5)
    generator, err := goflakeid.NewGenerator(*config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate a single ID
    id, err := generator.Generate()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated ID: %d\n", id)
    // Output: Generated ID: 7384629384729384729
}
```

### 2. Decoding IDs

```go
// Decode an ID to see its components
components := generator.Decode(id)

fmt.Printf("Timestamp: %s\n", components.Timestamp)
fmt.Printf("Region ID: %d\n", components.RegionID)
fmt.Printf("App ID: %d\n", components.AppID)
fmt.Printf("Machine ID: %d\n", components.MachineID)
fmt.Printf("Sequence: %d\n", components.Sequence)

// Output:
// Timestamp: 2024-03-15 10:30:45.123 +0000 UTC
// Region ID: 1
// App ID: 2
// Machine ID: 5
// Sequence: 0
```

### 3. Batch Generation

```go
// Generate multiple IDs at once
ids, err := generator.GenerateBatch(1000)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d IDs\n", len(ids))
for i, id := range ids[:5] {
    fmt.Printf("ID %d: %d\n", i, id)
}
```

## Configuration Options

### 1. Custom Epoch

Setting a custom epoch extends the lifespan of your ID system:

```go
// Set epoch to your service launch date
config := goflakeid.NewConfig(1, 2, 5).
    WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

generator, _ := goflakeid.NewGenerator(*config)

// IDs will have timestamps relative to 2024-01-01
```

### 2. Auto Machine ID

Let the system automatically determine machine ID:

```go
// Auto-detect machine ID from environment
config := goflakeid.NewConfig(1, 2, 0).
    WithAutoMachineID()

generator, _ := goflakeid.NewGenerator(*config)

// Check what machine ID was assigned
stats := generator.Stats()
fmt.Printf("Auto-assigned machine ID: %d\n", stats.Config.MachineID)
```

### 3. Custom Bit Layout

Adjust bit allocation for your scale requirements:

```go
// High-throughput, single region setup
highThroughputLayout := goflakeid.BitLayout{
    TimestampBits: 41,  // ~69 years
    RegionBits:    0,   // No regions
    AppBits:       0,   // No app separation
    MachineBits:   10,  // 1024 machines
    SequenceBits:  13,  // 8192 IDs per ms
}

config := goflakeid.NewConfig(0, 0, 123).
    WithBitLayout(highThroughputLayout)

// Multi-region, lower throughput setup
multiRegionLayout := goflakeid.BitLayout{
    TimestampBits: 40,  // ~34 years
    RegionBits:    6,   // 64 regions
    AppBits:       6,   // 64 apps
    MachineBits:   6,   // 64 machines
    SequenceBits:  6,   // 64 IDs per ms
}

config2 := goflakeid.NewConfig(15, 10, 20).
    WithBitLayout(multiRegionLayout)
```

## Real-World Examples

### 1. Web Service Integration

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/yourusername/goflakeid"
)

type Server struct {
    idGen *goflakeid.Generator
}

type User struct {
    ID    uint64 `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func NewServer(regionID uint16, appID uint8) (*Server, error) {
    // Derive machine ID from hostname
    config := goflakeid.NewConfig(regionID, appID, 0).
        WithAutoMachineID().
        WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    generator, err := goflakeid.NewGenerator(*config)
    if err != nil {
        return nil, err
    }
    
    return &Server{idGen: generator}, nil
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Generate unique ID for the user
    id, err := s.idGen.Generate()
    if err != nil {
        http.Error(w, "Failed to generate ID", http.StatusInternalServerError)
        return
    }
    
    user.ID = id
    
    // Save user to database...
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func main() {
    server, err := NewServer(1, 1) // Region 1, App 1
    if err != nil {
        log.Fatal(err)
    }
    
    http.HandleFunc("/users", server.CreateUser)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 2. Database Integration

```go
package models

import (
    "database/sql"
    "github.com/yourusername/goflakeid"
)

type UserRepository struct {
    db    *sql.DB
    idGen *goflakeid.Generator
}

func NewUserRepository(db *sql.DB, idGen *goflakeid.Generator) *UserRepository {
    return &UserRepository{db: db, idGen: idGen}
}

func (r *UserRepository) Create(user *User) error {
    // Generate ID
    id, err := r.idGen.Generate()
    if err != nil {
        return fmt.Errorf("failed to generate ID: %w", err)
    }
    
    user.ID = id
    user.CreatedAt = time.Now()
    
    // Insert into database
    query := `INSERT INTO users (id, name, email, created_at) VALUES (?, ?, ?, ?)`
    _, err = r.db.Exec(query, user.ID, user.Name, user.Email, user.CreatedAt)
    
    return err
}

func (r *UserRepository) GetByID(id uint64) (*User, error) {
    var user User
    query := `SELECT id, name, email, created_at FROM users WHERE id = ?`
    
    err := r.db.QueryRow(query, id).Scan(
        &user.ID, &user.Name, &user.Email, &user.CreatedAt,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}

// Decode ID for debugging/monitoring
func (r *UserRepository) GetUserInfo(userID uint64) map[string]interface{} {
    components := r.idGen.Decode(userID)
    
    return map[string]interface{}{
        "user_id":     userID,
        "created_at":  components.Timestamp,
        "region":      components.RegionID,
        "app":         components.AppID,
        "machine":     components.MachineID,
        "sequence":    components.Sequence,
    }
}
```

### 3. Microservices Architecture

```go
// Shared configuration package
package config

import (
    "os"
    "strconv"
    "time"
    "github.com/yourusername/goflakeid"
)

// GetIDGenerator creates a generator for a microservice
func GetIDGenerator(appID uint8) (*goflakeid.Generator, error) {
    // Get region from environment
    regionID := uint16(1) // default
    if r := os.Getenv("REGION_ID"); r != "" {
        if id, err := strconv.Atoi(r); err == nil {
            regionID = uint16(id)
        }
    }
    
    // Get machine ID from environment or auto-generate
    machineID := uint8(0)
    if m := os.Getenv("MACHINE_ID"); m != "" {
        if id, err := strconv.Atoi(m); err == nil {
            machineID = uint8(id)
        }
    }
    
    config := goflakeid.NewConfig(regionID, appID, machineID).
        WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    if machineID == 0 {
        config.WithAutoMachineID()
    }
    
    return goflakeid.NewGenerator(*config)
}

// User Service
package main

import "your-project/config"

func main() {
    const UserServiceAppID = 1
    
    idGen, err := config.GetIDGenerator(UserServiceAppID)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use idGen for user IDs...
}

// Order Service
package main

import "your-project/config"

func main() {
    const OrderServiceAppID = 2
    
    idGen, err := config.GetIDGenerator(OrderServiceAppID)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use idGen for order IDs...
}
```

### 4. Kubernetes Deployment

```go
package main

import (
    "os"
    "strings"
    "github.com/yourusername/goflakeid"
)

func getK8sIDGenerator() (*goflakeid.Generator, error) {
    // In Kubernetes, use pod name for machine ID
    podName := os.Getenv("HOSTNAME") // Set by K8s
    
    // Extract machine ID from pod name
    // Example: user-service-7f8b9c-5 -> machine ID 5
    var machineID uint8
    if parts := strings.Split(podName, "-"); len(parts) > 0 {
        if id, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
            machineID = uint8(id & 0x1F) // Ensure it fits in 5 bits
        }
    }
    
    // Get region from node labels via downward API
    region := uint16(1)
    if r := os.Getenv("NODE_REGION"); r != "" {
        switch r {
        case "us-east-1":
            region = 1
        case "us-west-2":
            region = 2
        case "eu-west-1":
            region = 3
        case "ap-south-1":
            region = 4
        }
    }
    
    config := goflakeid.NewConfig(region, 1, machineID).
        WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    return goflakeid.NewGenerator(*config)
}
```

## Integration Patterns

### 1. Singleton Pattern

```go
package idgen

import (
    "sync"
    "github.com/yourusername/goflakeid"
)

var (
    instance *goflakeid.Generator
    once     sync.Once
)

// GetGenerator returns a singleton ID generator
func GetGenerator() *goflakeid.Generator {
    once.Do(func() {
        config := goflakeid.NewConfig(1, 1, 1).
            WithAutoMachineID()
        
        gen, err := goflakeid.NewGenerator(*config)
        if err != nil {
            panic(err)
        }
        instance = gen
    })
    return instance
}

// Usage anywhere in your app
func CreateOrder() (*Order, error) {
    id, err := idgen.GetGenerator().Generate()
    if err != nil {
        return nil, err
    }
    
    return &Order{ID: id}, nil
}
```

### 2. Dependency Injection

```go
// Wire up in main
func main() {
    // Create ID generator
    idGen, err := createIDGenerator()
    if err != nil {
        log.Fatal(err)
    }
    
    // Inject into services
    userService := services.NewUserService(db, idGen)
    orderService := services.NewOrderService(db, idGen)
    
    // Start server...
}

// Service implementation
type UserService struct {
    db    *sql.DB
    idGen *goflakeid.Generator
}

func NewUserService(db *sql.DB, idGen *goflakeid.Generator) *UserService {
    return &UserService{db: db, idGen: idGen}
}
```

### 3. ID Wrapper Types

```go
// Define domain-specific ID types
type UserID uint64
type OrderID uint64
type ProductID uint64

// ID service with typed methods
type IDService struct {
    userGen    *goflakeid.Generator
    orderGen   *goflakeid.Generator
    productGen *goflakeid.Generator
}

func NewIDService() (*IDService, error) {
    // Different app IDs for different entities
    userGen, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 1, 1))
    orderGen, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 2, 1))
    productGen, _ := goflakeid.NewGenerator(*goflakeid.NewConfig(1, 3, 1))
    
    return &IDService{
        userGen:    userGen,
        orderGen:   orderGen,
        productGen: productGen,
    }, nil
}

func (s *IDService) NewUserID() (UserID, error) {
    id, err := s.userGen.Generate()
    return UserID(id), err
}

func (s *IDService) NewOrderID() (OrderID, error) {
    id, err := s.orderGen.Generate()
    return OrderID(id), err
}

func (s *IDService) NewProductID() (ProductID, error) {
    id, err := s.productGen.Generate()
    return ProductID(id), err
}
```

## Testing & Debugging

### 1. Unit Testing

```go
package yourpackage

import (
    "testing"
    "time"
    "github.com/yourusername/goflakeid"
)

func TestIDGeneration(t *testing.T) {
    config := goflakeid.NewConfig(1, 1, 1)
    gen, err := goflakeid.NewGenerator(*config)
    if err != nil {
        t.Fatal(err)
    }
    
    // Test single generation
    id, err := gen.Generate()
    if err != nil {
        t.Fatal(err)
    }
    
    if id == 0 {
        t.Error("Generated ID should not be zero")
    }
    
    // Test uniqueness
    seen := make(map[uint64]bool)
    for i := 0; i < 10000; i++ {
        id, err := gen.Generate()
        if err != nil {
            t.Fatal(err)
        }
        
        if seen[id] {
            t.Errorf("Duplicate ID generated: %d", id)
        }
        seen[id] = true
    }
}

func TestIDDecoding(t *testing.T) {
    config := goflakeid.NewConfig(5, 3, 10).
        WithEpoch(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    
    gen, _ := goflakeid.NewGenerator(*config)
    
    id, _ := gen.Generate()
    components := gen.Decode(id)
    
    if components.RegionID != 5 {
        t.Errorf("Expected region 5, got %d", components.RegionID)
    }
    
    if components.AppID != 3 {
        t.Errorf("Expected app 3, got %d", components.AppID)
    }
    
    if components.MachineID != 10 {
        t.Errorf("Expected machine 10, got %d", components.MachineID)
    }
}

func TestConcurrentGeneration(t *testing.T) {
    config := goflakeid.NewConfig(1, 1, 1)
    gen, _ := goflakeid.NewGenerator(*config)
    
    const goroutines = 100
    const idsPerGoroutine = 1000
    
    results := make(chan uint64, goroutines*idsPerGoroutine)
    
    // Generate IDs concurrently
    start := time.Now()
    for i := 0; i < goroutines; i++ {
        go func() {
            for j := 0; j < idsPerGoroutine; j++ {
                id, err := gen.Generate()
                if err != nil {
                    t.Error(err)
                    return
                }
                results <- id
            }
        }()
    }
    
    // Collect results
    seen := make(map[uint64]bool)
    for i := 0; i < goroutines*idsPerGoroutine; i++ {
        id := <-results
        if seen[id] {
            t.Errorf("Duplicate ID in concurrent generation: %d", id)
        }
        seen[id] = true
    }
    
    duration := time.Since(start)
    rate := float64(goroutines*idsPerGoroutine) / duration.Seconds()
    t.Logf("Generated %d IDs in %v (%.0f IDs/sec)", 
        goroutines*idsPerGoroutine, duration, rate)
}
```

### 2. Benchmarking

```go
func BenchmarkGenerate(b *testing.B) {
    config := goflakeid.NewConfig(1, 1, 1)
    gen, _ := goflakeid.NewGenerator(*config)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = gen.Generate()
    }
}

func BenchmarkGenerateParallel(b *testing.B) {
    config := goflakeid.NewConfig(1, 1, 1)
    gen, _ := goflakeid.NewGenerator(*config)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = gen.Generate()
        }
    })
}

func BenchmarkBatch1000(b *testing.B) {
    config := goflakeid.NewConfig(1, 1, 1)
    gen, _ := goflakeid.NewGenerator(*config)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = gen.GenerateBatch(1000)
    }
}
```

### 3. Monitoring & Debugging

```go
// Monitor ID generation patterns
func MonitorIDGeneration(gen *goflakeid.Generator) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    var lastCount uint64
    
    for range ticker.C {
        stats := gen.Stats()
        
        // Estimate IDs per second
        currentCount := uint64(stats.CurrentSequence)
        rate := currentCount - lastCount
        lastCount = currentCount
        
        log.Printf("ID Generation Stats:")
        log.Printf("  Region: %d, App: %d, Machine: %d", 
            stats.Config.RegionID, 
            stats.Config.AppID, 
            stats.Config.MachineID)
        log.Printf("  Current Sequence: %d", stats.CurrentSequence)
        log.Printf("  Last Generated: %v", stats.LastTimestamp)
        log.Printf("  Approx Rate: %d IDs/sec", rate)
    }
}

// Debug ID components
func DebugID(gen *goflakeid.Generator, id uint64) {
    components := gen.Decode(id)
    
    fmt.Printf("ID Debug Info for: %d\n", id)
    fmt.Printf("  Binary: %064b\n", id)
    fmt.Printf("  Timestamp: %v\n", components.Timestamp)
    fmt.Printf("  Region ID: %d\n", components.RegionID)
    fmt.Printf("  App ID: %d\n", components.AppID)
    fmt.Printf("  Machine ID: %d\n", components.MachineID)
    fmt.Printf("  Sequence: %d\n", components.Sequence)
    fmt.Printf("  Age: %v\n", time.Since(components.Timestamp))
}
```

## Performance Tuning

### 1. Optimal Bit Layout Selection

```go
// For single-region, high-throughput systems
func HighThroughputConfig() *goflakeid.Config {
    layout := goflakeid.BitLayout{
        TimestampBits: 41,  // ~69 years
        RegionBits:    0,   // Single region
        AppBits:       5,   // 32 apps
        MachineBits:   8,   // 256 machines per app
        SequenceBits:  10,  // 1024 IDs/ms
    }
    
    return goflakeid.NewConfig(0, 1, 1).
        WithBitLayout(layout)
}

// For globally distributed systems
func GlobalConfig() *goflakeid.Config {
    layout := goflakeid.BitLayout{
        TimestampBits: 42,  // ~139 years
        RegionBits:    4,   // 16 regions worldwide
        AppBits:       3,   // 8 apps per region
        MachineBits:   5,   // 32 machines per app
        SequenceBits:  10,  // 1024 IDs/ms
    }
    
    return goflakeid.NewConfig(1, 1, 1).
        WithBitLayout(layout)
}

// For long-term systems with moderate throughput
func LongTermConfig() *goflakeid.Config {
    layout := goflakeid.BitLayout{
        TimestampBits: 45,  // ~1000 years
        RegionBits:    3,   // 8 regions
        AppBits:       3,   // 8 apps
        MachineBits:   5,   // 32 machines
        SequenceBits:  8,   // 256 IDs/ms
    }
    
    return goflakeid.NewConfig(1, 1, 1).
        WithBitLayout(layout)
}
```

### 2. Batch Operations

```go
// Efficient bulk operations
func BulkInsertUsers(users []User, gen *goflakeid.Generator) error {
    // Pre-generate all IDs
    ids, err := gen.GenerateBatch(len(users))
    if err != nil {
        return err
    }
    
    // Assign IDs
    for i := range users {
        users[i].ID = ids[i]
    }
    
    // Bulk insert...
    return bulkInsertToDB(users)
}
```

### 3. Caching Strategies

```go
// Pre-generate IDs for low-latency requirements
type IDPool struct {
    gen  *goflakeid.Generator
    pool chan uint64
}

func NewIDPool(gen *goflakeid.Generator, size int) *IDPool {
    pool := &IDPool{
        gen:  gen,
        pool: make(chan uint64, size),
    }
    
    // Start background ID generation
    go pool.refill()
    
    return pool
}

func (p *IDPool) refill() {
    for {
        id, err := p.gen.Generate()
        if err != nil {
            log.Printf("ID generation error: %v", err)
            time.Sleep(time.Millisecond)
            continue
        }
        
        p.pool <- id
    }
}

func (p *IDPool) Get() uint64 {
    return <-p.pool
}

// Usage
pool := NewIDPool(generator, 10000)
id := pool.Get() // Instant, no generation latency
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Clock Skew Errors

```go
// Problem: "clock moved backwards" error
// Solution: Use NTP synchronization and handle gracefully

func GenerateWithRetry(gen *goflakeid.Generator, maxRetries int) (uint64, error) {
    for i := 0; i < maxRetries; i++ {
        id, err := gen.Generate()
        if err == nil {
            return id, nil
        }
        
        if errors.Is(err, goflakeid.ErrClockBackwards) {
            log.Printf("Clock skew detected, waiting...")
            time.Sleep(time.Millisecond * time.Duration(i+1))
            continue
        }
        
        return 0, err
    }
    
    return 0, fmt.Errorf("failed after %d retries", maxRetries)
}
```

#### 2. Sequence Overflow

```go
// Problem: Generating too many IDs per millisecond
// Solution: Use multiple generators or adjust bit layout

// Option 1: Multiple generators with different machine IDs
func CreateLoadBalancedGenerators(count int) ([]*goflakeid.Generator, error) {
    generators := make([]*goflakeid.Generator, count)
    
    for i := 0; i < count; i++ {
        config := goflakeid.NewConfig(1, 1, uint8(i))
        gen, err := goflakeid.NewGenerator(*config)
        if err != nil {
            return nil, err
        }
        generators[i] = gen
    }
    
    return generators, nil
}

// Option 2: Increase sequence bits
config := goflakeid.NewConfig(1, 1, 1).
    WithBitLayout(goflakeid.BitLayout{
        TimestampBits: 41,
        RegionBits:    3,
        AppBits:       3,
        MachineBits:   5,
        SequenceBits:  12, // 4096 IDs/ms instead of 1024
    })
```

#### 3. ID Collisions

```go
// Debugging potential collisions
func VerifyUniqueConfiguration(configs []*goflakeid.Config) error {
    seen := make(map[string]bool)
    
    for _, cfg := range configs {
        key := fmt.Sprintf("%d-%d-%d", cfg.RegionID, cfg.AppID, cfg.MachineID)
        if seen[key] {
            return fmt.Errorf("duplicate configuration: region=%d, app=%d, machine=%d",
                cfg.RegionID, cfg.AppID, cfg.MachineID)
        }
        seen[key] = true
    }
    
    return nil
}
```

#### 4. Performance Issues

```go
// Profile ID generation
func ProfileIDGeneration(gen *goflakeid.Generator, duration time.Duration) {
    start := time.Now()
    count := 0
    
    for time.Since(start) < duration {
        _, err := gen.Generate()
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        count++
    }
    
    elapsed := time.Since(start)
    rate := float64(count) / elapsed.Seconds()
    
    fmt.Printf("Performance Profile:\n")
    fmt.Printf("  Duration: %v\n", elapsed)
    fmt.Printf("  IDs Generated: %d\n", count)
    fmt.Printf("  Rate: %.2f IDs/second\n", rate)
    fmt.Printf("  Avg Latency: %.2f ns/ID\n", float64(elapsed.Nanoseconds())/float64(count))
}
```

## Best Practices Summary

1. **Choose appropriate bit layout** based on your scale
2. **Set epoch close to launch date** to maximize timestamp range
3. **Use consistent configuration** across all instances
4. **Monitor sequence numbers** in production
5. **Handle errors gracefully** especially clock skew
6. **Use batch generation** for bulk operations
7. **Test thoroughly** including concurrent scenarios
8. **Document your ID scheme** for future maintainers

## Migration Guide

If migrating from another ID system:

```go
// Gradual migration strategy
type HybridIDService struct {
    oldSystem OldIDGenerator
    newSystem *goflakeid.Generator
    cutoffDate time.Time
}

func (h *HybridIDService) GenerateID() (uint64, error) {
    if time.Now().After(h.cutoffDate) {
        return h.newSystem.Generate()
    }
    return h.oldSystem.Generate()
}

func (h *HybridIDService) IsNewID(id uint64) bool {
    // Decode and check timestamp
    components := h.newSystem.Decode(id)
    return components.Timestamp.After(h.cutoffDate)
}
```