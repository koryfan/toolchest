package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"tool-dev/internal/config"
	"tool-dev/internal/services"
	"tool-dev/internal/toolset"
)

// 前端产物嵌入二进制（frontend/dist 由 npm run build 生成）
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 启动时加载持久化设置（缩进/隧道配置等），装配工具
	cfg := config.New()
	cfg.Load()
	toolset.Init(cfg)

	app := application.New(application.Options{
		Name:        "tool-dev",
		Description: "开发者自用工具集",
		Services: []application.Service{
			application.NewService(services.NewAppService(cfg)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "tool-dev",
		Width:            1000,
		Height:           700,
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	// 退出时停止全部隧道（Wails v3 beta.9 无法拦截窗口关闭，
	// 系统标题栏 X 直接退出，此处保证资源清理）
	defer toolset.CloseAll()

	if err := app.Run(); err != nil {
		toolset.CloseAll()
		log.Fatal(err)
	}
}
