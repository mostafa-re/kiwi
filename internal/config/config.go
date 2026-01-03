package config

import (
	"os"
	"strings"
)

// Version info - set at build time via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Role defines the node role in the cluster
type Role string

const (
	RoleMaster Role = "master"
	RoleSlave  Role = "slave"
)

// Config holds application configuration
type Config struct {
	Port         string
	DatabasePath string
	AppName      string
	Version      string
	GitCommit    string
	BuildTime    string

	// Replication settings
	NodeID     string   // Unique identifier for this node
	Role       Role     // master or slave
	GRPCPort   string   // Port for gRPC replication service
	MasterAddr string   // Master address (for slaves to connect)
	SlaveAddrs []string // Slave addresses (for master to replicate to)
}

// Load reads configuration from environment variables with defaults
func Load() *Config {
	role := Role(getEnv("ROLE", "master"))
	if role != RoleMaster && role != RoleSlave {
		role = RoleMaster
	}

	slaveAddrs := []string{}
	if addrs := getEnv("SLAVE_ADDRS", ""); addrs != "" {
		slaveAddrs = strings.Split(addrs, ",")
	}

	return &Config{
		Port:         getEnv("PORT", "3300"),
		DatabasePath: getEnv("DB_PATH", "./data"),
		AppName:      "kiwi",
		Version:      Version,
		GitCommit:    GitCommit,
		BuildTime:    BuildTime,

		NodeID:     getEnv("NODE_ID", "node-1"),
		Role:       role,
		GRPCPort:   getEnv("GRPC_PORT", "50051"),
		MasterAddr: getEnv("MASTER_ADDR", ""),
		SlaveAddrs: slaveAddrs,
	}
}

// IsMaster returns true if this node is the master
func (c *Config) IsMaster() bool {
	return c.Role == RoleMaster
}

// IsSlave returns true if this node is a slave
func (c *Config) IsSlave() bool {
	return c.Role == RoleSlave
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
