package master

import (
	"encoding/json"
	"github.com/fengjun2016/crontab/common"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

//任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

//单例模式 在golang里面实现起来就是定义一个包级别的全局变量
var (
	//单例对象
	G_apiServer *ApiServer
)

//保存任务接口
//POST job={"name":"job1", "command":"echo hello", "conEXpr":"* * * * * *"}
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	//保存到etcd中
	var (
		err      error
		postJob  string
		job      common.Job
		oldJob   *common.Job
		resBytes []byte
	)
	//1.解析POST表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	//2.取表单中的job字段
	postJob = r.PostForm.Get("job")

	//3.反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	//4.保存到etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	//5.返回正常应答
	if resBytes, err = common.BuildResponse(0, "successs", oldJob); err == nil {
		log.Printf("handleJobSave success response : %v", string(resBytes))
		w.Write(resBytes)
	}
	return
ERR:
	//6.返回异常应答
	if resBytes, err = common.BuildResponse(-1, err.Error(), oldJob); err != nil {
		log.Printf("handleJobSave err response : %v", string(resBytes))
		w.Write(resBytes)
	}
	return
}

//删除任务接口
//POST  /job/delete name=job1
func handleJobDelete(rw http.ResponseWriter, req *http.Request) {
	var (
		err      error
		name     string
		oldJob   *common.Job
		resBytes []byte
	)
	//1.解析表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//获取删除的任务名称
	name = req.PostForm.Get("name")

	//删除etcd中的任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}

	//返回正常响应
	if resBytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		log.Printf("handleJobDelete response: %v", string(resBytes))
		rw.Write(resBytes)
	}
	return
ERR:
	//返回异常响应
	if resBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		log.Printf("handleJobDelete err response: %v", string(resBytes))
		rw.Write(resBytes)
	}
	return
}

//获取所有任务列表
//GET 请求 无须任何参数
func handleJobList(rw http.ResponseWriter, req *http.Request) {
	var (
		err      error
		jobLists []*common.Job
		resBytes []byte
	)

	//获取所有任务列表
	if jobLists, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}

	//返回正常响应
	if resBytes, err = common.BuildResponse(0, "success", jobLists); err == nil {
		log.Printf("handleJobList success response : %v", string(resBytes))
		rw.Write(resBytes)
	}

	return
ERR:
	//返回异常响应信息
	if resBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		log.Printf("handleJobList err response : %v", string(resBytes))
		rw.Write(resBytes)
	}

	return
}

func handleJobKill(rw http.ResponseWriter, req *http.Request) {
	var (
		err      error
		name     string
		resBytes []byte
	)
	//1.解析表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//2.获取参数
	name = req.PostForm.Get("name")

	//3.去etcd中强杀该掉指令和任务
	if err = G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}

	//4.返回成功的正常响应
	if resBytes, err = common.BuildResponse(0, "success", nil); err == nil {
		log.Printf("handleJobKiller success response: %v", string(resBytes))
		rw.Write(resBytes)
	}
	return
ERR:
	//5.返回错误信息的响应
	if resBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		log.Printf("handleJobKiller err response: %v", string(resBytes))
		rw.Write(resBytes)
	}
	return
}

//初始化http服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server //静态文件根目录
		staticDir     http.Dir
		staticHandler http.Handler //静态文件的 HTTP 回调函数
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)

	//静态文件目录 mux路由遵循最长匹配原则 /index.html
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler)) //去掉/index.html中的/ 然后拼接成 ./webroot/index.html

	//启动TCP监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return err
	}

	//创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	//赋值单例
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	//开启协程运行http.Server 这样就启动了服务端
	go httpServer.Serve(listener)
	return
}
