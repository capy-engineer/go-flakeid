// Package idgen provides a generic, distributed ID generation library.
// Users define their own entity types and configurations when importing.
package goflakeid

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EntityType represents any entity type - users define their own
type EntityType uint8

// BitLayout defines the bit allocation for ID components
type BitLayout struct {
	TimestampBits   uint8 `json:"timestamp_bits"`
	EntityTypeBits  uint8 `json:"entity_type_bits"`
	DatacenterBits  uint8 `json:"datacenter_bits"`
	MachineBits     uint8 `json:"machine_bits"`
	SequenceBits    uint8 `json:"sequence_bits"`
}

// Validate ensures bit layout sums to 64
func (bl BitLayout) Validate() error {
	total := bl.TimestampBits + bl.EntityTypeBits + bl.DatacenterBits + bl.MachineBits + bl.SequenceBits
	if total != 64 {
		return fmt.Errorf("bit layout must sum to 64, got %d", total)
	}
	return nil
}

// DefaultBitLayout provides a sensible default (same as Snowflake-style)
var DefaultBitLayout = BitLayout{
	TimestampBits:  41, // ~69 years from epoch
	EntityTypeBits: 5,  // 32 entity types
	DatacenterBits: 2,  // 4 datacenters
	MachineBits:    6,  // 64 machines per datacenter
	SequenceBits:   10, // 1024 IDs/ms per machine
}

