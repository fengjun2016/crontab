package master

import (
	"context"
	"encoding/json"
	"github.com/fengjun2016/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"log"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	//单例
	G_jobMgr *JobMgr
)

//初始化管理器
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)

	//初始化配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     //节点集群列表配置
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, //连接超时
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	//得到KV和Lease子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	//赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}

//保存任务的方法
func (jM *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	//把任务保存到/cron/jobs/任务名 -> json
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)

	//etcd的保存key
	jobKey = common.JOB_SAVE_DIR + job.Name

	//任务信息json
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	//保存到etcd
	if putResp, err = jM.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}

	//如果是更新，那么返回旧值
	if putResp.PrevKv != nil {
		//对旧值做一个反序列化操作
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

//删除任务
func (jM *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	//把任务从etcd中删除掉
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)
	//etcd中保存的key
	jobKey = common.JOB_SAVE_DIR + name

	if delResp, err = jM.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	//查看要删除的旧值
	if delResp.PrevKvs != nil {
		//对旧值做一个反序列化操作
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

//获取任务列表
func (jM *JobMgr) ListJobs() (jobLists []*common.Job, err error) {
	var (
		dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		job     *common.Job
	)

	//获取目录key
	dirKey = common.JOB_SAVE_DIR

	//获取任务列表
	if getResp, err = jM.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		return
	}

	//初始化数组空间 len(jobListe) == 0
	jobLists = make([]*common.Job, 0)

	//遍历所有任务，进行反序列化
	for _, kvPair = range getResp.Kvs {
		log.Print(string(kvPair.Value))
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			log.Printf("unmarshal err : %v", err)
			err = nil
			continue
		}
		jobLists = append(jobLists, job)
	}

	log.Printf("jm jobs list: %v", getResp.Kvs)
	return
}

//杀死任务
func (jM *JobMgr) KillJob(name string) (err error) {
	//做法思想就是: 更新一下key=/cron/killer/任务名
	var (
		killKey        string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)

	//通知worker杀死对应任务
	killKey = common.JOB_KILL_DIR + name

	//让worker监听到一次put操作即可，创建一个租约让其稍后自动过期即可
	if leaseGrantResp, err = jM.lease.Grant(context.TODO(), 1); err != nil {
		return
	}

	//租约ID
	leaseId = leaseGrantResp.ID

	//设置killer标记
	if _, err = jM.kv.Put(context.TODO(), killKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return
}
