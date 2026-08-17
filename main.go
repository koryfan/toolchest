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
	// 启动时加载持久化设置（缩进等）
	if states, err := config.Load(); err == nil {
		toolset.Init(states)
	}

	app := application.New(application.Options{
		Name:        "tool-dev",
		Description: "开发者自用工具集",
		Services: []application.Service{
			application.NewService(&services.JSONService{}),
			application.NewService(&services.TimeService{}),
			application.NewService(&services.AppService{}),
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

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
