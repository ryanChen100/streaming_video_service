package config

import "time"

// APIGateway definition api_gateway YAML structure
type APIGateway struct {
	Port             string        `mapstructure:"port"`
	MemberService    ServiceConfig `mapstructure:"member"`
	StreamingService ServiceConfig `mapstructure:"streaming"`
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

// Streaming definition streaming_service YAML structure
type Streaming struct {
	Port string `mapstructure:"port"`
	IP   string `mapstructure:"ip"`

	PostgreSQL DatabaseConfig `mapstructure:"pg"`
	MinIO      MinIOConfig    `mapstructure:"minio"`
	RabbitMQ   RabbitMQConfig `mapstructure:"rabbit_mq"`
	// KafKa      KafkaConfig    `mapstructure:"kafka"`
}

// ServiceConfig definition service port & name
type ServiceConfig struct {
	IP   string `mapstructure:"service_ip"`
	Port string `mapstructure:"service_port"`
}

// RedisConfig definition redis setting
type RedisConfig struct {
	RedisDB int `mapstructure:"redis_db"`
}

// DatabaseConfig definition db setting
type DatabaseConfig struct {
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	User          string        `mapstructure:"user"`
	Password      string        `mapstructure:"password"`
	Database      string        `mapstructure:"database"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	RetryCount    int           `mapstructure:"retry_count"`
}

// MinIOConfig definition minio setting
type MinIOConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	BucketName string `mapstructure:"bucket_name"`
	UseSSL     bool   `mapstructure:"use_ssl"`

	RetryInterval time.Duration `mapstructure:"retry_interval"`
	RetryCount    int           `mapstructure:"retry_count"`
}

// RabbitMQConfig definition rabbit setting
type RabbitMQConfig struct {
	Port          string        `mapstructure:"port"`
	IP            string        `mapstructure:"host"`
	User          string        `mapstructure:"user"`
	Password      string        `mapstructure:"password"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	RetryCount    int           `mapstructure:"retry_count"`
}

// KafkaConfig definition kafka setting
type KafkaConfig struct {
	Brokers       []string      `mapstructure:"brokers"`
	Topic         string        `mapstructure:"topic"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	RetryCount    int           `mapstructure:"retry_count"`
}