// EntityConfig holds configuration for a single entity type
type EntityConfig struct {
	Type        EntityType `json:"type"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Description string     `json:"description,omitempty"`
}

// Encoder interface allows pluggable encoding strategies
type Encoder interface {
	Encode(id uint64) string
	Decode(encoded string) (uint64, error)
	Name() string
}

// TimeProvider interface allows testable/mockable time sources
type TimeProvider interface {
	UnixMilli() int64
}

// SystemTimeProvider uses system time
type SystemTimeProvider struct{}

func (s SystemTimeProvider) UnixMilli() int64 { return time.Now().UnixMilli() }

// Base62Encoder implements Base62 encoding with configurable alphabet
type Base62Encoder struct {
	alphabet string
}

// NewBase62Encoder creates a Base62 encoder
func NewBase62Encoder(alphabet ...string) *Base62Encoder {
	defaultAlphabet := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if len(alphabet) > 0 && alphabet[0] != "" {
		defaultAlphabet = alphabet[0]
	}
	return &Base62Encoder{alphabet: defaultAlphabet}
}

func (e *Base62Encoder) Name() string { return "base62" }

func (e *Base62Encoder) Encode(num uint64) string {
	if num == 0 {
		return string(e.alphabet[0])
	}
	
	var result []byte
	base := uint64(len(e.alphabet))
	
	for num > 0 {
		result = append([]byte{e.alphabet[num%base]}, result...)
		num /= base
	}
	
	return string(result)
}

func (e *Base62Encoder) Decode(str string) (uint64, error) {
	if str == "" {
		return 0, errors.New("empty string")
	}
	
	var result uint64
	base := uint64(len(e.alphabet))
	
	for _, char := range str {
		index := strings.IndexRune(e.alphabet, char)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", char)
		}
		result = result*base + uint64(index)
	}
	
	return result, nil
}

// HexEncoder implements hexadecimal encoding
type HexEncoder struct{}

func (h HexEncoder) Name() string             { return "hex" }
func (h HexEncoder) Encode(id uint64) string  { return fmt.Sprintf("%x", id) }
func (h HexEncoder) Decode(encoded string) (uint64, error) {
	return strconv.ParseUint(encoded, 16, 64)
}

// Config holds all generator configuration
type Config struct {
	// Core configuration
	DatacenterID uint8        `json:"datacenter_id"`
	MachineID    uint8        `json:"machine_id"`
	Epoch        time.Time    `json:"epoch"`
	
	// Layout configuration
	BitLayout BitLayout `json:"bit_layout"`
	
	// Entity types mapping
	EntityTypes map[EntityType]EntityConfig `json:"entity_types"`
	
	// Optional dependencies
	TimeProvider TimeProvider `json:"-"`
	Encoder      Encoder      `json:"-"`
}

// Validate ensures configuration is valid
func (c Config) Validate() error {
	if err := c.BitLayout.Validate(); err != nil {
		return fmt.Errorf("invalid bit layout: %w", err)
	}
	
	maxDatacenter := uint64(1<<c.BitLayout.DatacenterBits) - 1
	if uint64(c.DatacenterID) > maxDatacenter {
		return fmt.Errorf("datacenter ID %d exceeds maximum %d", c.DatacenterID, maxDatacenter)
	}
	
	maxMachine := uint64(1<<c.BitLayout.MachineBits) - 1
	if uint64(c.MachineID) > maxMachine {
		return fmt.Errorf("machine ID %d exceeds maximum %d", c.MachineID, maxMachine)
	}
	
	// Validate entity types don't exceed bit allocation
	maxEntityType := uint64(1<<c.BitLayout.EntityTypeBits) - 1
	for entityType := range c.EntityTypes {
		if uint64(entityType) > maxEntityType {
			return fmt.Errorf("entity type %d exceeds maximum %d", entityType, maxEntityType)
		}
	}
	
	// Check for duplicate prefixes
	prefixes := make(map[string]EntityType)
	for entityType, config := range c.EntityTypes {
		if existing, exists := prefixes[config.Prefix]; exists {
			return fmt.Errorf("duplicate prefix '%s' for entity types %d and %d", config.Prefix, existing, entityType)
		}
		prefixes[config.Prefix] = entityType
	}
	
	return nil
}

// NewConfig creates a minimal configuration - users add their entity types
func NewConfig(datacenterID, machineID uint8) Config {
	return Config{
		DatacenterID: datacenterID,
		MachineID:    machineID,
		Epoch:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		BitLayout:    DefaultBitLayout,
		EntityTypes:  make(map[EntityType]EntityConfig),
		TimeProvider: SystemTimeProvider{},
		Encoder:      NewBase62Encoder(),
	}
}

// AddEntityType adds an entity type to the configuration
func (c *Config) AddEntityType(entityType EntityType, name, prefix string, description ...string) *Config {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	
	c.EntityTypes[entityType] = EntityConfig{
		Type:        entityType,
		Name:        name,
		Prefix:      prefix,
		Description: desc,
	}
	return c
}

// SetEpoch sets a custom epoch
func (c *Config) SetEpoch(epoch time.Time) *Config {
	c.Epoch = epoch
	return c
}

// SetBitLayout sets a custom bit layout
func (c *Config) SetBitLayout(layout BitLayout) *Config {
	c.BitLayout = layout
	return c
}

// SetEncoder sets a custom encoder
func (c *Config) SetEncoder(encoder Encoder) *Config {
	c.Encoder = encoder
	return c
}

// SetTimeProvider sets a custom time provider
func (c *Config) SetTimeProvider(provider TimeProvider) *Config {
	c.TimeProvider = provider
	return c
}

// IDGenerator generates distributed unique IDs
type IDGenerator struct {
	mu     sync.Mutex
	config Config
	
	// Runtime state
	sequence uint16
	lastTime int64
	
	// Precomputed values for performance
	maxSequence     uint64
	maxEntityType   uint64
	timestampShift  uint8
	entityShift     uint8
	datacenterShift uint8
	machineShift    uint8
	
	// Reverse mapping for decoding
	prefixToEntity map[string]EntityType
}

// NewGenerator creates a new ID generator with the given configuration
func NewGenerator(config Config) (*IDGenerator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	// Set defaults if not provided
	if config.TimeProvider == nil {
		config.TimeProvider = SystemTimeProvider{}
	}
	if config.Encoder == nil {
		config.Encoder = NewBase62Encoder()
	}
	
	// Precompute values for performance
	layout := config.BitLayout
	maxSequence := uint64(1<<layout.SequenceBits) - 1
	maxEntityType := uint64(1<<layout.EntityTypeBits) - 1
	
	timestampShift := layout.EntityTypeBits + layout.DatacenterBits + layout.MachineBits + layout.SequenceBits
	entityShift := layout.DatacenterBits + layout.MachineBits + layout.SequenceBits
	datacenterShift := layout.MachineBits + layout.SequenceBits
	machineShift := layout.SequenceBits
	
	// Build reverse mapping for prefix -> entity type
	prefixToEntity := make(map[string]EntityType)
	for entityType, entityConfig := range config.EntityTypes {
		prefixToEntity[entityConfig.Prefix] = entityType
	}
	
	return &IDGenerator{
		config:          config,
		maxSequence:     maxSequence,
		maxEntityType:   maxEntityType,
		timestampShift:  timestampShift,
		entityShift:     entityShift,
		datacenterShift: datacenterShift,
		machineShift:    machineShift,
		prefixToEntity:  prefixToEntity,
	}, nil
}

// Generate creates a new raw ID for the specified entity type
func (g *IDGenerator) Generate(entityType EntityType) (uint64, error) {
	// Validate entity type is configured
	if _, exists := g.config.EntityTypes[entityType]; !exists {
		return 0, fmt.Errorf("entity type %d not configured", entityType)
	}
	
	if uint64(entityType) > g.maxEntityType {
		return 0, fmt.Errorf("entity type %d exceeds maximum %d", entityType, g.maxEntityType)
	}
	
	g.mu.Lock()
	defer g.mu.Unlock()
	
	now := g.config.TimeProvider.UnixMilli()
	
	if now < g.lastTime {
		return 0, errors.New("clock moved backwards")
	}
	
	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & uint16(g.maxSequence)
		if g.sequence == 0 {
			// Wait for next millisecond
			for now <= g.lastTime {
				now = g.config.TimeProvider.UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}
	
	g.lastTime = now
	
	// Build ID according to bit layout
	timestamp := uint64(now - g.config.Epoch.UnixMilli())
	
	id := (timestamp << g.timestampShift) |
		(uint64(entityType) << g.entityShift) |
		(uint64(g.config.DatacenterID) << g.datacenterShift) |
		(uint64(g.config.MachineID) << g.machineShift) |
		uint64(g.sequence)
	
	return id, nil
}

// GeneratePublic creates a prefixed, encoded public ID
func (g *IDGenerator) GeneratePublic(entityType EntityType) (string, error) {
	id, err := g.Generate(entityType)
	if err != nil {
		return "", err
	}
	
	return g.EncodePublic(id, entityType)
}

// EncodePublic converts a raw ID to public format
func (g *IDGenerator) EncodePublic(id uint64, entityType EntityType) (string, error) {
	entityConfig, exists := g.config.EntityTypes[entityType]
	if !exists {
		return "", fmt.Errorf("entity type %d not configured", entityType)
	}
	
	encoded := g.config.Encoder.Encode(id)
	return fmt.Sprintf("%s_%s", entityConfig.Prefix, encoded), nil
}

// DecodePublic parses a public ID back to raw form and entity type
func (g *IDGenerator) DecodePublic(publicID string) (uint64, EntityType, error) {
	parts := strings.SplitN(publicID, "_", 2)
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid public ID format: missing underscore")
	}
	
	prefix, encoded := parts[0], parts[1]
	
	entityType, exists := g.prefixToEntity[prefix]
	if !exists {
		return 0, 0, fmt.Errorf("unknown entity prefix: %s", prefix)
	}
	
	id, err := g.config.Encoder.Decode(encoded)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode ID: %w", err)
	}
	
	return id, entityType, nil
}

// ParsedID represents the components extracted from an ID
type ParsedID struct {
	Timestamp    time.Time   `json:"timestamp"`
	EntityType   EntityType  `json:"entity_type"`
	DatacenterID uint8       `json:"datacenter_id"`
	MachineID    uint8       `json:"machine_id"`
	Sequence     uint16      `json:"sequence"`
}

// Parse extracts all components from a raw ID
func (g *IDGenerator) Parse(id uint64) ParsedID {
	layout := g.config.BitLayout
	
	sequence := id & g.maxSequence
	id >>= layout.SequenceBits
	
	machineID := id & ((1 << layout.MachineBits) - 1)
	id >>= layout.MachineBits
	
	datacenterID := id & ((1 << layout.DatacenterBits) - 1)
	id >>= layout.DatacenterBits
	
	entityType := EntityType(id & g.maxEntityType)
	id >>= layout.EntityTypeBits
	
	timestamp := id
	
	return ParsedID{
		Timestamp:    time.UnixMilli(int64(timestamp) + g.config.Epoch.UnixMilli()),
		EntityType:   entityType,
		DatacenterID: uint8(datacenterID),
		MachineID:    uint8(machineID),
		Sequence:     uint16(sequence),
	}
}

// GenerateBatch creates multiple IDs efficiently
func (g *IDGenerator) GenerateBatch(entityType EntityType, count int) ([]uint64, error) {
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	if uint64(count) > g.maxSequence {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", count, g.maxSequence)
	}
	
	ids := make([]uint64, 0, count)
	for i := 0; i < count; i++ {
		id, err := g.Generate(entityType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID %d: %w", i, err)
		}
		ids = append(ids, id)
	}
	
	return ids, nil
}

// GetEntityConfig returns the configuration for an entity type
func (g *IDGenerator) GetEntityConfig(entityType EntityType) (EntityConfig, bool) {
	config, exists := g.config.EntityTypes[entityType]
	return config, exists
}

// ListEntityTypes returns all configured entity types
func (g *IDGenerator) ListEntityTypes() []EntityConfig {
	configs := make([]EntityConfig, 0, len(g.config.EntityTypes))
	for _, config := range g.config.EntityTypes {
		configs = append(configs, config)
	}
	return configs
}

// Stats returns current generator statistics
func (g *IDGenerator) Stats() GeneratorStats {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	return GeneratorStats{
		DatacenterID:    g.config.DatacenterID,
		MachineID:       g.config.MachineID,
		Epoch:           g.config.Epoch,
		LastGenerated:   time.UnixMilli(g.lastTime),
		CurrentSequence: g.sequence,
		BitLayout:       g.config.BitLayout,
		EntityCount:     len(g.config.EntityTypes),
		EncoderName:     g.config.Encoder.Name(),
	}
}

// GeneratorStats holds runtime statistics
type GeneratorStats struct {
	DatacenterID    uint8     `json:"datacenter_id"`
	MachineID       uint8     `json:"machine_id"`
	Epoch           time.Time `json:"epoch"`
	LastGenerated   time.Time `json:"last_generated"`
	CurrentSequence uint16    `json:"current_sequence"`
	BitLayout       BitLayout `json:"bit_layout"`
	EntityCount     int       `json:"entity_count"`
	EncoderName     string    `json:"encoder_name"`
}

// Example showing how users would configure their own entity types
func ExampleUsage() {
	// Users define their own entity types
	const (
		EntityTrack    EntityType = 0
		EntityArtist   EntityType = 1
		EntityPlaylist EntityType = 2
		EntityUser     EntityType = 3
		// ... add as many as needed
	)
	
	// Configure the generator with their entity types
	config := NewConfig(1, 5).
		AddEntityType(EntityTrack, "track", "t", "Music track").
		AddEntityType(EntityArtist, "artist", "a", "Music artist").
		AddEntityType(EntityPlaylist, "playlist", "p", "User playlist").
		AddEntityType(EntityUser, "user", "u", "Platform user").
		SetEpoch(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	
	generator, err := NewGenerator(*config)
	if err != nil {
		panic(err)
	}
	
	// Generate IDs
	trackID, _ := generator.Generate(EntityTrack)
	publicID, _ := generator.GeneratePublic(EntityArtist)
	
	fmt.Printf("Track ID: %d\n", trackID)
	fmt.Printf("Public Artist ID: %s\n", publicID) // e.g., "a_9q81j4K2F"
	
	// Parse back
	rawID, entityType, _ := generator.DecodePublic(publicID)
	parsed := generator.Parse(rawID)
	
	fmt.Printf("Generated at: %s, Type: %d\n", parsed.Timestamp, parsed.EntityType)
}