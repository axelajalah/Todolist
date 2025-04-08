package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() error {
	// First connect without specifying database
	rootDB, err := sql.Open("mysql", "root:@tcp(localhost:3306)/?parseTime=true")
	if err != nil {
		return fmt.Errorf("root database connection failed: %w", err)
	}
	defer rootDB.Close()

	// Create database if not exists
	_, err = rootDB.Exec("CREATE DATABASE IF NOT EXISTS todolist")
	if err != nil {
		return fmt.Errorf("database creation failed: %w", err)
	}

	// Now connect to our specific database
	DB, err = sql.Open("mysql", "root:@tcp(localhost:3306)/todolist?parseTime=true")
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Create todo table if not exists
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS todo (
			id INT AUTO_INCREMENT PRIMARY KEY,
			judul VARCHAR(255) NOT NULL,
			deskripsi TEXT NOT NULL
		)`)
	if err != nil {
		return fmt.Errorf("todo table creation failed: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Println("Database and tables initialized successfully")
	return nil
}

// Health check function removed
