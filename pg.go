package main

import (
	"database/sql"
	"fmt"
	"log"
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
