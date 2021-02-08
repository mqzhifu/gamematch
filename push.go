package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

const(
	PushCategorySignTimeout 	= 1
	PushCategorySuccess 		= 2
	PushCategorySuccessTimeout	= 3

	PushStatusWait		= 1
	PushStatusRetry 	= 2
	//PushStatusOk		= 3
	//PushStatusFiled 	= 4



)
//匹配成功后，会推送给3方，
type PushElement struct {
	Id  		int
	ATime 		int
	UTime   	int		//最后更新的时间
	LinkId		int
	Status  	int 	//状态：1未推送2推送失败，等待重试3推送成功4推送失败，不再重试
	Times  		int		//已推送次数
	Category	int		//1:报名超时 2成功结果超时
	Payload 	string	//自定义的载体
}

type Push struct {
	Mutex 		sync.Mutex
	Rule 		Rule
	//queueSign 	*QueueSign
	QueueSuccess	*QueueSuccess
}

func NewPush(rule Rule)*Push{
	push := new(Push)
	push.Rule = rule


	//push.queueSign = NewQueueSign(rule)
	//这里有个问题，循环 new
	push.QueueSuccess = NewQueueSuccess(rule,nil)
	return push
}

var PushRetryPeriod = []int{10,30,60,600}

//成功类的整个：大前缀
func (push *Push)  getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "push"
}
//不同的匹配池(规则)，要有不同的KEY
func (push *Push)  getRedisCatePrefixKey(  )string{
	return push.getRedisPrefixKey() + redisSeparation + push.Rule.CategoryKey
}

func (push *Push) GetPushIncId( )  int{
	key := push.getRedisPushIncKey()
	res,_ := redis.Int(myredis.RedisDo("INCR",key))
	return res
}

func (push *Push) getRedisKeyPushStatus()string{
	return push.getRedisCatePrefixKey() + redisSeparation + "push_status"
}

func (push *Push) getRedisKeyPushPrefix()string{
	return push.getRedisCatePrefixKey() + redisSeparation + "push"
}

func (push *Push) getRedisKeyPush( id int)string{
	return push.getRedisKeyPushPrefix() + redisSeparation + "push" + redisSeparation + strconv.Itoa(id)
}
func (push *Push)  getRedisPushIncKey()string{
	return push.getRedisCatePrefixKey()  + redisSeparation + "inc_id"
}

func (push *Push) getById (id int) (element PushElement) {
	key := push.getRedisKeyPush(id)
	res,_ := redis.String(myredis.RedisDo("get",key))
	if res == ""{
		return element
	}

	element = push.pushStrToStruct(res)
	return element
}

func  (push *Push) addOnePush (linkId int,category int ,payload string) int {
	id :=  push.GetPushIncId()
	key := push.getRedisKeyPush(id)
	pushElement := PushElement{
		Id : id,
		ATime: zlib.GetNowTimeSecondToInt(),
		Status: 1,
		UTime: zlib.GetNowTimeSecondToInt(),
		Times:0,
		LinkId: linkId,
		Category : category,
		Payload : payload,
	}
	pushStr := push.pushStructToStr(pushElement)
	res,err := myredis.RedisDo("set",redis.Args{}.Add(key).Add(pushStr)...)
	mylog.Info("addOnePush rs : ",res,err)

	pushKey := push.getRedisKeyPushStatus()
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(pushKey).Add(PushStatusWait).Add(id)...)
	mylog.Info("add push status : ",res,err)

	return id
}

func (push *Push)  pushStrToStruct(redisStr string )PushElement{
	strArr := strings.Split(redisStr,separation)
	result := PushElement{
		Id:zlib.Atoi(strArr[0]),
		LinkId 	:	zlib.Atoi(strArr[1]),
		ATime 	:	zlib.Atoi(strArr[2]),
		Status : zlib.Atoi(strArr[3]),
		UTime : zlib.Atoi(strArr[4]),
		Times :  zlib.Atoi(strArr[5]),
		Category: zlib.Atoi(strArr[6]),
		Payload :  strArr[7],
	}
	return result
}

func (push *Push) pushStructToStr(pushElement PushElement)string{
	str :=
		strconv.Itoa(pushElement.Id) + separation +
		strconv.Itoa(pushElement.LinkId) + separation +
		strconv.Itoa(pushElement.ATime) + separation +
		strconv.Itoa(pushElement.Status) + separation +
		strconv.Itoa(pushElement.UTime) + separation +
		strconv.Itoa(pushElement.Times) + separation +
		strconv.Itoa(pushElement.Category) + separation +
			pushElement.Payload + separation

	return str
}

func (push *Push)   delAll(){
	key := push.getRedisPrefixKey()
	myredis.RedisDo("del",key)
}

func (push *Push)   delOneRule(){
	mylog.Debug(" push delOneRule : ",push.getRedisCatePrefixKey())
	push.delAllPush()
	push.delAllStatus()
}

