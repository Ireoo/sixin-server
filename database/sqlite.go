package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite" // 使用纯 Go 实现的 SQLite 驱动
)

var sqliteDB *sql.DB

type SQLiteDBClinet struct {
	DB *sql.DB
}

func getSqlitePath() string {
	path := os.Getenv("SQLITE_DB")
	if path == "" {
		path = "./database.db" // 默认值
	}
	return path
}

func InitSqliteDB() error {
	// 连接数据库
	var err error
	sqliteDB, err = sql.Open("sqlite", getSqlitePath())
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 测试连接
	if err = sqliteDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("数据库连接成功建立。")
	return nil
}

// 添加 GetDB 函数
func GetSqliteDB() *sql.DB {
	return sqliteDB
}

// 实现 SQLiteDB 的查询方法
func (s *SQLiteDBClinet) Init(connectionString string) error {
	var err error
	s.DB, err = sql.Open("sqlite", connectionString) // 使用 "sqlite" 而不是 "sqlite3"
	if err != nil {
		return err
	}

	// 初始化数据表
	createTablesQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		wechat_id TEXT UNIQUE,
		name TEXT,
		phone TEXT,
		province TEXT,
		signature TEXT,
		type INTEGER,
		weixin TEXT,
		alias TEXT,
		avatar TEXT,
		city TEXT,
		friend BOOLEAN,
		gender TEXT,
		updated_at INTEGER,
		created_at INTEGER
	);
	CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id TEXT UNIQUE,
		topic TEXT,
		owner_id TEXT,
		member_id_list TEXT,
		avatar TEXT,
		admin_id_list TEXT,
		updated_at INTEGER,
		created_at INTEGER
	);
	CREATE TABLE IF NOT EXISTS room_by_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		alias TEXT,
		topic TEXT,
		updated_at INTEGER,
		created_at INTEGER
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		msg_id TEXT UNIQUE,
		talker_id TEXT,
		listener_id TEXT,
		text TEXT,
		timestamp INTEGER,
		type INTEGER,
		room_id TEXT,
		mention_id_list TEXT,
		updated_at INTEGER,
		created_at INTEGER,
		content TEXT
	);`
	_, err = s.DB.Exec(createTablesQuery)
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	return nil
}

func (db *SQLiteDBClinet) Close() error {
	return db.DB.Close()
}
