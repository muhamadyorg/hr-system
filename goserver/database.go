package goserver

import (
        "database/sql"
        "fmt"
        "os"

        _ "github.com/lib/pq"
)

var DB *sql.DB

func InitDatabase() error {
        connStr := os.Getenv("VPS_DATABASE_URL")
        if connStr == "" {
                connStr = os.Getenv("DATABASE_URL")
        }
        if connStr == "" {
                return fmt.Errorf("DATABASE_URL environment variable is not set")
        }

        var err error
        DB, err = sql.Open("postgres", connStr)
        if err != nil {
                return fmt.Errorf("failed to open database: %w", err)
        }

        if err = DB.Ping(); err != nil {
                return fmt.Errorf("failed to ping database: %w", err)
        }

        DB.SetMaxOpenConns(25)
        DB.SetMaxIdleConns(5)

        if err = createTables(); err != nil {
                return fmt.Errorf("failed to create tables: %w", err)
        }

        return nil
}

func createTables() error {
        queries := []string{
                `DO $$ BEGIN
                        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
                                CREATE TYPE user_role AS ENUM ('sudo', 'admin');
                        END IF;
                END $$`,
                `DO $$ BEGIN
                        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_status') THEN
                                CREATE TYPE attendance_status AS ENUM ('check_in', 'check_out', 'break_out', 'break_in', 'overtime_in', 'overtime_out');
                        END IF;
                END $$`,
                `CREATE TABLE IF NOT EXISTS users (
                        id SERIAL PRIMARY KEY,
                        username TEXT NOT NULL UNIQUE,
                        password TEXT NOT NULL,
                        full_name TEXT NOT NULL,
                        role user_role NOT NULL DEFAULT 'admin',
                        created_by INTEGER,
                        is_active BOOLEAN NOT NULL DEFAULT true,
                        telegram_user_id TEXT
                )`,
                `CREATE TABLE IF NOT EXISTS groups (
                        id SERIAL PRIMARY KEY,
                        name TEXT NOT NULL UNIQUE,
                        login TEXT,
                        password TEXT,
                        description TEXT,
                        created_by INTEGER NOT NULL,
                        assigned_admin_id INTEGER,
                        is_active BOOLEAN NOT NULL DEFAULT true
                )`,
                `CREATE TABLE IF NOT EXISTS employees (
                        id SERIAL PRIMARY KEY,
                        employee_no TEXT NOT NULL UNIQUE,
                        full_name TEXT NOT NULL,
                        position TEXT,
                        group_id INTEGER,
                        photo_url TEXT,
                        phone TEXT,
                        is_active BOOLEAN NOT NULL DEFAULT true,
                        hikvision_synced BOOLEAN NOT NULL DEFAULT false,
                        telegram_user_id TEXT
                )`,
                `CREATE TABLE IF NOT EXISTS attendance_records (
                        id SERIAL PRIMARY KEY,
                        employee_id INTEGER NOT NULL,
                        employee_no TEXT NOT NULL,
                        event_time TIMESTAMP NOT NULL,
                        status attendance_status NOT NULL DEFAULT 'check_in',
                        photo_url TEXT,
                        device_ip TEXT,
                        device_name TEXT
                )`,
                `CREATE TABLE IF NOT EXISTS admin_group_access (
                        id SERIAL PRIMARY KEY,
                        admin_id INTEGER NOT NULL,
                        group_id INTEGER NOT NULL
                )`,
                `CREATE TABLE IF NOT EXISTS settings (
                        key TEXT PRIMARY KEY,
                        value TEXT NOT NULL
                )`,
                `CREATE TABLE IF NOT EXISTS "session" (
                        "sid" VARCHAR NOT NULL COLLATE "default",
                        "sess" JSON NOT NULL,
                        "expire" TIMESTAMP(6) NOT NULL,
                        PRIMARY KEY ("sid")
                )`,
                `CREATE INDEX IF NOT EXISTS "IDX_session_expire" ON "session" ("expire")`,
        }

        for _, q := range queries {
                if _, err := DB.Exec(q); err != nil {
                        return fmt.Errorf("query failed: %s: %w", q[:50], err)
                }
        }

        return nil
}
