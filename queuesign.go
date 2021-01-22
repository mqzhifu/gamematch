package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
	"sync"
)

type QueueSign struct {
	//计算匹配结果时，要加锁
	//1、阻塞住报名，2、阻塞其它匹配的协程
	Mutex 			sync.Mutex
}

func NewQueueSign()*QueueSign{
	queueSign := new(QueueSign)
	return queueSign
}

type QueueSignGroupElement struct {
	Players 	[]Player
	ATime		int		//报名时间
	Timeout		int		//多少秒后无人来取，即超时，更新用户状态，删除数据
	GroupId		int
	GroupPerson	int		//小组人数
	GroupWeight	float32	//小组权重
	MatchTimes	int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃
}

//type QueueSignTimeoutElement struct {
//	PlayerId	int
//	ATime		int
//	Flag 		int		//1超时 2状态变更
//	Memo 		string	//备注
//	GroupId		int
//}

//报名类的整个：大前缀
func (queueSign *QueueSign) getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "sign"
}
//不同的匹配池(规则)，要有不同的KEY
func (queueSign *QueueSign) getRedisCatePrefixKey( rule Rule )string{
	return queueSign.getRedisPrefixKey()+redisSeparation + rule.CategoryKey
}
//有序集合：组索引，所有报名的组，都在这个集合中，weight => groupId
func (queueSign *QueueSign) getRedisKeyGroupWeight(rule Rule)string{
	return queueSign.getRedisCatePrefixKey(rule) + redisSeparation + "group_weight"
}
//有序集合：组的人数索引，每个规则的池，允许N人成组，其中，每个组里有多少个人，就是这个索引
func (queueSign *QueueSign) getRedisKeyGroupPersonIndexPrefix(rule Rule)string{
	return queueSign.getRedisCatePrefixKey(rule) + redisSeparation + "group_person"
}
//组人数=>groupId
func (queueSign *QueueSign) getRedisKeyGroupPersonIndex(rule Rule,personNum int)string{
	return queueSign.getRedisKeyGroupPersonIndexPrefix(rule) + redisSeparation + strconv.Itoa(personNum)
}
//最简单的string：一个组的详细信息
func (queueSign *QueueSign) getRedisKeyGroupElementPrefix(rule Rule )string{
	return queueSign.getRedisCatePrefixKey(rule) + redisSeparation + "element"
}
func (queueSign *QueueSign) getRedisKeyGroupElement(rule Rule,groupId int)string{
	return queueSign.getRedisKeyGroupElementPrefix(rule) + redisSeparation + strconv.Itoa(groupId)
}
//有序集合：一个小组，包含的所有玩家ID
func (queueSign *QueueSign) getRedisKeyGroupPlayer(rule Rule)string{
	return queueSign.getRedisCatePrefixKey(rule) + redisSeparation + "player"
}
//组自增ID，因为匹配最小单位是基于组，而不是基于一个人，组ID就得做到全局唯一，很重要
func (queueSign *QueueSign) getRedisGroupIncKey(rule Rule)string{
	return queueSign.getRedisCatePrefixKey(rule)  + redisSeparation + "group_inc_id"
}
//组的超时索引,有序集合
func (queueSign *QueueSign) getRedisKeyGroupTimeout(rule Rule)string{
	return queueSign.getRedisCatePrefixKey(rule)  + redisSeparation + "timeout"
}
//===============================以上是 redis key 相关============================

