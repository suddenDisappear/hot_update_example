package main

import (
	"context"
	"errors"
	"fmt"
	"hot_update/infrastructure/config"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func main() {
	// pprof
	go http.ListenAndServe(":9999", nil)
	// application
	err := start(context.Background())
	if err != nil {
		panic(err)
	}
}

func configure() error {
	return config.MustLoadConfig()
}

func reloadHttpServer(srv *http.Server) (*http.Server, error) {
	if srv == nil {
		return nil, errors.New("invalid server instance")
	}
	err := srv.Shutdown(context.Background())
	if err != nil {
		return nil, err
	}
	reloadServer := initHttpServer()
	return reloadServer, nil
}

func initHttpServer() *http.Server {
	handler := http.NewServeMux()
	handler.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World!")
	}))
	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.C.Http.Host, config.C.Http.Port),
		Handler: handler,
	}
	fmt.Printf("server at %s:%d\n", config.C.Http.Host, config.C.Http.Port)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return &srv
}

// Start 启动服务
func start(ctx context.Context) error {
	var state int32 = 1
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	configure()
	srv := initHttpServer()
	// 使用debounce包裹防止config短时间内多次变更
	debounceWrap := debounce(time.Second)
	var mu sync.Mutex
	viper.OnConfigChange(func(in fsnotify.Event) {
		if in.Op&fsnotify.Write == 0 {
			return
		}
		debounceWrap(func() {
			mu.Lock()
			defer mu.Unlock()
			err := viper.Unmarshal(&config.C)
			if err != nil {
				panic(err)
			}
			srv, err = reloadHttpServer(srv)
			if err != nil {
				panic(err)
			}
		})
	})

EXIT:
	for {
		sig := <-sc
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			atomic.CompareAndSwapInt32(&state, 1, 0)
			break EXIT
		case syscall.SIGHUP:
		default:
			break EXIT
		}
	}

	err := srv.Shutdown(context.Background())
	if err != nil {
		return err
	}

	time.Sleep(time.Second)
	os.Exit(int(atomic.LoadInt32(&state)))
	return nil
}

func debounce(delay time.Duration) func(fn func()) {
	var t *time.Timer
	return func(fn func()) {
		if t != nil {
			t.Stop()
		}
		t = time.AfterFunc(delay, fn)
	}
}
