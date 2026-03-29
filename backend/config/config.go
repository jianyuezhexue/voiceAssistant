package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Milvus   MilvusConfig   `yaml:"milvus"`
	ASR      ASRConfig      `yaml:"asr"`
	LLM      LLMConfig      `yaml:"llm"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Source string `yaml:"source"`
	// Host     string `yaml:"host"`
	// Port     int    `yaml:"port"`
	// User     string `yaml:"user"`
	// Password string `yaml:"password"`
	// Database string `yaml:"database"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// MilvusConfig Milvus 配置
type MilvusConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// ASRConfig 语音识别配置
type ASRConfig struct {
	Provider        string `yaml:"provider"`
	AppKey          string `yaml:"app_key"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
}

// LLMConfig 大模型配置
type LLMConfig struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url"`
}

// Load 加载配置文件
func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
