package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"ws-server/src/wsserver"

	"github.com/RobertGumpert/gopherpc"
	"github.com/gin-gonic/gin"
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
	MAX_POOL_CLIENTS := func() int {
		env, exist := os.LookupEnv("MAX_POOL_CLIENTS")
		if !exist {
			log.Fatal("MAX_POOL_CLIENTS NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	MAX_MESSAGES_PER_SECOND := func() int {
		env, exist := os.LookupEnv("MAX_MESSAGES_PER_SECOND")
		if !exist {
			log.Fatal("MAX_MESSAGES_PER_SECOND NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	MAX_POOL_RETRY_MESSAGES := func() int {
		env, exist := os.LookupEnv("MAX_POOL_RETRY_MESSAGES")
		if !exist {
			log.Fatal("MAX_POOL_RETRY_MESSAGES NOT FOUND.")
		}
		n, err := strconv.Atoi(env)
		if err != nil {
			log.Fatal(err)
		}
		return n
	}()
	proxyServer := wsserver.NewServer(
		MAX_POOL_CLIENTS,
		MAX_MESSAGES_PER_SECOND,
		MAX_POOL_RETRY_MESSAGES,
	)
	server := gin.Default()
	server.GET("/switch/to/ws", func(c *gin.Context) {
		err := proxyServer.NewClient(
			http.ResponseWriter(c.Writer),
			c.Request,
		)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	})
	server.POST("/rpc", func(c *gin.Context) {
		jsonData, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			response := gopherpc.Error(
				gopherpc.ErrParse,
				err.Error(),
			)
			c.AbortWithStatusJSON(http.StatusBadRequest, response)
			return
		}
		request, response, err := gopherpc.ParseRequest(jsonData)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, response)
			return
		}
		switch request.Method {
		case "echo":
			report := proxyServer.GetReport()
			response, err :=request.Response(report)
			if err != nil {
				response := gopherpc.Error(
					gopherpc.ErrInternalError,
					err.Error(),
				)
				c.AbortWithStatusJSON(http.StatusBadRequest, response)
				return
			}
			c.AbortWithStatusJSON(http.StatusOK, response)
			return
		case "sendMessage":
			response, err := request.ParseParams(new(message))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, response)
				return
			}
			proxyServer.SendMessage(
				request.Params.(*message).Content,
			)
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		if err != nil {
			c.AbortWithStatus(http.StatusLocked)
			return
		}
	})

	err := server.Run(port)
	if err != nil {
		log.Fatal(err)
	}
}
