package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	ASR      ASRConfig      `yaml:"asr"`
	TTS      TTSConfig      `yaml:"tts"`
	WebRTC   WebRTCConfig   `yaml:"webrtc"`
	Redis    RedisConfig    `yaml:"redis"`
	Session  SessionConfig  `yaml:"session"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Source string `yaml:"source"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode string `yaml:"mode"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LLMConfig LLM 配置
type LLMConfig struct {
	APIKey      string        `yaml:"api_key"`
	Model       string        `yaml:"model"`
	BaseURL     string        `yaml:"base_url"`
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
	TopP        float64       `yaml:"top_p"`
	Timeout     time.Duration `yaml:"timeout"`
}

// ASRConfig ASR 配置
type ASRConfig struct {
	Provider        string        `yaml:"provider"`
	AppKey          string        `yaml:"app_key"`
	AccessKeyID     string        `yaml:"access_key_id"`
	AccessKeySecret string        `yaml:"access_key_secret"`
	ModelPath       string        `yaml:"model_path"`
	TokensPath      string        `yaml:"tokens_path"`
	SampleRate      int           `yaml:"sample_rate"`
	Threshold       float32       `yaml:"threshold"`
	Timeout         time.Duration `yaml:"timeout"`
}

// TTSConfig TTS 配置
type TTSConfig struct {
	Provider     string        `yaml:"provider"`
	ModelPath    string        `yaml:"model_path"`
	LexiconPath  string        `yaml:"lexicon_path"`
	SpeakersPath string        `yaml:"speakers_path"`
	SampleRate   int           `yaml:"sample_rate"`
	Speed        float32       `yaml:"speed"`
	Timeout      time.Duration `yaml:"timeout"`
}

// WebRTCConfig WebRTC 配置
type WebRTCConfig struct {
	STUNServer string `yaml:"stun_server"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// SessionConfig 会话配置
type SessionConfig struct {
	Timeout time.Duration `yaml:"timeout"`
}

// 全局配置实例
var GlobalConfig *Config

// LoadConfig 加载配置
func LoadConfig() *Config {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("读取配置文件失败: %v", err))
	}

	GlobalConfig = &Config{}
	if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
		panic(fmt.Sprintf("解析配置文件失败: %v", err))
	}

	// 设置默认值
	setDefaults(GlobalConfig)

	return GlobalConfig
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "config/config.yaml"
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 2000
	}
	if cfg.LLM.Temperature == 0 {
		cfg.LLM.Temperature = 0.7
	}
	if cfg.LLM.TopP == 0 {
		cfg.LLM.TopP = 0.8
	}
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 30 * time.Second
	}
	if cfg.ASR.SampleRate == 0 {
		cfg.ASR.SampleRate = 16000
	}
	if cfg.ASR.Threshold == 0 {
		cfg.ASR.Threshold = 0.5
	}
	if cfg.ASR.Timeout == 0 {
		cfg.ASR.Timeout = 10 * time.Second
	}
	if cfg.TTS.SampleRate == 0 {
		cfg.TTS.SampleRate = 24000
	}
	if cfg.TTS.Speed == 0 {
		cfg.TTS.Speed = 1.0
	}
	if cfg.TTS.Timeout == 0 {
		cfg.TTS.Timeout = 30 * time.Second
	}
	if cfg.WebRTC.STUNServer == "" {
		cfg.WebRTC.STUNServer = "stun:stun.l.google.com:19302"
	}
	if cfg.Session.Timeout == 0 {
		cfg.Session.Timeout = 30 * time.Minute
	}
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if GlobalConfig == nil {
		return LoadConfig()
	}
	return GlobalConfig
}
