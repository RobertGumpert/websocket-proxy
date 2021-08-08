package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"ws-server/src/proxy"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type message struct {
	Content string `json:"content"`
}

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(" ---> pr. dir: ", dir)
	err = os.Chdir(dir)
	if err != nil {
		log.Fatal(err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(" ---> execute dir: ", pwd)
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	port, exist := os.LookupEnv("PORT")
	if !exist {
		log.Fatal("PORT NOT FOUND.")
	}
	maxCountClients := func() int {
		env, exist := os.LookupEnv("CLIENTS_LIMIT")
		if !exist {
			log.Fatal("CLIENTS_LIMIT NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	maximumLoad := func() int {
		env, exist := os.LookupEnv("MESSAGES_LIMIT")
		if !exist {
			log.Fatal("MESSAGES_LIMIT NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	maximumPoolSize := func() int {
		env, exist := os.LookupEnv("POOL_SIZE_LIMIT")
		if !exist {
			log.Fatal("POOL_SIZE_LIMIT NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	maximumCountRepeat := func() int {
		env, exist := os.LookupEnv("REPEAT_LIMIT")
		if !exist {
			log.Fatal("REPEAT_LIMIT NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	proxyServer := proxy.NewServer(
		maxCountClients,
		maximumLoad,
		maximumPoolSize,
		maximumCountRepeat,
	)
	server := gin.Default()
	server.GET("/switch/to/ws", func(c *gin.Context) {
		err := proxyServer.AddNewClient(
			http.ResponseWriter(c.Writer),
			c.Request,
		)
		if err != nil {
			c.AbortWithStatus(http.StatusLocked)
			return
		}
	})
	server.POST("/send/message", func(c *gin.Context) {
		msg := new(message)
		if err := c.BindJSON(msg); err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		proxyServer.WriteMessage(
			websocket.TextMessage,
			[]byte(msg.Content),
			fmt.Sprintf("127.0.0.1%s", port),
		)
		c.AbortWithStatus(http.StatusOK)
	})
	err := server.Run(port)
	if err != nil {
		log.Fatal(err)
	}
}
