// Package goflakeid provides a fast, minimal, distributed ID generator inspired by Twitter Snowflake.
// It generates 64-bit unique IDs that are sortable by generation time with zero external dependencies.
package goflakeid

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// Default bit layout: 42 + 4 + 3 + 5 + 10 = 64 bits
const (
	DefaultTimestampBits = 42 // ~139 years from epoch
	DefaultRegionBits    = 4  // 16 regions
	DefaultAppBits       = 3  // 8 apps
	DefaultMachineBits   = 5  // 32 machines per app
	DefaultSequenceBits  = 10 // 1024 IDs per millisecond
)

// Errors
var (
	ErrInvalidBitLayout = errors.New("bit layout must sum to 64")
	ErrClockBackwards   = errors.New("clock moved backwards")
	ErrSequenceOverflow = errors.New("sequence overflow")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// BitLayout defines the bit allocation for ID components
type BitLayout struct {
	TimestampBits uint8
	RegionBits    uint8
	AppBits       uint8
	MachineBits   uint8
	SequenceBits  uint8
}

// Validate ensures bit layout sums to 64
func (b BitLayout) Validate() error {
	total := b.TimestampBits + b.RegionBits + b.AppBits + b.MachineBits + b.SequenceBits
	if total != 64 {
		return fmt.Errorf("%w: got %d bits", ErrInvalidBitLayout, total)
	}
	return nil
}

// DefaultBitLayout returns the default bit layout (42+4+3+5+10=64)
func DefaultBitLayout() BitLayout {
	return BitLayout{
		TimestampBits: DefaultTimestampBits,
		RegionBits:    DefaultRegionBits,
		AppBits:       DefaultAppBits,
		MachineBits:   DefaultMachineBits,
		SequenceBits:  DefaultSequenceBits,
	}
}

// Config holds generator configuration
type Config struct {
	// Required fields
	RegionID  uint16
	AppID     uint8
	MachineID uint8
	
	// Optional fields with defaults
	Epoch        time.Time
	BitLayout    BitLayout
	MachineIDGen func() uint8 // Optional auto-generation function
}

// Validate ensures configuration is valid
func (c *Config) Validate() error {
	if err := c.BitLayout.Validate(); err != nil {
		return err
	}
	
	// Check bounds
	if c.RegionID >= (1 << c.BitLayout.RegionBits) {
		return fmt.Errorf("%w: region ID %d exceeds %d bits", ErrInvalidConfig, c.RegionID, c.BitLayout.RegionBits)
	}
	if c.AppID >= (1 << c.BitLayout.AppBits) {
		return fmt.Errorf("%w: app ID %d exceeds %d bits", ErrInvalidConfig, c.AppID, c.BitLayout.AppBits)
	}
	if c.MachineID >= (1 << c.BitLayout.MachineBits) {
		return fmt.Errorf("%w: machine ID %d exceeds %d bits", ErrInvalidConfig, c.MachineID, c.BitLayout.MachineBits)
	}
	
	return nil
}

// NewConfig creates a configuration with required fields and defaults
func NewConfig(regionID uint16, appID, machineID uint8) *Config {
	return &Config{
		RegionID:  regionID,
		AppID:     appID,
		MachineID: machineID,
		Epoch:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		BitLayout: DefaultBitLayout(),
	}
}

// WithEpoch sets a custom epoch
func (c *Config) WithEpoch(epoch time.Time) *Config {
	c.Epoch = epoch
	return c
}

// WithBitLayout sets a custom bit layout
func (c *Config) WithBitLayout(layout BitLayout) *Config {
	c.BitLayout = layout
	return c
}

// WithAutoMachineID enables automatic machine ID generation
func (c *Config) WithAutoMachineID() *Config {
	c.MachineIDGen = DefaultMachineIDGenerator
	if id := c.MachineIDGen(); id > 0 {
		c.MachineID = id
	}
	return c
}

// Generator generates distributed unique IDs using lock-free algorithms
type Generator struct {
	config Config
	
	// Atomic state for lock-free operation
	state atomic.Uint64
	
	// Pre-computed masks and shifts for performance
	timestampShift uint8
	regionShift    uint8
	appShift       uint8
	machineShift   uint8
	
	maxSequence uint64
	maxTimestamp uint64
	
	// Component masks
	sequenceMask uint64
}

// NewGenerator creates a new ID generator with validation
func NewGenerator(config Config) (*Generator, error) {
	// Apply auto machine ID if configured
	if config.MachineIDGen != nil && config.MachineID == 0 {
		config.MachineID = config.MachineIDGen()
	}
	
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	layout := config.BitLayout
	
	// Calculate shifts
	timestampShift := layout.RegionBits + layout.AppBits + layout.MachineBits + layout.SequenceBits
	regionShift := layout.AppBits + layout.MachineBits + layout.SequenceBits
	appShift := layout.MachineBits + layout.SequenceBits
	machineShift := layout.SequenceBits
	
	// Calculate masks
	maxSequence := uint64((1 << layout.SequenceBits) - 1)
	maxTimestamp := uint64((1 << layout.TimestampBits) - 1)
	
	g := &Generator{
		config:         config,
		timestampShift: timestampShift,
		regionShift:    regionShift,
		appShift:       appShift,
		machineShift:   machineShift,
		maxSequence:    maxSequence,
		maxTimestamp:   maxTimestamp,
		sequenceMask:   maxSequence,
	}
	
	// Initialize state with current timestamp
	now := time.Now().UnixMilli() - config.Epoch.UnixMilli()
	initialState := uint64(now) << 22 // 22 = sequence(10) + machine(5) + app(3) + region(4)
	g.state.Store(initialState)
	
	return g, nil
}

// Generate creates a new unique ID using lock-free atomic operations
func (g *Generator) Generate() (uint64, error) {
	for {
		// Get current time
		now := time.Now().UnixMilli() - g.config.Epoch.UnixMilli()
		if now < 0 {
			return 0, fmt.Errorf("%w: epoch is in the future", ErrClockBackwards)
		}
		if uint64(now) > g.maxTimestamp {
			return 0, fmt.Errorf("timestamp exceeds %d bits", g.config.BitLayout.TimestampBits)
		}
		
		// Load current state
		oldState := g.state.Load()
		oldTimestamp := oldState >> 22
		oldSequence := oldState & g.sequenceMask
		
		var newSequence uint64
		newTimestamp := uint64(now)
		
		// Calculate new sequence
		if newTimestamp == oldTimestamp {
			newSequence = oldSequence + 1
			if newSequence > g.maxSequence {
				// Wait for next millisecond
				time.Sleep(time.Microsecond)
				continue
			}
		} else if newTimestamp > oldTimestamp {
			newSequence = 0
		} else {
			// Clock moved backwards
			return 0, ErrClockBackwards
		}
		
		// Build new state
		newState := (newTimestamp << 22) | newSequence
		
		// Try to update state atomically
		if g.state.CompareAndSwap(oldState, newState) {
			// Successfully updated, build the ID
			id := (newTimestamp << g.timestampShift) |
				(uint64(g.config.RegionID) << g.regionShift) |
				(uint64(g.config.AppID) << g.appShift) |
				(uint64(g.config.MachineID) << g.machineShift) |
				newSequence
			
			return id, nil
		}
		// Another goroutine updated the state, retry
	}
}

// GenerateBatch generates multiple IDs efficiently
func (g *Generator) GenerateBatch(count int) ([]uint64, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	
	ids := make([]uint64, count)
	for i := 0; i < count; i++ {
		id, err := g.Generate()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID %d: %w", i, err)
		}
		ids[i] = id
	}
	return ids, nil
}

