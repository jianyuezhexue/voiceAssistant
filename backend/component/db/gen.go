package db

import (
	"database/sql"
	"errors"
)

// 表字段元数据
type Column struct {
	TableSchema            string        // 数据库名
	TableName              string        // 数据表名
	TableComment           string        // 表注释
	ColumnName             string        // 字段名
	IsNullable             string        // 是否可为null
	DataType               string        // 字段类型(元)：varchar,bigint,longtext,datetime,int,tinyint,decimal,double,text,mediumtext,smallint,timestamp,date,char,enum,bit,longblob,blob,mediumint,set,float,time,tinytext,varbinary
	CharacterMaximumLength sql.NullInt64 // 字符长度
	NumericPrecision       sql.NullInt64 // 数字长度
	NumericScale           sql.NullInt64 // 浮点尾数
	ColumnComment          string        // 字段注释
	ColumnKey              string        // 索引类型：PRI,MUL,UNI
	ColumnDefault          string        // 默认值 0,NULL,CURRENT_TIMESTAMP...
}

// 字段类型转换
// 完整版：https://github.com/asdf072/struct-create/blob/master/main.go
func mysqlType2GoType(col *Column) (string, error) {
	var gt string = ""
	switch col.DataType {
	case "char", "varchar", "enum", "text", "longtext", "mediumtext", "tinytext":
		gt = "string"
	case "blob", "mediumblob", "longblob", "varbinary", "binary":
		gt = "[]byte"
	case "date", "time", "datetime", "timestamp":
		gt = "string"
	case "tinyint", "smallint", "int", "mediumint", "bigint":
		gt = "int64"
	case "float", "decimal", "double":
		gt = "float64"
	}
	if gt == "" {
		n := col.TableName + "." + col.ColumnName
		return "", errors.New("No compatible datatype (" + col.DataType + ") for " + n + " found")
	}
	return gt, nil
}

// 查询表信息
func GetTableInfo(schema, tableName string) ([]*Column, error) {
	colums := []*Column{}
	sql := `select  t.TABLE_SCHEMA             as 'TableSchema',
					t.TABLE_NAME               as 'TableName',
					t.TABLE_COMMENT            as 'TableComment',
					c.COLUMN_NAME              as 'ColumnName',
					c.IS_NULLABLE              as 'IsNullable',
					c.DATA_TYPE                as 'DataType',
					c.CHARACTER_MAXIMUM_LENGTH as 'CharacterMaximumLength',
					c.NUMERIC_PRECISION        as 'NumericPrecision',
					c.NUMERIC_SCALE            as 'NumericScale',
					c.COLUMN_COMMENT           as 'ColumnComment',
					c.COLUMN_KEY               as 'ColumnKey',
					c.COLUMN_DEFAULT           as 'ColumnDefault'
			from INFORMATION_SCHEMA.COLUMNS c join INFORMATION_SCHEMA.TABLES t on c.TABLE_SCHEMA = t.TABLE_SCHEMA
			where t.TABLE_SCHEMA = ?
			and   t.TABLE_NAME = ?
			and   c.TABLE_NAME = ?
			order by c.ORDINAL_POSITION;`
	db := InitDb()
	err := db.Raw(sql, schema, tableName, tableName).Scan(&colums).Error
	if err != nil {
		return nil, err
	}
	return colums, nil
}

// 生成业务实体
func GenBusinessEntity(domain, schema, tableName string) error {

	// 查询表信息

	return nil
}
