package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/spf13/viper"
)

// ConfigType 配置结构体类型
type ConfigType struct {
	Server struct {
		Mode string `json:"mode"`
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	Mysql struct {
		DbSource string `json:"dbSource"`
	}
	Redis struct {
		Address  string `json:"address"`
		Password string `json:"password"`
	}
	Asr Asr
	Tts Tts
}

// Asr 封装了初始化实时语音识别所需的基本配置
type Asr struct {
	AppKey             string
	Token              string
	VocabularyId       string
	CustomizationId    string
	MaxSentenceSilence int
}

// Tts 封装了初始化语音合成所需的基本配置
type Tts struct {
	AppKey     string
	Token      string
	Voice      string // 发音人, 默认 xiaoyun
	Format     string // 音频格式, 默认 mp3
	SampleRate int    // 采样率, 默认 16000
	Volume     int    // 音量 0-100, 默认 50
	SpeechRate int    // 语速 -500~500, 默认 0
	PitchRate  int    // 音高 -500~500, 默认 0
}

var Config ConfigType

// LoadConfig 加载配置
func LoadConfig() *ConfigType {
	return &Config
}

func init() {
	viper := viper.New()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 添加多个配置文件搜索路径
	for _, path := range getConfigPaths() {
		viper.AddConfigPath(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置信息错误: %s", err))
	}
	if err := viper.Unmarshal(&Config); err != nil {
		fmt.Println(err)
	}

	// 环境变量覆盖：便于容器化部署时注入监听地址、数据源等，无需修改 config.yaml
	applyEnvOverrides()
}

// applyEnvOverrides 用环境变量覆盖配置项（仅当对应 env 非空时生效）
func applyEnvOverrides() {
	if h := os.Getenv("SERVER_HOST"); h != "" {
		Config.Server.Host = h
	}
	if p := os.Getenv("SERVER_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			Config.Server.Port = v
		}
	}
	if d := os.Getenv("MYSQL_DSN"); d != "" {
		Config.Mysql.DbSource = d
	}
	if a := os.Getenv("REDIS_ADDRESS"); a != "" {
		Config.Redis.Address = a
	}
}

// 获取所有可能的配置文件路径
func getConfigPaths() []string {
	var paths []string

	// 1. 可执行文件所在目录（生产环境）
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths, execDir)
	}

	// 2. 当前工作目录
	if workDir, err := os.Getwd(); err == nil {
		paths = append(paths, workDir)
	}

	// 3. 源代码 config 目录（开发环境）
	if _, filename, _, ok := runtime.Caller(0); ok {
		srcDir := filepath.Dir(filename)
		paths = append(paths, srcDir)
	}

	return paths
}
