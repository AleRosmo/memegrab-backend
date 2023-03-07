package main

import (
	"database/sql"
	"fmt"
	"log"
	"memegrab/sessions"
	"time"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type pgConf struct {
	ip       string
	port     string
	user     string
	password string
	db       string
	sslmode  string
}

func pgInit(conf pgConf) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", conf.ip, conf.port, conf.user, conf.password, conf.db, conf.sslmode)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println("Failed to connect to DB")
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		log.Println("Failed to ping DB")
		return nil, err
	}
	log.Println("Successfully initialized DB connection")
	return db, nil
}

func dbLogin(db *sql.DB, username string) (*sessions.Credentials, error) {
	sqlStatement := `SELECT id, username, password FROM users.login WHERE username=$1;`

	var id int
	var hash string

	row := db.QueryRow(sqlStatement, username)

	switch err := row.Scan(&id, &username, &hash); err {
	case sql.ErrNoRows:
		log.Println("No rows return")
		return nil, err
	case nil:
		creds := sessions.Credentials{
			ID:       id,
			Username: username,
			Password: hash,
		}
		return &creds, nil
	default:
		log.Println("Login general error")
		return nil, err
	}
}

func userRead(db *sql.DB, id int) (userProfile *profile, err error) {
	var username string
	var displayName string
	var permissions []string
	var isOnline bool
	var lastLogin time.Time
	var lastOffline time.Time

	sqlStatement := `SELECT * FROM public.all_user_profiles WHERE id = $1`

	var row *sql.Row

	if id == 0 {
		return userProfile, err
	}
	row = db.QueryRow(sqlStatement, id)
	// Here means: it assigns err with the row.Scan()
	// then "; err" means use "err" in the "switch" statement
	switch err := row.Scan(&id, &username, &displayName, pq.Array(&permissions), &isOnline, &lastLogin, &lastOffline); err {
	case sql.ErrNoRows:
		log.Println("DATABASE", "No USER found!")
		return userProfile, err
	case nil:
		userProfile := &profile{
			ID:          id,
			Username:    username,
			Permissions: permissions,
			DisplayName: displayName,
			LastOffline: lastOffline,
			IsOnline:    isOnline,
		}
		return userProfile, nil
	default:
		log.Println("DATABASE", "Error in UserRead")
		return userProfile, err
	}
}

func testInitGorm(conf pgConf) (*gorm.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", conf.ip, conf.port, conf.user, conf.password, conf.db, conf.sslmode)
	db := postgres.Open(psqlInfo)
	gorm, err := gorm.Open(db, &gorm.Config{QueryFields: true})
	if err != nil {
		return nil, err
	}
	return gorm, nil
}
