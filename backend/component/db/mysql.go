package db

import (
	"context"
	"database/sql"
	"sync"
	"time"
	"voice-assistant/backend/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	globalDB *gorm.DB
	once     sync.Once
)

func InitDb() *gorm.DB {
	once.Do(func() {
		sqlDB, err := sql.Open("mysql", config.Config.Mysql.DbSource)
		if err != nil {
			panic("数据库连接失败:" + err.Error())
		}

		// 验证数据库实际连通性
		if err = sqlDB.Ping(); err != nil {
			panic("数据库连接失败（Ping）: " + err.Error())
		}

		// 连接池参数优化（根据数据库配置调整）
		sqlDB.SetMaxIdleConns(50)                     // 空闲连接数（建议为CPU核数*2）
		sqlDB.SetMaxOpenConns(100)                    // 最大打开连接数（不超过数据库max_connections）
		sqlDB.SetConnMaxLifetime(28700 * time.Second) // 略小于数据库wait_timeout（默认28800秒）

		// 初始化GORM
		Db, err := gorm.Open(
			mysql.New(mysql.Config{Conn: sqlDB}),
			&gorm.Config{
				NamingStrategy: schema.NamingStrategy{SingularTable: true},
			},
		)
		if err != nil {
			panic(err)
		}

		// 重新初始化db的context
		globalDB = Db.WithContext(context.Background())
	})

	// 兜底判断
	if globalDB == nil {
		panic("数据库连接失败,请检查数据库链接配置")
	}

	return globalDB
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	if globalDB == nil {
		return InitDb()
	}
	return globalDB
}

// 清除分页和偏移量
var ClearOffset = func(db *gorm.DB) *gorm.DB {
	db = db.Limit(-1).Offset(-1)
	return db
}