func  (push *Push)  delAllPush( ){
	prefix := push.getRedisKeyPushPrefix()
	res,_ := redis.Strings( myredis.RedisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		mylog.Notice(" GroupElement by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(myredis.RedisDo("del",v))
		zlib.MyPrint("del group element v :",res)
	}
}

func  (push *Push)  delAllStatus( ){
	key := push.getRedisKeyPushStatus()
	res,_ := redis.Strings( myredis.RedisDo("del",key ))
	mylog.Debug("delAllStatus :",res)
}

func  (push *Push)  delOneStatus( pushId int){
	key := push.getRedisKeyPushStatus()
	res,_ :=  myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(pushId) )
	zlib.MyPrint(" delOneRuleOneResult res",res)
}

func  (push *Push)  delOnePush( id int){
	key := push.getRedisKeyPush(id)
	res,_ := redis.Strings( myredis.RedisDo("del",key ))
	zlib.MyPrint(" delOnePush res",res)

	push.delOneStatus(id)
}
func  (push *Push) hook(id int,status int){
	mylog.Info(" action hook")

	element := push.getById(id)
	if status == PushStatusWait{
		push.pushAndUpInfo(element,PushStatusRetry)
	}else{
		if element.Times >= len(PushRetryPeriod) {
			push.delOnePush(id)
		}else{
			time := PushRetryPeriod[element.Times]

			d := zlib.GetNowTimeSecondToInt() - element.UTime
			if d >= time{
				push.pushAndUpInfo(element,PushStatusRetry)
			}
		}
	}
}
func  (push *Push)pushAndUpInfo(element PushElement,upStatus int){
	mylog.Debug("pushAndUpInfo",element,upStatus)

	//zlib.ExitPrint(element.Category , PushCategorySignTimeout)
	var httpRs zlib.ResponseMsgST
	var err error
	//if element.Category == PushCategorySignTimeout{//测试使用
	//	return
	//}
	if element.Category == PushCategorySignTimeout ||  element.Category == PushCategorySuccessTimeout {
		payload := strings.Replace(element.Payload,PayloadSeparation,separation,-1)
		myGroup := GroupStrToStruct(payload)
		httpRs,err = myservice.HttpPost("gameroom","v1/match/error",myGroup)
	}else if element.Category == PushCategorySuccess{
		payload := strings.Replace(element.Payload,PayloadSeparation,separation,-1)
		thisResult := push.QueueSuccess.strToStruct (payload)

		aaa:= push.QueueSuccess.GetResultById(thisResult.Id,1,0)
		httpRs ,err = myservice.HttpPost("","v1/match/succ",aaa)
	}

	if err != nil{
		msg := myerr.MakeOneStringReplace(err.Error())
		myerr.NewErrorCodeReplace(911,msg)
		push.upRetryPushInfo(element)
		return
	}

	if(httpRs.Code == 0){
		push.delOnePush(element.Id)
		return
	}

	if element.Category == PushCategorySignTimeout ||  element.Category == PushCategorySuccessTimeout {
		if httpRs.Code == 116 || httpRs.Code ==  119{
			//ErrGroupStatus   119 队伍当前不可操作（不要重试）
			//ErrServerError 102 服务器内部错误
			//ErrMarshalError 101 json解析错误
			//ErrGroupLock 117 组队数据读写中
			//ErrGroupExpire 116 组队过期或者不存在 （不要重试）
			push.delOnePush(element.Id)
			return
		}
	}else{

	}

}
//失败且需要重试的PUSH-ELEMENT
func  (push *Push)  upRetryPushInfo(element PushElement ){
	element.Status = PushStatusRetry
	element.UTime = zlib.GetNowTimeSecondToInt()
	element.Times = element.Times + 1
	key := push.getRedisKeyPush(element.Id)
	pushStr := push.pushStructToStr(element)
	res,err := myredis.RedisDo("set",redis.Args{}.Add(key).Add(pushStr)...)

	statuskey := push.getRedisKeyPushStatus()

	push.delOneStatus(element.Id)
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(statuskey).Add(PushStatusRetry).Add(element.Id)...)
	mylog.Info("add push status : ",res,err)


}
func  (push *Push)  checkStatus(){
	mylog.Info("start one rule checkStatus : ")
	key := push.getRedisKeyPushStatus()

	inc := 1
	for {
		mylog.Info("loop inc : ",inc )
		if inc >= 2147483647{
			inc = 0
		}
		inc++

		push.checkOneByStatus(key,PushStatusWait)
		push.checkOneByStatus(key,PushStatusRetry)

		mySleepSecond(1)
	}

}

func  (push *Push)  checkOneByStatus(key string,status int){
	res,err := redis.Ints(  myredis.RedisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(key).Add(status).Add(status)...))
	mylog.Info("sign timeout group element total : ",len(res))
	if err != nil{
		mylog.Error("redis keys err :",err.Error())
		return
	}

	if len(res) == 0{
		mylog.Notice(" empty , no need process")
	}
	for _,id := range res{
		push.hook(id,status)
	}
}