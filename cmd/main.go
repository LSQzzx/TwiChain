package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"twichain/internal/blockchain"
	"twichain/internal/config"
	"twichain/internal/network"
	"twichain/internal/storage"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// 加载配置文件
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 使用配置初始化数据库
	dbPath := filepath.Clean(cfg.Database.Path) // 清理路径
	if !filepath.IsAbs(dbPath) {
		// 如果是相对路径，则相对于当前工作目录
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		dbPath = filepath.Join(currentDir, dbPath)
	}

	store, err := storage.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// 使用配置初始化区块链
	bc := blockchain.NewBlockchain(store, cfg.Blockchain.NodeAddress, cfg.Server.Port)
    if bc == nil {
        log.Fatal("Failed to initialize blockchain")
    }
	log.Println("Blockchain initialized successfully")

	// 启动服务器,使用配置的主机和端口
	server := network.NewServer(bc, cfg.Server.Port)
	log.Printf("Starting blockchain server on %s...\n", cfg.Server.Port)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
