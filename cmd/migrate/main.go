package main

import (
	"fmt"
	"log"
	"os"

	"golang-sms-broadcast/internal/config"
	"golang-sms-broadcast/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	conf := config.FromEnv()

	fmt.Println("🔗 Connecting to database...")
	fmt.Println("DSN:", conf.DatabaseURL)

	db, err := gorm.Open(postgres.Open(conf.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database")
	fmt.Println("🔄 Running migrations...")

	if err := db.AutoMigrate(&domain.Broadcast{}, &domain.Message{}); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	fmt.Println("✅ Migration complete!")
	fmt.Println("")
	fmt.Println("📊 Checking tables...")

	var tables []string
	db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables)

	if len(tables) == 0 {
		fmt.Println("⚠️  No tables found")
		os.Exit(1)
	}

	fmt.Println("✅ Tables created:")
	for _, table := range tables {
		fmt.Printf("  - %s\n", table)
	}

	// Show table structure
	fmt.Println("")
	fmt.Println("📋 Broadcasts table structure:")
	db.Raw("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'broadcasts'").Scan(&[]map[string]interface{}{})

	fmt.Println("📋 Messages table structure:")
	db.Raw("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'messages'").Scan(&[]map[string]interface{}{})

	fmt.Println("")
	fmt.Println("🎉 Database ready!")
}
