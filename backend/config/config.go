package config

import (
	"os"
	"strconv"
	"time"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	LLM      LLMConfig
	ASR      ASRConfig
	TTS      TTSConfig
	WebRTC   WebRTCConfig
	Redis    RedisConfig
	Session  SessionConfig
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Source string
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode string
	Host string
	Port int
}

// LLMConfig LLM 配置
type LLMConfig struct {
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Timeout     time.Duration
}

// ASRConfig ASR 配置 (sherpa-onnx 本地模型)
type ASRConfig struct {
	ModelPath  string        // 模型路径 (paraformer.onnx)
	TokensPath string        // 词表路径 (tokens.json)
	SampleRate int           // 采样率 (默认16000)
	Threshold  float32       // VAD阈值 (默认0.5)
	Timeout    time.Duration // 超时时间
}

// TTSConfig TTS 配置 (sherpa-onnx 本地模型)
type TTSConfig struct {
	ModelPath    string        // 模型路径 (vits.onnx)
	LexiconPath  string        // 词典路径 (lexicon.txt)
	SpeakersPath string        // 说话人路径 (speakers.txt)
	SampleRate   int           // 采样率 (默认24000)
	Speed        float32       // 语速 (默认1.0)
	Timeout      time.Duration // 超时时间
}

// WebRTCConfig WebRTC 配置
type WebRTCConfig struct {
	STUNServer string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// SessionConfig 会话配置
type SessionConfig struct {
	Timeout time.Duration
}

// 全局配置实例
var GlobalConfig *Config

// LoadConfig 加载配置
func LoadConfig() *Config {
	GlobalConfig = &Config{
		Server: ServerConfig{
			Mode: getEnv("GIN_MODE", "debug"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			MySQL: MySQLConfig{
				Source: getEnv("MYSQL_SOURCE", "root:root@tcp(localhost:3306)/voice_assistant?charset=utf8mb4&parseTime=True&loc=Local"),
			},
		},
		LLM: LLMConfig{
			APIKey:      getEnv("DASHSCOPE_API_KEY", ""),
			Model:       getEnv("LLM_MODEL", "qwen-plus"),
			BaseURL:     getEnv("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
			MaxTokens:   getEnvInt("LLM_MAX_TOKENS", 2000),
			Temperature: getEnvFloat("LLM_TEMPERATURE", 0.7),
			TopP:        getEnvFloat("LLM_TOP_P", 0.8),
			Timeout:     getEnvDuration("LLM_TIMEOUT", 30*time.Second),
		},
		ASR: ASRConfig{
			ModelPath:  getEnv("ASR_MODEL_PATH", "./models/paraformer.onnx"),
			TokensPath: getEnv("ASR_TOKENS_PATH", "./models/tokens.json"),
			SampleRate: getEnvInt("ASR_SAMPLE_RATE", 16000),
			Threshold:  float32(getEnvFloat("ASR_THRESHOLD", 0.5)),
			Timeout:    getEnvDuration("ASR_TIMEOUT", 10*time.Second),
		},
		TTS: TTSConfig{
			ModelPath:    getEnv("TTS_MODEL_PATH", "./models/vits.onnx"),
			LexiconPath:  getEnv("TTS_LEXICON_PATH", "./models/lexicon.txt"),
			SpeakersPath: getEnv("TTS_SPEAKERS_PATH", "./models/speakers.txt"),
			SampleRate:   getEnvInt("TTS_SAMPLE_RATE", 24000),
			Speed:        float32(getEnvFloat("TTS_SPEED", 1.0)),
			Timeout:      getEnvDuration("TTS_TIMEOUT", 30*time.Second),
		},
		WebRTC: WebRTCConfig{
			STUNServer: getEnv("STUN_SERVER", "stun:stun.l.google.com:19302"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Session: SessionConfig{
			Timeout: getEnvDuration("SESSION_TIMEOUT", 30*time.Minute),
		},
	}

	return GlobalConfig
}

// getEnv 获取环境变量字符串
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取环境变量整数
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvFloat 获取环境变量浮点数
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// getEnvDuration 获取环境变量时间间隔
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if GlobalConfig == nil {
		return LoadConfig()
	}
	return GlobalConfig
}
