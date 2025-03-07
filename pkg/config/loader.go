package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// EnvInfo 集合服務端口 from .env
type EnvInfo struct {
	// image name
	APIGateway    string
	MemberService string
	ChatService   string
	Streaming     string

	// service ports
	APIGatewayPort    string
	MemberServicePort string
	ChatServicePort   string
	StreamingPort     string

	// service yaml path
	APIGatewayYAMLPath    string
	MemberServiceYAMLPath string
	ChatServiceYAMLPath   string
	StreamingYAMLPath     string

	// service log path
	APIGatewayLogPath    string
	MemberServiceLogPath string
	ChatServiceLogPath   string
	StreamingLogPath     string
}

// EnvConfig 集合服務端口
var (
	EnvConfig = initEnv()
	envConfig EnvInfo
	once      sync.Once
	env       string
)

func initEnv() EnvInfo {
	once.Do(func() {

		path, err := GetPath(".env", 5)
		if err != nil {
			log.Printf("Warning: Could not get .env path: %v", err)
		}

		if err := godotenv.Load(path); err != nil {
			log.Printf("Warning: Could not load .env file: %v", err)
		}

		env = os.Getenv("ENV")

		envConfig = EnvInfo{
			APIGateway:    os.Getenv("API_GATEWAY"),
			MemberService: os.Getenv("MEMBER_SERVICE"),
			ChatService:   os.Getenv("CHAT_SERVICE"),
			Streaming:     os.Getenv("STREAMING_SERVICE"),

			APIGatewayPort:    os.Getenv("API_GATEWAY_PORT"),
			MemberServicePort: os.Getenv("MEMBER_SERVICE_PORT"),
			ChatServicePort:   os.Getenv("CHAT_SERVICE_PORT"),
			StreamingPort:     os.Getenv("STREAMING_SERVICE_PORT"),

			APIGatewayYAMLPath:    os.Getenv("API_GATEWAY_YAML"),
			MemberServiceYAMLPath: os.Getenv("MEMBER_SERVICE_YAML"),
			ChatServiceYAMLPath:   os.Getenv("CHAT_SERVICE_YAML"),
			StreamingYAMLPath:     os.Getenv("STREAMING_SERVICE_YAML"),

			APIGatewayLogPath:    os.Getenv("API_GATEWAY_LOG"),
			MemberServiceLogPath: os.Getenv("MEMBER_SERVICE_LOG"),
			ChatServiceLogPath:   os.Getenv("CHAT_SERVICE_LOG"),
			StreamingLogPath:     os.Getenv("STREAMING_SERVICE_LOG"),
		}
		fmt.Println("Service:", envConfig)
	})

	return envConfig
}

// IsProduction check run env
func IsProduction() bool {
	var b bool
	if env == "production" {
		b = true
	}
	return b
}

// IsLocal check run env
func IsLocal() bool {
	var b bool
	if env == "local" {
		b = true
	}
	return b
}

// LoadConfig 加載配置
func LoadConfig[T any](serviceName string, configPath string) T {
	v := viper.New()
	// 設置配置文件基本信息
	v.SetConfigName(serviceName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	// 自動讀取環境變數
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 讀取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	// 獲取配置文件的內容
	rawConfig, err := os.ReadFile(v.ConfigFileUsed())
	if err != nil {
		log.Fatalf("Error reading raw config file: %v", err)
	}

	// 替換 ${} 占位符為環境變數的值
	expandedConfig := os.ExpandEnv(string(rawConfig))

	// 使用 Viper 再次解析替換後的配置
	if err := v.ReadConfig(bytes.NewBuffer([]byte(expandedConfig))); err != nil {
		log.Fatalf("Error reading expanded config: %v", err)
	}

	// 解構到 Config 結構
	var cfg T
	fmt.Printf("!!! :%v\n", cfg)
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}
	fmt.Printf("cfg: %v\n", cfg)
	return cfg
}

// GetRedisSetting get redis setting from .env
func GetRedisSetting() (string, []string) {
	path, err := GetPath(".env", 5)
	if err != nil {
		log.Printf("Warning: Could not get .env path: %v", err)
	}

	if err := godotenv.Load(path); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// 提取所有环境变量
	envs := os.Environ()

	// 保存解析后的 Sentinel 地址
	var (
		masterName    string
		sentinelAddrs []string
	)

	// 动态解析 REDIS_SENTINEL*_IP 和端口
	for _, env := range envs {
		// 解析环境变量名和值
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		// 匹配 REDIS_SENTINEL*_IP
		if strings.HasPrefix(key, "REDIS_SENTINEL") && strings.HasSuffix(key, "_IP") {
			// 获取哨兵端口变量名
			portKey := strings.Replace(key, "_IP", "_PORT", 1)
			port := os.Getenv(portKey)
			if port != "" {
				sentinelAddrs = append(sentinelAddrs, fmt.Sprintf("%s:%s", value, port))
			}
		}
	}

	masterName = os.Getenv("REDIS_MASTER_NAME")
	if masterName == "" {
		masterName = "mymaster"
	}

	return masterName, sentinelAddrs
}

// GetPath use fileName loop maxCount find file path
func GetPath(fileName string, maxCount int) (string, error) {
	path := "./" + fileName

	for i := 0; i < maxCount; i++ {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		path = "../" + path
	}
	return "", errors.New(fileName + "can't find path ")
}
