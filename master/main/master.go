package main

import (
	"flag"
	"fmt"
	"github.com/fengjun2016/crontab/master"
	"runtime"
	"time"
)

var (
	conFile string //配置文件路径
)

//解析命令行参数
func initArgs() {
	//master -config ./master.json启动
	flag.StringVar(&conFile, "config", "./master.json", "传入master.json配置文件")
	flag.Parse()
}

//初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU()) //获得运行机器的cpu核数
}

func main() {
	var (
		err error
	)
	//初始化命令行参数
	initArgs()

	//初始化线程
	initEnv()

	//加载配置
	if err = master.InitConfig(conFile); err != nil {
		goto ERR
	}

	//启动JobMgr 任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	//启动API HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	//正常退出
	for {
		time.Sleep(1 * time.Second) //防止http apiServer协程退出
	}
	return

ERR:
	fmt.Println(err)
}
