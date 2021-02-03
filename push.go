package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

const(
	PushCategorySign 	= 1
	PushCategorySuccess = 2

	PushStatusWait		= 1
	PushStatusRetry 	= 2
	PushStatusOk		= 3
	PushStatusFiled 	= 4



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
	Mutex 	sync.Mutex
	Rule 	Rule
}

func NewPush(rule Rule)*Push{
	push := new(Push)
	push.Rule = rule
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
	res,_ := redis.Int(redisDo("INCR",key))
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
	res,_ := redis.String(redisDo("get",key))
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
	res,err := redisDo("set",redis.Args{}.Add(key).Add(pushStr)...)
	zlib.MyPrint("addOnePush rs : ",res,err)

	pushKey := push.getRedisKeyPushStatus()
	res,err = redisDo("zadd",redis.Args{}.Add(pushKey).Add(1).Add(id)...)
	zlib.MyPrint("add push status : ",res,err)

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
		strconv.Itoa(pushElement.Times) + separation

	return str
}

func (push *Push)   delAll(){
	key := push.getRedisPrefixKey()
	redisDo("del",key)
}

func (push *Push)   delOneRule(){
	log.Debug(" push delOneRule : ",push.getRedisCatePrefixKey())
	push.delAllPush()
	push.delAllStatus()
}

func  (push *Push)  delAllPush( ){
	prefix := push.getRedisKeyPushPrefix()
	res,_ := redis.Strings( redisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		log.Notice(" GroupElement by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(redisDo("del",v))
		zlib.MyPrint("del group element v :",res)
	}
}

func  (push *Push)  delAllStatus( ){
	key := push.getRedisKeyPushStatus()
	res,_ := redis.Strings( redisDo("del",key ))
	log.Debug("delAllStatus :",res)
}

func  (push *Push)  delOneStatus( pushId int){
	key := push.getRedisKeyPushStatus()
	res,_ :=  redisDo("ZREM",redis.Args{}.Add(key).Add(pushId) )
	zlib.MyPrint(" delOneRuleOneResult res",res)
}

func  (push *Push)  delOnePush( id int){
	key := push.getRedisKeyPush(id)
	res,_ := redis.Strings( redisDo("del",key ))
	zlib.MyPrint(" delOnePush res",res)

	push.delOneStatus(id)
}
func  (push *Push) hook(id int,status int){
	element := push.getById(id)
	if element.Category == PushCategorySign{

	}else{

	}
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
func  (push *Push)pushAndUpInfo(element PushElement,status int){
	sendHttpRs := true
	if sendHttpRs{
		push.delOnePush(element.Id)
	}else{
		element.Status = status
		element.UTime = zlib.GetNowTimeSecondToInt()
		element.Times = element.Times + 1
		key := push.getRedisKeyPush(element.Id)
		pushStr := push.pushStructToStr(element)
		res,err := redisDo("set",redis.Args{}.Add(key).Add(pushStr)...)
		zlib.MyPrint(res,err)
	}
}

func  (push *Push)  checkStatus(){
	log.Info("start one rule checkStatus : ")
	key := push.getRedisKeyPushStatus()

	inc := 1
	for {
		log.Info("loop inc : ",inc )
		if inc >= 2147483647{
			inc = 0
		}
		inc++

		push.checkOneByStatus(key,PushStatusWait)
		push.checkOneByStatus(key,PushStatusRetry)

	}

}

func  (push *Push)  checkOneByStatus(key string,status int){
	res,err := redis.IntMap(  redisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(key).Add(status).Add(status)...))
	log.Info("sign timeout group element total : ",len(res))
	if err != nil{
		log.Error("redis keys err :",err.Error())
		return
	}

	if len(res) == 0{
		log.Notice(" empty , no need process")
		mySleepSecond(1)
	}
	for _,id := range res{
		push.hook(id,status)
	}
}