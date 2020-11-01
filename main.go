package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/1024casts/minichat/repository"

	"github.com/1024casts/minichat/config"
)

var (
	addr = flag.String("addr", ":8080", "http server address")
)

func main() {
	flag.Parse()

	db := config.InitDB()
	defer db.Close()

	config.CreateRedisClient()

	wsServer := NewWebsocketServer(&repository.RoomRepository{Db: db}, &repository.UserRepository{Db: db})
	go wsServer.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(wsServer, w, r)
	})

	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(*addr, nil))
}
