package rvn

import (
	"github.com/go-redis/redis"
	"log"
	"time"
)

var db *redis.Client

func dbConnect() {
	db = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if db == nil {
		log.Fatal("failed to connect to db")
	}
}

func dbAlive() bool {
	_, err := db.Ping().Result()
	if err != nil {
		log.Printf("ping db failed")
		return false
	}
	return true
}

func dbCheckConnection() {
	for db == nil {
		dbConnect()
		time.Sleep(1 * time.Second)
	}

	for !dbAlive() {
		dbConnect()
		time.Sleep(100 * time.Millisecond)
	}
}
