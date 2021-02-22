package gamematch

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"sync"
	"zlib"
)

//推送给3方，支持重试
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
	Log 		*zlib.Log
}

func NewPush(rule Rule)*Push{
	push := new(Push)
	push.Rule = rule


	//push.queueSign = NewQueueSign(rule)
	//这里有个问题，循环 new
	push.QueueSuccess = NewQueueSuccess(rule,push)
	push.Log = getRuleModuleLogInc(rule.CategoryKey,"push")
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
func (push *Push)  getRedisPushIncKey()string{
	return push.getRedisCatePrefixKey()  + redisSeparation + "inc_id"
}
func (push *Push) GetPushIncId( )  int{
	key := push.getRedisPushIncKey()
	res,_ := redis.Int(myredis.RedisDo("INCR",key))
	return res
}

func (push *Push) getRedisKeyPushStatus()string{
	return push.getRedisCatePrefixKey() + redisSeparation + "status"
}

//func (push *Push) getRedisKeyPushPrefix()string{
//	return push.getRedisCatePrefixKey() + redisSeparation + "push"
//}

func (push *Push) getRedisKeyPush( id int)string{
	return push.getRedisCatePrefixKey()   + redisSeparation + strconv.Itoa(id)
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
	mylog.Debug("addOnePush" , linkId,category,payload)
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
	mylog.Info("addOnePush status : ",res,err)

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
	prefix := push.getRedisCatePrefixKey()
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
	res,err :=  myredis.RedisDo("ZREM",redis.Args{}.Add(key).Add(pushId)... )
	mylog.Debug(" delOne PushStatus index res",res,err)
	push.Log.Info(" delOne PushStatus index res",res,err)
}

//失败且需要重试的PUSH-ELEMENT
func  (push *Push)  upRetryPushInfo(element PushElement ){
	element.Status = PushStatusRetry
	element.UTime = zlib.GetNowTimeSecondToInt()
	element.Times = element.Times + 1
	key := push.getRedisKeyPush(element.Id)
	pushStr := push.pushStructToStr(element)
	res,err := myredis.RedisDo("set",redis.Args{}.Add(key).Add(pushStr)...)

	push.Log.Info("upRetryPushElementInfo , ", element)
	//这里有个麻烦点，元素信息 和 索引信息，是分开放的，元素的变更比较简单，索引是一个集合，改起来有点麻烦
	//那就直接先删了，再重新添加一条
	statuskey := push.getRedisKeyPushStatus()
	push.Log.Info("del pushStatus index ,pushId : ",element.Id)
	push.delOneStatus(element.Id)
	res,err = myredis.RedisDo("zadd",redis.Args{}.Add(statuskey).Add(PushStatusRetry).Add(element.Id)...)

	push.Log.Info("add  new pushStatus index : ",res,err)
	mylog.Info("add  new pushStatus index : ",res,err)
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
		push.Log.Info("loop inc ",inc)

		push.checkOneByStatus(key,PushStatusWait)
		push.checkOneByStatus(key,PushStatusRetry)

		mySleepSecond(1, "  push checkStatus")
	}

}

func  (push *Push)  checkOneByStatus(key string,status int){
	mylog.Info("checkOneByStatus :",status)
	push.Log.Info("checkOneByStatus :",status)
	res,err := redis.Ints(  myredis.RedisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(key).Add(status).Add(status)...))
	push.Log.Info("need process element total : ",len(res))
	if err != nil{
		mylog.Error("redis keys err :",err.Error())
		push.Log.Error("redis keys err :",err.Error())
		return
	}

	if len(res) == 0{
		mylog.Notice(" empty , no need process")
		push.Log.Notice(" empty , no need process")
	}
	for _,id := range res{
		push.hook(id,status)
	}
}

