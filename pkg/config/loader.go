package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
var EnvConfig = initEnv()

func initEnv() EnvInfo {

	if err := godotenv.Load(getEnvPath()); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	var e EnvInfo

	e.APIGateway = os.Getenv("API_GATEWAY")
	e.MemberService = os.Getenv("MEMBER_SERVICE")
	e.ChatService = os.Getenv("CHAT_SERVICE")
	e.Streaming = os.Getenv("STREAMING")

	e.APIGatewayPort = os.Getenv("API_GATEWAY_PORT")
	e.MemberServicePort = os.Getenv("MEMBER_SERVICE_PORT")
	e.ChatServicePort = os.Getenv("CHAT_SERVICE_PORT")
	e.StreamingPort = os.Getenv("STREAMING_PORT")

	e.APIGatewayYAMLPath = os.Getenv("API_GATEWAY_YAML")
	e.MemberServiceYAMLPath = os.Getenv("MEMBER_SERVICE_YAML")
	e.ChatServiceYAMLPath = os.Getenv("CHAT_SERVICE_YAML")
	e.StreamingYAMLPath = os.Getenv("STREAMING_YAML")

	e.APIGatewayLogPath = os.Getenv("API_GATEWAY_LOG")
	e.MemberServiceLogPath = os.Getenv("MEMBER_SERVICE_LOG")
	e.ChatServiceLogPath = os.Getenv("CHAT_SERVICE_LOG")
	e.StreamingLogPath = os.Getenv("STREAMING_LOG")
	fmt.Println("Service:", e)

	return e
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

	if err := godotenv.Load(getEnvPath()); err != nil {
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

func getEnvPath() string {
	//因docker-compose已掛載.env,可以不用從當前目錄找.env
	//但本地執行仍需要,docker不影響則不註解，但會顯示"Warning: Could not load .env file"
	// 動態獲取當前目錄
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %v", err)
	}

	// 拼接專案根目錄的 .env 路徑
	return filepath.Join(workingDir, "../../.env")
}
