package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// handleFunc 是 mux.HandleFunc 的替代品，。
	// 它使用 http.route 模式丰富了 handler 的 HTTP 测量
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// 为 HTTP 测量配置 "http.route"。
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	// Register handlers.
	handleFunc("/roll", roll)

	// 为整个服务器添加 HTTP 测量。
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	// 平滑处理 SIGINT (CTRL+C) .
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// 设置 OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}
	// 妥善处理停机，确保无泄漏
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// 启动 HTTP server.
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// 等待中断.
	select {
	case err = <-srvErr:
		// 启动 HTTP 服务器时出错.
		return
	case <-ctx.Done():
		// 等待第一个 CTRL+C.
		// 尽快停止接收信号通知.
		stop()
	}

	// 调用 Shutdown 时，ListenAndServe 会立即返回 ErrServerClosed。
	err = srv.Shutdown(context.Background())
	return
}
