package common

import (
	""
)

//定时任务、
type Job struct {
    Name string  `json:"name"`   //任务名称
    Command string	`json:"command"`  //shell命令
    CronExpr string	`json:"cronExpr"`  //cron表达式
}

func main() {
	
}
	