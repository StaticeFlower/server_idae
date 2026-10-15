// Package database เชื่อมต่อ Postgres ของ Supabase ผ่าน GORM
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ideaspark-api/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	gormLogger := logger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             300 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			// ไม่แสดงค่า parameter ใน log (กัน password hash / ข้อมูลผู้ใช้หลุดลง log)
			ParameterizedQueries: true,
		},
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
		// จำเป็นเมื่อต่อผ่าน Supabase pooler แบบ transaction mode (พอร์ต 6543)
		// เพราะ pooler ไม่รองรับ prepared statement; ใช้กับ session mode/direct ได้ปกติ
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: gormLogger,
		// แปลง unique violation (23505) เป็น gorm.ErrDuplicatedKey
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// Migrate สร้าง/แก้ตารางด้วย GORM AutoMigrate
// ใช้เลือกอย่างใดอย่างหนึ่งกับ migrations/001_init.sql ไม่ควรใช้ทั้งสองอย่าง
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Idea{})
}