func  (push *Push) hook(id int,status int){
	//mylog.Info(" action hook , push id : " ,id ," status : ",status)
	push.Log.Info(" action hook , push id : " ,id ," status : ",status)
	element := push.getById(id)
	//fmt.Printf("%+v", element)
	if status == PushStatusWait{
		push.Log.Info("element first push")
		push.pushAndUpInfo(element,PushStatusRetry)
	}else{
		push.Log.Info("element retry")
		if element.Times >= len(PushRetryPeriod) {
			mylog.Notice(" push retry time > maxRetryTime")
			push.delOnePush(id)
		}else{
			time := PushRetryPeriod[element.Times]
			push.Log.Info("retry rule : ",PushRetryPeriod," this time : ",time )
			d := zlib.GetNowTimeSecondToInt() - element.UTime
			//mylog.Info("this time : ",time,"now :",zlib.GetNowTimeSecondToInt() , " - element.UTime ",element.UTime , " = ",d)
			push.Log.Info("this time : ",time,"now :",zlib.GetNowTimeSecondToInt() , " - element.UTime ",element.UTime , " = ",d)
			if d >= time{
				push.pushAndUpInfo(element,PushStatusRetry)
			}else{
				push.Log.Notice("The time is too short to trigger the Push!!! ")
			}
		}
	}
}
func getAnyType()(a interface{}){
	return a
}
func  (push *Push)pushAndUpInfo(element PushElement,upStatus int){
	mylog.Debug("pushAndUpInfo",element,upStatus)
	push.Log.Info("pushAndUpInfo",element," upStatus: ",upStatus)
	//zlib.ExitPrint(element.Category)
	var httpRs zlib.ResponseMsgST
	var err error
	//if element.Category == PushCategorySignTimeout{//测试使用
	//	return
	//}

	thirdMethodUri := ""
	postData := getAnyType()
	//thirdMethodPostData := interface{}
	if element.Category == PushCategorySignTimeout  {
		push.Log.Debug("element.Category == PushCategorySignTimeout")
		payload := strings.Replace(element.Payload,PayloadSeparation,separation,-1)
		postData = GroupStrToStruct(payload)

		thirdMethodUri = "v1/match/error"
		//httpRs,err = myservice.HttpPost(SERVICE_MSG_SERVER,"v1/match/error",myGroup)
	}else if element.Category == PushCategorySuccessTimeout {
		push.Log.Debug("element.Category == PushCategorySuccessTimeout")
		payload := strings.Replace(element.Payload,PayloadSeparation,separation,-1)
		//zlib.MyPrint(payload)
		postData = push.QueueSuccess.strToStruct(payload)
		thirdMethodUri = "v1/match/error"
		//httpRs,err = myservice.HttpPost(SERVICE_MSG_SERVER,"v1/match/error",myResult)
	}else if element.Category == PushCategorySuccess{
		push.Log.Debug("element.Category == PushCategorySuccess")
		payload := strings.Replace(element.Payload,PayloadSeparation,separation,-1)
		thisResult := push.QueueSuccess.strToStruct (payload)
		postData = push.QueueSuccess.GetResultById(thisResult.Id,1,0)
		//fmt.Printf("postData : %+v",postData)
		//zlib.ExitPrint(payload)
		thirdMethodUri = "v1/match/succ"
		//httpRs ,err = myservice.HttpPost(SERVICE_MSG_SERVER,"v1/match/succ",postData)
	}else{
		mylog.Error("element.Category error.")
		push.Log.Error("element.Category error.")
		return
	}
	push.Log.Debug("push third service ,  uri : ",thirdMethodUri, " , postData : ",postData)
	httpRs,err = myservice.HttpPost(SERVICE_MSG_SERVER,thirdMethodUri,postData)
	push.Log.Debug("push third service , httpRs : ",httpRs , " err : ",err)

	if err != nil{
		push.upRetryPushInfo(element)

		msg := myerr.MakeOneStringReplace(err.Error())
		myerr.NewErrorCodeReplace(911,msg)
		push.Log.Error("push third service ",err.Error())
		return
	}

	if(httpRs.Code == 0){// 0 即是200 ，推送成功~
		push.delOnePush(element.Id)
		if element.Category == PushCategorySuccess || element.Category == PushCategorySuccessTimeout{
			push.Log.Info("delOneResult")
			push.QueueSuccess.delOneResult(element.LinkId,1,1,1,1)
		}
		return
	}

	if element.Category == PushCategorySignTimeout   {
		if httpRs.Code == 116 || httpRs.Code ==  119{
			push.delOnePush(element.Id)
		}else{
			push.upRetryPushInfo(element)

			httpRsJsonStr,_ := json.Marshal(httpRs)
			msg := myerr.MakeOneStringReplace(string(httpRsJsonStr))
			myerr.NewErrorCodeReplace(700,msg)
			return
		}
	}else if  element.Category == PushCategorySuccessTimeout{
		push.delOnePush(element.Id)
		push.Log.Info("delOneResult")
		push.QueueSuccess.delOneResult(element.LinkId,1,1,1,1)
	}else if element.Category == PushCategorySuccess{
		if httpRs.Code == 108 || httpRs.Code == 102{
			push.upRetryPushInfo(element)

			httpRsJsonStr,_ := json.Marshal(httpRs)
			msg := myerr.MakeOneStringReplace(string(httpRsJsonStr))
			myerr.NewErrorCodeReplace(700,msg)
			return
		}else{
			push.delOnePush(element.Id)
			push.Log.Info("delOneResult")
			push.QueueSuccess.delOneResult(element.LinkId,1,1,1,1)
			return
		}
	}else{
		mylog.Error("pushAndUpInfo element.Category not fuond!!!")
		push.Log.Error("pushAndUpInfo element.Category not fuond!!!")
	}
}

func  (push *Push)  delOnePush( id int){
	//mylog.Info("delOnePush",id)
	key := push.getRedisKeyPush(id)
	res,err :=   myredis.RedisDo("del",redis.Args{}.Add(key)... )
	mylog.Debug(" delOnePush (",id,")",res,err)
	push.Log.Info(" delOnePush (",id,")",res,err)

	push.delOneStatus(id)
}