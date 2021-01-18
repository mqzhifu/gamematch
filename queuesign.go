package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

type QueueSignElement struct {
	PlayerId 	int
	ATime		int		//报名时间
	Timeout		int		//多少秒后无人来取，即超时，更新用户状态，删除数据
	GroupId		int
	GroupPerson	int		//小组人数
	GroupWeight	float32	//小组权重
	MatchTimes	int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃
}

type QueueSign struct {
	//多协程同时，计算匹配结果时，要加锁
	Mutex sync.Mutex
}

type QueueSignTimeoutElement struct {
	PlayerId	int
	ATime		int
	Flag 		int		//1超时 2状态变更
	Memo 		string	//备注
	GroupId		int

}

func QueueSignTimeoutElementAddOne(){

}

func NewQueueSign()*QueueSign{
	queueSign := new(QueueSign)
	return queueSign
}

func (queueSign *QueueSign) getRedisKey(rule Rule)string{
	return redisPrefix + redisSeparation + "sign"+redisSeparation + rule.CategoryKey
}

func (queueSign *QueueSign) getRedisGroupIncKey()string{
	return redisPrefix + redisSeparation + "signGroup" + redisSeparation + "inc"
}

func (queueSign *QueueSign) GetGroupIncId( ) (int){
	key := queueSign.getRedisGroupIncKey()
	res,_ := redis.Int(redisDo("INCR",key))
	return res
}

func (queueSign *QueueSign) PushOne(queueSignElement QueueSignElement,rule Rule){
	key := queueSign.getRedisKey(rule)
	queueSignElement.ATime = getNowTime()
	content := queueSign.structToStr(queueSignElement)
	//zlib.MyPrint("push one :",content)
	redisDo("zadd",redis.Args{}.Add(key).Add(queueSignElement.GroupWeight).Add(content)...)
}

func  (queueSign *QueueSign)structToStr(queueSignElement QueueSignElement)string{
	//GroupWeight	float32	//小组权重
	//MatchTimes	int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃

	GroupWeight := zlib.FloatToString(queueSignElement.GroupWeight,3)
	content := strconv.Itoa( queueSignElement.PlayerId) + separation +
		strconv.Itoa(  queueSignElement.ATime )+ separation +
		strconv.Itoa(  queueSignElement.Timeout) + separation +
		strconv.Itoa(queueSignElement.GroupId) +separation +
		strconv.Itoa(queueSignElement.GroupPerson) + separation +
		GroupWeight + separation +
		strconv.Itoa(queueSignElement.MatchTimes) + separation

	return content
}

func  (queueSign *QueueSign)strToStruct(redisStr string)QueueSignElement{
	strArr := strings.Split(redisStr,separation)
	element := QueueSignElement{
		PlayerId 	:	zlib.Atoi(strArr[0]),
		ATime 		:	zlib.Atoi(strArr[1]),
		Timeout 	:	zlib.Atoi(strArr[2]),
		GroupId 	:	zlib.Atoi(strArr[3]),
		GroupPerson :	zlib.Atoi(strArr[4]),
		GroupWeight	:	zlib.StringToFloat(strArr[5]),
		MatchTimes :	zlib.Atoi(strArr[6]),
	}
	return element
}

func (queueSign *QueueSign) PopOne(rule Rule)(queueSignElement QueueSignElement, isEmpty bool){
	key := queueSign.getRedisKey(rule)
	res,err := redis.String(redisDo("rpop",key))
	if err != nil {
		zlib.ExitPrint("pop one err",err.Error())
	}
	if res == ""{
		return queueSignElement,true
	}

	queueSignElement = queueSign.strToStruct(res)
	return queueSignElement,false
}
//以组为单位，一次弹出多个玩家，也可能是一个
func (queueSign *QueueSign) PopOneByGroup(rule Rule)(queueSignElements []QueueSignElement, isEmpty bool,firstGroupPerson int,firstGroupId int){
	key := queueSign.getRedisKey(rule)
	//queueSignElementRs := make(map[int]QueueSignElement)
	i := 0
	GroupPerson := 1
	grouopId := 0
	for{
		res,err := redis.String(redisDo("rpop",key))
		if err != nil {
			zlib.ExitPrint("pop one err",err.Error())
		}
		if res == ""{
			if i == 0{
				return queueSignElements,true,0,0
			}else{
				return queueSignElements,true,GroupPerson,grouopId
			}

		}
		queueSignElements[i] = queueSign.strToStruct(res)
		if i == 0{//以第一个玩家的组人数，做为依据
			GroupPerson = queueSignElements[i].GroupPerson
			grouopId = queueSignElements[i].GroupId
		}
		if i >= GroupPerson - 1{
			break
		}
	}
	return queueSignElements,true,GroupPerson,grouopId
}

func (queueSign *QueueSign) PopSome(rule Rule,start int ,stop int)(data map[string]QueueSignElement){
	if start > stop{
		zlib.ExitPrint("start > stop")
	}
	key := queueSign.getRedisKey(rule)
	res,err := redis.StringMap( redisDo("LRANGE",redis.Args{}.Add(key).Add(start).Add(stop)...  ) )
	if err != nil{
		zlib.ExitPrint("PopSome larange err",err.Error())
	}

	for k,v := range  res{
		data[k] = queueSign.strToStruct(v)
	}

	return data
}

func   (queueSign *QueueSign) length(key string)int{
	res,_ := redis.Int(redisDo("Llen",key))
	return res
}

func (queueSign *QueueSign) GetList(){

}

func (queueSign *QueueSign) CheckTimeout(element QueueSignElement )bool{
	now := getNowTime()
	if now > element.Timeout{
		return true
	}
	return false
}

func (queueSign *QueueSign) ZREVRANGEBYSCORETOTAL(rule Rule )(queueSignElements map[int]QueueSignElement){
	key := queueSign.getRedisKey(rule)
	res,err := redis.StringMap(redisDo("ZREVRANGEBYSCORE", redis.Args{}.Add(key).Add("+inf").Add("-inf").Add("WITHSCORES")...))
	if err != nil{
		zlib.MyPrint("ZREVRANGEBYSCORETOTAL err",err.Error())
	}
	if len(res) <= 0{
		return queueSignElements
	}

	return queueSignElements
	//for k,v := range res{
	//	queueSignElement := queueSign.strToStruct(v)
	//	queueSignElements[k] = queueSignElement
	//}
	//
	//return res
}

func (queueSign *QueueSign) ZREVRANGEBYSCORE(rule Rule,scoreMax int,scoreMin int){
	key := queueSign.getRedisKey(rule)
	scoreMaxStr := strconv.Itoa(scoreMax) + ".9999"
	redisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(key).Add(scoreMaxStr).Add(scoreMin).Add("WITHSCORES"))
}

func (queueSign *QueueSign) ZCOUNTTOTAL(rule Rule)int{
	key := queueSign.getRedisKey(rule)
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//删除所有已报名的玩家数据
func  (queueSign *QueueSign)  delAllPlayers(rule Rule){
	key := queueSign.getRedisKey(rule)
	redisDo("del",key)
}


func (queueSign *QueueSign) ZCOUNT(rule Rule,scoreMin int,scoreMax float32)int{
	key := queueSign.getRedisKey(rule)
	//scoreMaxStr := strconv.Itoa(scoreMax) + ".9999"
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add(scoreMin).Add(scoreMax)...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}