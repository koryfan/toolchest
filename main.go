package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"tool-dev/internal/services"
)

// 前端产物嵌入二进制（frontend/dist 由 npm run build 生成）
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "tool-dev",
		Description: "开发者自用工具集",
		Services: []application.Service{
			application.NewService(&services.JSONService{}),
			application.NewService(&services.TimeService{}),
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
