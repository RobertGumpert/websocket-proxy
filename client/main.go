package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"wsclient/src/proxyserver"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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
	var port string
	fmt.Print("PORT: ")
	fmt.Fscan(os.Stdin, &port)
	serverURL := func() *url.URL {
		env, exist := os.LookupEnv("SERVER")
		if !exist {
			log.Fatal("SERVER NOT FOUND.")
		}
		n, err := url.Parse(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()	
	proxyServer := proxyserver.NewServer(
		serverURL,
	)
	server := gin.Default()
	server.GET("/connect/to/server", func(c *gin.Context) {
		err := proxyServer.ConnectToServer()
		if err != nil {
			c.AbortWithStatus(http.StatusLocked)
			return
		}
	})
	server.GET("/disconnect/server", func(c *gin.Context) {
		proxyServer.CloseConnection <- struct{}{}
		c.AbortWithStatus(http.StatusOK)
	})
	server.GET("/send/message", func(c *gin.Context) {
		proxyServer.SendMessages <- struct{}{}
		c.AbortWithStatus(http.StatusOK)
	})
	err := server.Run(port)
	if err != nil {
		log.Fatal(err)
	}
}
