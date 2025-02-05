package config

import "time"

// APIGateway definition api_gateway YAML structure
type APIGateway struct {
	Port          string        `mapstructure:"port"`
	MemberService ServiceConfig `mapstructure:"member"`
}

// Member definition member_service YAML structure
type Member struct {
	Port       string        `mapstructure:"port"`
	IP         string        `mapstructure:"ip"`
	SessionTTL time.Duration `mapstructure:"session_ttl"`

	PostgreSQL  DatabaseConfig `mapstructure:"pg"`
	RedisMember RedisConfig    `mapstructure:"redis"`
}

// Chat definition chat_service YAML structure
type Chat struct {
	Port          string
	MongoSQL      DatabaseConfig `mapstructure:"mongo"`
	Redis         RedisConfig    `mapstructure:"redis"`
	MemberService ServiceConfig  `mapstructure:"member"`
}

// ServiceConfig definition service port & name
type ServiceConfig struct {
	Port string `mapstructure:"service_port"`
	Name string `mapstructure:"service_name"`
}

// RedisConfig definition redis setting
type RedisConfig struct {
	RedisDB int `mapstructure:"redis_db"`
}

// DatabaseConfig definition db setting
type DatabaseConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	Database      string `mapstructure:"database"`
	RetryInterval int    `mapstructure:"retry_interval"`
	RetryCount    int    `mapstructure:"retry_count"`
}
