package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"lexer-parser/repl"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.POST("/code", func(c *gin.Context) {
		buf := make([]byte, 1024)
		n, _ := c.Request.Body.Read(buf)
		c.Request.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
		raw_code := string(buf[0:n])
		fmt.Println("body: ", raw_code)

		channel := make(chan string, 1)
		wait_channel := make(chan bool, 1)

		go func() {
			channel <- repl.StartHandle(raw_code)
			wait_channel <- true
		}()

		select {
		case <-wait_channel:
			ret := <-channel
			fmt.Println("Response: ", ret)
			c.JSON(http.StatusOK, ret)
		case <- time.After(time.Duration(5) * time.Second):
			fmt.Println("Handle Timeout")
			c.JSON(http.StatusLocked, "")
		}

	})
	r.Run(":8888")
}

// func CommandUsed() {
// user, err := user.Current()
// if err != nil {
// 	panic(err)
// }

// fmt.Printf("Hello %s! This is the Monkey programming language!\n", user.Username)
// fmt.Printf("Typing Commands\n")
// repl.Start(os.Stdin, os.Stdout)
// }
