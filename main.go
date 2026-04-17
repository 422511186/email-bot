package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"email-bot/config"
	"email-bot/core"
	"email-bot/tui"
)

var Version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本并退出")
	flag.Parse()

	if *showVersion {
		fmt.Println("email-bot 版本:", Version)
		os.Exit(0)
	}

	// ── 加载配置 ──────────────────────────────────────────────
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌  加载配置失败（%s）: %v\n", *configPath, err)
		fmt.Fprintf(os.Stderr, "    请参考 config.yaml.example。\n")
		os.Exit(1)
	}

	// ── 创建并启动机器人引擎 ─────────────────────────────────
	bot, err := core.NewBot(cfg)
	if err != nil {
		log.Fatalf("创建机器人失败: %v", err)
	}

	go bot.Run()

	// ── 启动 TUI ─────────────────────────────────────────────────
	model := tui.NewModel(bot, cfg)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // 全屏模式
		tea.WithMouseCellMotion(), // 启用鼠标支持
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI 错误: %v", err)
	}

	bot.Stop()
}
