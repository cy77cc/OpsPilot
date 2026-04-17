// Package server 提供 HTTP 服务器启动和管理功能。
//
// 本文件实现基于 Gin 的 HTTP 服务器，支持优雅关闭。
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// @title           k8s Manager API
// @version         1.0
// @description     devops台后端接口
// @termsOfService  https://blog.cy77cc.cn/

// @contact.name   Your Name
// @contact.url    https://github.com/cy77cc
// @contact.email  zhangdp9527@163.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// Start 启动 HTTP 服务器。
//
// 非阻塞调用，在后台启动服务器。
// 当 context 被取消时优雅关闭。
func Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("nil context")
	}

	started := make(chan struct{})
	serveErr := make(chan error, 1)
	done := make(chan struct{})
	go startServer(ctx, started, serveErr, done)

	select {
	case err := <-serveErr:
		return err
	case <-started:
	case <-ctx.Done():
		select {
		case err := <-serveErr:
			return err
		default:
			return ctx.Err()
		}
	}

	logger.L().Info("http server started", logger.String("addr", fmt.Sprintf("%s:%d", config.CFG.Server.Host, config.CFG.Server.Port)))

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.L().Info("Shutting Down...........")
		<-done
		select {
		case err := <-serveErr:
			return err
		default:
			return nil
		}
	}
}

// startServer 启动 Gin 服务。
//
// 初始化服务上下文、创建路由、启动监听。
// 支持优雅关闭，超时时间为 10 秒。
func startServer(ctx context.Context, started chan struct{}, serveErr chan error, done chan struct{}) {
	defer close(done)

	srv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", config.CFG.Server.Host, config.CFG.Server.Port),
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		serveErr <- err
		return
	}
	defer listener.Close()

	svcCtx := svc.MustNewServiceContext()
	srv.Handler = NewRouter(ctx, svcCtx)

	go func() {
		<-ctx.Done()

		logger.L().Info("http server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.L().Error("http shutdown error", logger.Error(err))
		}
	}()

	close(started)

	// 阻塞监听
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serveErr <- err
	}
}
