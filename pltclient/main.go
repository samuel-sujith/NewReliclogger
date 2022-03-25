package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	rand.Seed(time.Now().UnixNano())
	ticker := time.NewTicker(500 * time.Millisecond)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)
				rno := rand.Intn(5)
				switch rno {
				case 0:
					fmt.Println("client push 0")
					httpcall("http://localhost:8000/redisup")
				case 1:
					fmt.Println("client push 1")
					httpcall("http://localhost:8000/redisdown")
				case 2:
					fmt.Println("client push 2")
					httpcall("http://localhost:8000/redisslow")
				case 3:
					fmt.Println("client push 3")
					httpcall("http://localhost:8000/gitupstatus")
				case 4:
					fmt.Println("client push 4")
					httpcall("http://localhost:8000/gitdownstatus")
				default:
					fmt.Println("client push 5")
					break
				}
			}
		}
	}()

	sig := <-sigs
	fmt.Println()
	fmt.Println(sig)
	ticker.Stop()
	done <- true
	fmt.Println("Ticker stopped")
}

func httpcall(path string) {
	c := http.Client{Timeout: time.Duration(1) * time.Second}
	resp, err := c.Get(path)
	if err != nil {
		fmt.Printf("Error %s", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Error %s", err)
		return
	}

	fmt.Printf("Body : %s", body)
}
