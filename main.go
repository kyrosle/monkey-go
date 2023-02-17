package main

import (
	"bytes"
	"context"
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Microsecond * 900))
		defer cancel()
		timer := time.NewTimer(time.Duration(time.Microsecond * 2000))

		buf := make([]byte, 1024)
		n, _ := c.Request.Body.Read(buf)
		c.Request.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
		raw_code := string(buf[0:n])
		fmt.Println("body: ", raw_code)

		channel := make(chan string, 1)
		check := make(chan bool, 1)

		go func(ctx context.Context) {
			res, ok := repl.StartHandle(raw_code)
			channel <- res
			check <- ok
		}(ctx)

		select {
		case <-ctx.Done():
			ret := <-channel
			fmt.Println("Response: ", ret)
			check_ok := <-check
			if check_ok {
				c.JSON(http.StatusOK, ret)
			} else {
				c.JSON(http.StatusNotAcceptable, ret)
			}
			return
		case <-timer.C:
			fmt.Println("Handle Timeout")
			c.JSON(http.StatusNotAcceptable, "Program RunTimeout")
			return
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