// Components represents the decoded components of an ID
type Components struct {
	Timestamp time.Time
	RegionID  uint16
	AppID     uint8
	MachineID uint8
	Sequence  uint16
}

// Decode extracts components from a generated ID
func (g *Generator) Decode(id uint64) Components {
	layout := g.config.BitLayout
	
	// Extract components using bit manipulation
	sequence := uint16(id & g.sequenceMask)
	id >>= layout.SequenceBits
	
	machineID := uint8(id & ((1 << layout.MachineBits) - 1))
	id >>= layout.MachineBits
	
	appID := uint8(id & ((1 << layout.AppBits) - 1))
	id >>= layout.AppBits
	
	regionID := uint16(id & ((1 << layout.RegionBits) - 1))
	id >>= layout.RegionBits
	
	timestamp := id
	
	return Components{
		Timestamp: g.config.Epoch.Add(time.Duration(timestamp) * time.Millisecond),
		RegionID:  regionID,
		AppID:     appID,
		MachineID: machineID,
		Sequence:  sequence,
	}
}

// DefaultMachineIDGenerator derives machine ID from network interface
func DefaultMachineIDGenerator() uint8 {
	// Try hostname first
	if hostname, err := os.Hostname(); err == nil && len(hostname) > 0 {
		hash := uint8(0)
		for _, char := range hostname {
			hash = hash*31 + uint8(char)
		}
		return hash & 0x1F // 5 bits for default layout
	}
	
	// Fallback to MAC address
	interfaces, err := net.Interfaces()
	if err != nil {
		return 1 // Default fallback
	}
	
	for _, iface := range interfaces {
		if len(iface.HardwareAddr) >= 6 {
			// Use last byte of MAC address
			return iface.HardwareAddr[5] & 0x1F
		}
	}
	
	return 1 // Default fallback
}

// MachineIDFromIP derives machine ID from IP address
func MachineIDFromIP(ip string) uint8 {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0
	}
	
	// Use last octet for IPv4, or last 2 bytes for IPv6
	if v4 := parsedIP.To4(); v4 != nil {
		return v4[3] & 0x1F
	}
	
	// IPv6: combine last 2 bytes
	return (parsedIP[14] ^ parsedIP[15]) & 0x1F
}

// Stats returns generator statistics
type Stats struct {
	Config          Config
	CurrentSequence uint16
	LastTimestamp   time.Time
}

// Stats returns current generator statistics
func (g *Generator) Stats() Stats {
	state := g.state.Load()
	timestamp := state >> 22
	sequence := uint16(state & g.sequenceMask)
	
	return Stats{
		Config:          g.config,
		CurrentSequence: sequence,
		LastTimestamp:   g.config.Epoch.Add(time.Duration(timestamp) * time.Millisecond),
	}
}