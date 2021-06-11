package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var done = make(chan int)

func main() {
	group, ctx := errgroup.WithContext(context.Background())
	group.Go(func() error {
		mux := http.NewServeMux()
		mux.HandleFunc("/close", func(writer http.ResponseWriter, request *http.Request) {
			done <- 1
		})
		s := NewServer(":8080", mux)
		go func() {
			err := s.Start()
			if err != nil {
				fmt.Println(err)
			}
		}()

		select {
		case <-done:
			return s.Stop()
		case <-ctx.Done():
			return errors.New("【通知】信号关闭")
		}
	})
	group.Go(func() error {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-quit:
			return errors.New("信号关闭")
		case <-ctx.Done():
			return errors.New("【通知】http服务关闭")
		}
	})

	// 捕获err
	fmt.Println("开始捕捉err")
	err := group.Wait()
	fmt.Println("=======", err)
}

//http服务
type httpServer struct {
	s       *http.Server
	handler http.Handler
	cxt     context.Context
}

func NewServer(address string, mux http.Handler) *httpServer {
	h := &httpServer{cxt: context.Background()}
	h.s = &http.Server{
		Addr:         address,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	return h
}

func (h *httpServer) Start() error {
	fmt.Println("httpServer start")
	return h.s.ListenAndServe()
}

func (h *httpServer) Stop() error {
	_ = h.s.Shutdown(h.cxt)
	return fmt.Errorf("httpServer结束")
}
