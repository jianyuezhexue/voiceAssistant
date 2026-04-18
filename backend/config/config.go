package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
}

// Asr 封装了初始化实时语音识别所需的基本配置
type Asr struct {
	AppKey             string
	Token              string
	VocabularyId       string
	CustomizationId    string
	MaxSentenceSilence int
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