//获取当前所有，已报名的，所有组，总数
func (queueSign *QueueSign) getGroupsTotal( rule Rule )  int{
	key := queueSign.getRedisKeyGroupWeight(rule)
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//获取当前所有，已报名的，组，总数
func (queueSign *QueueSign) getGroupPersonTotal( rule Rule ,personNum int )  int{
	key := queueSign.getRedisKeyGroupPersonIndex(rule,personNum)
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//获取当前所有，已报名的，玩家，总数
func (queueSign *QueueSign) getPlayersTotal( rule Rule )  int{
	////目前，小组支持最大人数为：5
	//var total int
	//for i :=- 1 ;i <=5 ; i++{
	//	groupPersonTotal := queueSign.getGroupPersonTotal(rule,i)
	//	total += i * groupPersonTotal
	//}
	key := queueSign.getRedisKeyGroupPlayer(rule)
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}
//============================以上是 各种获取总数===============================

//获取并生成一个自增GROUP-ID
func (queueSign *QueueSign) GetGroupIncId( rule Rule )  int{
	key := queueSign.getRedisGroupIncKey(rule)
	res,_ := redis.Int(redisDo("INCR",key))
	return res
}
func (queueSign *QueueSign) getGroupElementById( rule Rule ,groupId int ) (queueSignGroupElement  QueueSignGroupElement){
	key := queueSign.getRedisKeyGroupElement(rule,groupId)
	res,_ := redis.String(redisDo("get",key))
	if res == ""{
		zlib.MyPrint(" getGroupElementById is empty!")
		return queueSignGroupElement
	}
	queueSignGroupElement = queueSign.strToStruct(res)
	return queueSignGroupElement
}

func (queueSign *QueueSign) getPlayerIdById( rule Rule ,playerId int )   int {
	key := queueSign.getRedisKeyGroupPlayer(rule)
	memberIndex,_ := redis.Int(redisDo("zrank",key,playerId))
	if memberIndex == 0{
		zlib.MyPrint(" getGroupElementById is empty!")
	}
	groupId,_ := redis.Int( redisDo("zrange",redis.Args{}.Add(key).Add(memberIndex).Add(memberIndex).Add("WITHSCORES")...))
	//queueSignGroupElement := queueSign.strToStruct(res)
	return groupId
}
//============================以上都是根据ID，获取一条===============================
func (queueSign *QueueSign) getGroupWeightRangeCnt(rule Rule,scoreMin int,scoreMax float32)int{
	key := queueSign.getRedisKeyGroupWeight(rule)
	res,err := redis.Int(redisDo("ZCOUNT",redis.Args{}.Add(key).Add(scoreMin).Add(scoreMax)...))
	if err !=nil{
		zlib.ExitPrint("ZCOUNT err",err.Error())
	}
	return res
}

func (queueSign *QueueSign) getGroupPersonRangeListAll(rule Rule,personNum int,limitOffset int ,limitCnt int)(groupIds map[int]int){
	key := queueSign.getRedisKeyGroupPersonIndex(rule,personNum)
	//这里有个问题，person=>groupId,person是重复的，如果带着分值一并返回，MAP会有重复值的情况
	res,err := redis.Ints(redisDo("ZREVRANGEBYSCORE", redis.Args{}.Add(key).Add("+inf").Add("-inf").Add("limit").Add(limitOffset).Add(limitCnt)...))
	if err != nil{
		zlib.MyPrint("getGroupPersonRangeList err",err.Error())
	}
	if len(res) <= 0{
		return groupIds
	}
	zlib.ExitPrint(res)
	inc := 0
	for _,v := range res{
		groupIds[inc]  = v
		inc++
	}
	return groupIds
}

func (queueSign *QueueSign) getGroupPersonRangeList(rule Rule,personNum int ,scoreMin int,scoreMax float32,limitOffset int ,limitCnt int)(groupIds map[int]int){
	key := queueSign.getRedisKeyGroupPersonIndex(rule,personNum)
	res,err := redis.IntMap(redisDo("ZREVRANGEBYSCORE", redis.Args{}.Add(key).Add(scoreMax).Add(scoreMin).Add("limit").Add(limitOffset).Add(limitCnt)...))
	if err != nil{
		zlib.MyPrint("getGroupPersonRangeList err",err.Error())
	}
	if len(res) <= 0{
		return groupIds
	}

	inc := 0
	for _,v := range res{
		groupIds[inc]  = v
		inc++
	}
	return groupIds
}

//============================以上都是范围性的取值===============================

//报名
func (queueSign *QueueSign) PushOneGroup(queueSignGroupElement QueueSignGroupElement,rule Rule){
	queueSignGroupElement.ATime = getNowTime()

	groupIndexKey := queueSign.getRedisKeyGroupWeight(rule)
	res,err := redisDo("zadd",redis.Args{}.Add(groupIndexKey).Add(queueSignGroupElement.GroupWeight).Add(queueSignGroupElement.GroupId)...)
	zlib.MyPrint("add GroupWeight rs ",res,err)

	groupPersonIndexKey := queueSign.getRedisKeyGroupPersonIndex(rule,queueSignGroupElement.GroupPerson)
	res,err = redisDo("zadd",redis.Args{}.Add(groupPersonIndexKey).Add(queueSignGroupElement.GroupWeight).Add(queueSignGroupElement.GroupId)...)
	zlib.MyPrint("add groupPersonIndex ( ",queueSignGroupElement.GroupPerson," ) rs ",res,err)

	groupTimeoutKey := queueSign.getRedisKeyGroupTimeout(rule)
	res,err =redisDo("zadd",redis.Args{}.Add(groupTimeoutKey).Add(queueSignGroupElement.Timeout).Add(queueSignGroupElement.GroupId)...)
	zlib.MyPrint("add groupTimeout rs ",res,err)

	groupElementRedisKey := queueSign.getRedisKeyGroupElement(rule,queueSignGroupElement.GroupId)
	content := queueSign.structToStr(queueSignGroupElement)

	groupPlayersKey := queueSign.getRedisKeyGroupPlayer(rule)
	for _,v := range queueSignGroupElement.Players{
		res,err = redisDo("zadd",redis.Args{}.Add(groupPlayersKey).Add(queueSignGroupElement.GroupId).Add(v.Id)...)
		zlib.MyPrint("add player rs ",res,err)
	}

	res,err = redisDo("set",redis.Args{}.Add(groupElementRedisKey).Add(content)...)
	zlib.MyPrint("add groupElement rs ",res,err)

}

func  (queueSign *QueueSign)structToStr(queueSignGroupElement QueueSignGroupElement)string{
	//GroupWeight	float32	//小组权重
	//MatchTimes	int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃

	playersIds := ""
	for _,v := range queueSignGroupElement.Players{
		playersIds +=  strconv.Itoa(v.Id) + IdsSeparation
	}
	playersIds = playersIds[0:len(playersIds)-1]
	GroupWeight := zlib.FloatToString(queueSignGroupElement.GroupWeight,3)
	content := playersIds + separation +
		strconv.Itoa(  queueSignGroupElement.ATime )+ separation +
		strconv.Itoa(  queueSignGroupElement.Timeout) + separation +
		strconv.Itoa(queueSignGroupElement.GroupId) +separation +
		strconv.Itoa(queueSignGroupElement.GroupPerson) + separation +
		GroupWeight + separation +
		strconv.Itoa(queueSignGroupElement.MatchTimes)

	return content
}

func  (queueSign *QueueSign)strToStruct(redisStr string)QueueSignGroupElement{
	//zlib.MyPrint("redis str : ",redisStr)
	strArr := strings.Split(redisStr,separation)
	//zlib.ExitPrint("redisStr",redisStr)
	//这条是错误的，这里有个特殊的情况，有序集合，返回列表的时候，加上<分数>参数，返回的<分数>，被加在最后一个元素的末尾，且以冒号分隔
	//arrLastIndex := len(strArr) - 1
	//lastArr := strings.Split(":",strArr[arrLastIndex])
	//lastElement := lastArr[0]
	//source := lastArr[1]

	playersArr := strings.Split(strArr[0],IdsSeparation)
	var players []Player
	for _,v := range playersArr{
		players = append(players,Player{Id : zlib.Atoi(v)})
	}
	//zlib.ExitPrint(players)
	element := QueueSignGroupElement{
		Players 	:	players,
		ATime 		:	zlib.Atoi(strArr[1]),
		Timeout 	:	zlib.Atoi(strArr[2]),
		GroupId 	:	zlib.Atoi(strArr[3]),
		GroupPerson :	zlib.Atoi(strArr[4]),
		GroupWeight	:	zlib.StringToFloat(strArr[5]),
		MatchTimes :	zlib.Atoi(strArr[6]),
	}
	return element
}
//func (queueSign *QueueSign) CheckTimeout(element QueueSignElement )bool{
//	now := getNowTime()
//	if now > element.Timeout{
//		return true
//	}
//	return false
//}
//
//func (queueSign *QueueSign) ZREVRANGEBYSCORE(rule Rule,scoreMax float32,scoreMin int)(queueSignElements map[int]QueueSignElement){
//	key := queueSign.getRedisKey(rule)
//	res,err := redis.StringMap(redisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(key).Add(scoreMax).Add(scoreMin).Add("WITHSCORES")...))
//	if err !=nil{
//		zlib.ExitPrint("ZCOUNT err",err.Error())
//	}
//	//zlib.ExitPrint(res)
//	if len(res) == 0{
//		return queueSignElements
//	}
//	queueSignElements = make(map[int]QueueSignElement)
//	i := 0
//	for k,_ := range res{
//		//zlib.MyPrint("ragne ",k,v)
//		queueSignElements[i] = queueSign.strToStruct(k)
//		i++
//	}
//	return queueSignElements
//
//}
//
//
//
//

//==============以下均是 删除操作======================================


//删除所有：池里的报名组、玩家、索引等-有点暴力，尽量不用
func  (queueSign *QueueSign)  delAll(){
	key := queueSign.getRedisPrefixKey()
	redisDo("del",key)
}
//删除一条规则的所有匹配信息
func  (queueSign *QueueSign)  delOneRuleAll(rule Rule){
	queueSign.delOneRuleALLGroupElement(rule)
	queueSign.delOneRuleALLGroupPersonIndex(rule)
	queueSign.delOneRuleAllGroupTimeout(rule)
	queueSign.delOneRuleALLPlayers(rule)
	queueSign.delOneRuleALLWeight(rule)
}
//====================================================

//删除一条规则的，所有玩家索引
func  (queueSign *QueueSign)  delOneRuleALLPlayers(rule Rule){
	key := queueSign.getRedisKeyGroupPlayer(rule)
	res,_ := redis.Int(redisDo("del",key))
	zlib.MyPrint("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，所有权重索引
func  (queueSign *QueueSign)  delOneRuleALLWeight(rule Rule){
	key := queueSign.getRedisKeyGroupWeight(rule)
	res,_ := redis.Int(redisDo("del",key))
	zlib.MyPrint("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，所有人数分组索引
func  (queueSign *QueueSign)  delOneRuleALLGroupPersonIndex(rule Rule){
	for i:=1 ; i <= RuleGroupPersonMax;i++{
		queueSign.delOneRuleOneGroupPersonIndex(rule,i)
	}
}

//删除一条规则的，所有分组详细信息
func  (queueSign *QueueSign)  delOneRuleALLGroupElement(rule Rule){
	prefix := queueSign.getRedisKeyGroupElementPrefix(rule)
	res,_ := redis.Strings( redisDo("keys",prefix + "*"  ))
	if len(res) == 0{
		zlib.MyPrint(" GroupElement by keys(*) : is empty")
		return
	}
	//zlib.ExitPrint(res,-200)
	for _,v := range res{
		res,_ := redis.Int(redisDo("del",v))
		zlib.MyPrint("del group element v :",res)
	}
}

//删除一条规则的，所有人组超时索引
func  (queueSign *QueueSign)  delOneRuleAllGroupTimeout(rule Rule){
	key := queueSign.getRedisKeyGroupTimeout(rule)
	res,_ := redis.Int(redisDo("del",key))
	zlib.MyPrint("delOneRuleALLPlayers : ",res)
}
//删除一条规则的，某一人数各类的，所有人数分组索引
func  (queueSign *QueueSign)  delOneRuleOneGroupPersonIndex(rule Rule,personNum int){
	key := queueSign.getRedisKeyGroupPersonIndex(rule,personNum)
	res,_ := redis.Int(redisDo("del",key))
	zlib.MyPrint("delOneRuleALLPlayers : ",res)
}
//====================================================

func  (queueSign *QueueSign)  delOneRuleOneGroupPersonIndexByGroupId(rule Rule,personNum int,groupId int){
	key := queueSign.getRedisKeyGroupPersonIndex(rule,personNum)
	res, _ := redisDo("ZREM",redis.Args{}.Add(key).Add(groupId)...)
	zlib.MyPrint("delOne GroupPersonIndexByGroupId : ",res)
}

//删除一个组的所有玩家信息
func  (queueSign *QueueSign) delOneRuleOneGroupPlayers(rule Rule ,groupId int){
	key := queueSign.getRedisKeyGroupPlayer(rule)
	res,_ := redisDo("ZREMRANGEBYSCORE",redis.Args{}.Add(key).Add(groupId).Add(groupId)...)
	zlib.MyPrint("delOne GroupPlayers : ",res)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupTimeout(rule Rule,groupId int){
	key := queueSign.getRedisKeyGroupTimeout(rule)
	res, _ := redisDo("ZREM",redis.Args{}.Add(key).Add(groupId)...)
	zlib.MyPrint("delOne GroupTimeoutById : ",res)
}
//删除一条规则的权限分组索引
func  (queueSign *QueueSign)  delOneRuleOneGroupWeight(rule Rule,groupId int){
	key := queueSign.getRedisKeyGroupWeight(rule)
	res,_ := redisDo("ZREM",redis.Args{}.Add(key).Add(groupId)...)
	zlib.MyPrint("delOneGroupWeight : ",res)
}
//删除一条规则的一个组的详细信息
func  (queueSign *QueueSign) delOneRuleOneGroupElement(rule Rule,groupId int){
	key := queueSign.getRedisKeyGroupElement(rule,groupId)
	res,_ := redisDo("del",key)
	zlib.MyPrint("delOneGroupElement : ",res)
}
//删除一个玩家的报名
func (queueSign *QueueSign) delOneByPlayerId(rule Rule,playerId int){
	groupId := queueSign.getPlayerIdById(rule,playerId)
	queueSign.delOneRuleOneGroup(rule,groupId)
	//这里还要更新当前组所有用户的，状态信息
}

//====================================================

//删除一个组
func (queueSign *QueueSign) delOneRuleOneGroup(rule Rule,groupId int){
	zlib.MyPrint("func : delOneRuleOneGroup id:" , groupId)
	group := queueSign.getGroupElementById(rule,groupId)
	zlib.MyPrint(group)

	queueSign.delOneRuleOneGroupElement(rule,groupId)
	queueSign.delOneRuleOneGroupWeight(rule,groupId)
	queueSign.delOneRuleOneGroupPersonIndexByGroupId(rule,group.GroupPerson,groupId)
	queueSign.delOneRuleOneGroupPlayers(rule,groupId)
	queueSign.delOneRuleOneGroupTimeout(rule,groupId)

	for _,v := range group.Players{
		playerStatus.signTimeoutUpInfo(v)
	}
}