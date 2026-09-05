package main

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB     // own DB (referral tables)
var authDB *sql.DB  // taskAuth DB (for superuser checks)
var billDB *sql.DB  // taskBill DB (for billing_referral_edge, commission_accrual)

func openDB(dsn string, repoRoot string) error {
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		return err
	}
	// Schema/seed: dataMigrate/taskReferral via 9999 init / `migrate` CLI only.
	return nil
}

func openAuthDB(dsn string) error {
	var err error
	authDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	authDB.SetMaxOpenConns(2)
	authDB.SetMaxIdleConns(1)
	authDB.SetConnMaxLifetime(5 * time.Minute)
	authDB.SetConnMaxIdleTime(2 * time.Minute)
	return authDB.Ping()
}

func openBillDB(dsn string) error {
	var err error
	billDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	billDB.SetMaxOpenConns(3)
	billDB.SetMaxIdleConns(1)
	billDB.SetConnMaxLifetime(5 * time.Minute)
	billDB.SetConnMaxIdleTime(2 * time.Minute)
	return billDB.Ping()
}

func timeNowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.000000")
}
