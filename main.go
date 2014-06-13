package main

import (
	"flag"
	"fmt"
	"log"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/bitly/go-nsq"
]	"github.com/garyburd/redigo/redis"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"github.com/zenazn/goji/graceful"
)

var (
	configPath = flag.String("config-file", "./config.toml", "path to qmd config file")
	config     Config

	producer *nsq.Producer
	consumer *nsq.Consumer
	redisDB  *redis.Pool
)

func main() {
	flag.Parse()
	fmt.Printf("Using config file from: %s\n", *configPath)

	var err error
	_, err = toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Setting up producer")
	producer = nsq.NewProducer(config.QueueAddr, nsq.NewConfig())

	fmt.Println("Creating Redis connection pool")
	redisDB = newPool(config.RedisAddr)

	// Setup and start worker.
	fmt.Println("Creating worker")
	worker, err := NewWorker(config)
	if err != nil {
		log.Println(err)
		return
	}
	go worker.Run()

	// Http server
	w := web.New()

	// w.Use(RequestLogger)
	w.Use(middleware.Logger)

	w.Get("/", ServiceRoot)
	w.Post("/", ServiceRoot)
	w.Get("/scripts", GetAllScripts)
	w.Put("/scripts", ReloadScripts)
	w.Post("/scripts/:name", RunScript)
	w.Get("/scripts/:name/logs", GetAllLogs)
	w.Get("/scripts/:name/logs/:id", GetLog)

	err = graceful.ListenAndServe(config.ListenOnAddr, w)
	if err != nil {
		log.Fatal(err)
	}
	graceful.AddSignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	graceful.Wait()
}

// func ExampleMiddleware(h http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		log.Println("Request yooooooo")
// 		h.ServeHTTP(w, r)
// 	})
// }
