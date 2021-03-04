package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"zlib"
)

// 1. 等待计算   2计算中   3匹配成功
type PlayerStatusElement struct {
	PlayerId 		int
	Status 			int
	RuleId			int
	Weight			float32
	GroupId			int
	ATime 			int
	UTime 			int
	SignTimeout		int
	SuccessTimeout	int
}

type PlayerStatus struct {

}

func NewPlayerStatus()*PlayerStatus{
	mylog.Info("Global var :NewPlayerStatus")
	playerStatus := new(PlayerStatus)
	return playerStatus
}

func (playerStatus *PlayerStatus) newPlayerStatusElement ()PlayerStatusElement{
	newPlayerStatusElement := PlayerStatusElement{
		PlayerId: 0,
		Status:PlayerStatusInit,
		RuleId:0,
		SignTimeout:0,
		GroupId:0,
	}
	return newPlayerStatusElement
}

func (playerStatus *PlayerStatus) getRedisPrefixKey( )string{
	return redisPrefix + redisSeparation + "pstatus"+ redisSeparation
	//strPlayerId := strconv.Itoa( playerId )
	//return playerStatus.GetCommonRedisPrefix() + strPlayerId
}

func (playerStatus *PlayerStatus) getRedisStatusPrefixByPid(playerId int)string{
	//return redisPrefix + redisSeparation + "status" + redisSeparation
	//return redisPrefix + redisSeparation + "status"
	return playerStatus.getRedisPrefixKey()+ strconv.Itoa( playerId )
}

func (playerStatus *PlayerStatus) getRulePlayerPrefixById(ruleId int)string{
	return playerStatus.getRedisPrefixKey() + "rule" + redisSeparation + strconv.Itoa(ruleId)
}
//索引，一个rule里包含的玩家信息，主要用于批量删除，也可用做一个rule的当前所有玩家列表
func (playerStatus *PlayerStatus) addOneRulePlayer(playerId int ,ruleId int){
	key := playerStatus.getRulePlayerPrefixById(ruleId)
	res,err := myredis.RedisDo("zadd",redis.Args{}.Add(key).Add(0).Add(playerId)...)
	mylog.Info("addRulePlayer:",res,err)
}

func (playerStatus *PlayerStatus) delOneRulePlayer(playerId int ,ruleId int){
	key := playerStatus.getRulePlayerPrefixById(ruleId)
	res,err := myredis.RedisDo("zrem",redis.Args{}.Add(key).Add(playerId)...)
	mylog.Info("delOneRulePlayer:",res,err)
}
func (playerStatus *PlayerStatus) getOneRuleAllPlayer( ruleId int)[]string{
	key := playerStatus.getRulePlayerPrefixById(ruleId)
	//ZRANGEBYSCORE salary -inf +inf WITHSCORES
	res,err := redis.Strings( myredis.RedisDo("ZRANGEBYSCORE",redis.Args{}.Add(key).Add("-inf").Add("+inf")...))
	mylog.Info("getOneRuleAllPlayer:",res,err)
	return res
}

//根据PID 获取一个玩家的状态信息
func (playerStatus *PlayerStatus)GetById(playerId int)(playerStatusElement PlayerStatusElement ,isEmpty int){
	//var playerStatusElement PlayerStatusElement
	key := playerStatus.getRedisStatusPrefixByPid(playerId)
	res ,err := redis.Values(myredis.RedisDo("HGETALL",key))
	if err != nil{
		return playerStatusElement,1
	}
	//playerStatusElement := &PlayerStatusElement{}
	if err := redis.ScanStruct(res, &playerStatusElement); err != nil {
		return playerStatusElement,1
	}
	//playerStatusElement =  playerStatus.strToStruct(res)
	return playerStatusElement,0
}

func  (playerStatus *PlayerStatus)  upInfo( newPlayerStatusElement PlayerStatusElement)(bool ,error){
	//fmt.Printf("%+v , %+v \n",playerStatusElement,newPlayerStatusElement)
	//queueSign.Log.Info("upInfo status , old :",playerStatusElementMap[player.Id].Status ,  " , new : ",newPlayerStatusElement.Status )
	//mylog.Info("action Upinfo , old status :",playerStatusElement.Status , " new status :" ,newPlayerStatusElement.Status)

	oldPlayerStatusElement,isEmpty := playerStatus.GetById(newPlayerStatusElement.PlayerId)
	oldPlayerStatusElement.UTime = 0 //先置0，方便下面两个结构体比较
	if isEmpty == 1{
		mylog.Info("playerStatus add new One")
	}else{
		if oldPlayerStatusElement == newPlayerStatusElement{
			return false,myerr.NewErrorCode(458)
		}
		mylog.Info("playerStatus up One")
	}
	playerStatus.delOneRulePlayer(newPlayerStatusElement.PlayerId,newPlayerStatusElement.RuleId)
	playerStatus.addOneRulePlayer(newPlayerStatusElement.PlayerId,newPlayerStatusElement.RuleId)
	newPlayerStatusElement.UTime = zlib.GetNowTimeSecondToInt()
	playerStatus.setInfo(newPlayerStatusElement)

	return true,nil
}

func (playerStatus *PlayerStatus)strToStruct(str string)PlayerStatusElement{
	strArr := strings.Split(str,separation)
	playerStatusElement := PlayerStatusElement{
		PlayerId 		: zlib.Atoi(strArr[0]),
		Status 			:zlib.Atoi(strArr[1]),
		RuleId			:zlib.Atoi(strArr[2]),
		//Weight			:zlib.Atoi(strArr[0]),
		GroupId			:zlib.Atoi(strArr[4]),
		ATime 			:zlib.Atoi(strArr[5]),
		UTime 			:zlib.Atoi(strArr[6]),
		SignTimeout		:zlib.Atoi(strArr[7]),
		SuccessTimeout	:zlib.Atoi(strArr[8]),
	}

	return playerStatusElement
}

func  (playerStatus *PlayerStatus) setInfo(playerStatusElement PlayerStatusElement ){
	key := playerStatus.getRedisStatusPrefixByPid(playerStatusElement.PlayerId)
	res,err  := myredis.RedisDo("HMSET",redis.Args{}.Add(key).AddFlat(&playerStatusElement)...)
	mylog.Info("playerStatus setInfo : ",playerStatusElement,res,err)
}

func (playerStatus *PlayerStatus)  delOneById(playerId int){
	playerStatusElement,isEmpty := playerStatus.GetById(playerId)
	if isEmpty == 1{
		mylog.Error(" getByid is empty!!!")
		return
	}
	key := playerStatus.getRedisStatusPrefixByPid(playerId)
	res,_ := myredis.RedisDo("del",key)
	mylog.Notice("playerStatus delOneById , id : ",playerId , " , rs : ", res)

	playerStatus.delOneRulePlayer(playerId,playerStatusElement.RuleId)
}
//删除所有玩家状态值
func  (playerStatus *PlayerStatus)  delAllPlayers(){
	mylog.Warning("delAllPlayers ")
	key := playerStatus.getRedisPrefixKey()
	keys := key + "*"
	myredis.RedisDelAllByPrefix(keys)
}
func (playerStatus *PlayerStatus) checkSignTimeout(rule Rule,playerStatusElement PlayerStatusElement)(isTimeout bool ){
	now := zlib.GetNowTimeSecondToInt()
	if(now > playerStatusElement.SignTimeout){
		return true
	}
	return false
}

//func (playerStatus *PlayerStatus)  delOne(playerStatusElement PlayerStatusElement){
//	key := playerStatus.getRedisStatusPrefixByPid(playerStatusElement.PlayerId)
//	res,_ := myredis.RedisDo("del",key)
//	mylog.Notice("playerStatus delOne , id : ",playerStatusElement.PlayerId , " rs : ",res)
//}

//旧方法，暂时放弃
//func (playerStatus *PlayerStatus) GetOne(player Player)PlayerStatusElement{
//	key := playerStatus.getRedisStatusPrefixByPid(player.Id)
//	res ,err := redis.Values(myredis.RedisDo("HGETALL",key))
//	//zlib.ExitPrint(res,err)
//	if err != nil {
//		zlib.ExitPrint("redis.Do error :",err.Error())
//	}
//
//	if res == nil || len(res) == 0{//证明，该KEY不存在
//		mylog.Notice("PlayerStatusGetOne is empty!!!")
//		playerStatusElement := PlayerStatusElement{
//			PlayerId: player.Id,
//			Status: PlayerStatusNotExist,
//			ATime: zlib.GetNowTimeSecondToInt(),
//			UTime: zlib.GetNowTimeSecondToInt(),
//		}
//		return playerStatusElement
//	}
//	//转换成  MapCovertStruct 函数  可用的类型 map[string]interface{}
//	tmp := make(map[string]interface{})
//	mapKey := ""
//	for k,v := range res{
//		val :=string(v.([]byte))
//		//zlib.MyPrint(val)
//		if k % 2 == 0{
//			mapKey = val
//			continue
//		}else{
//			tmp[mapKey] = val
//		}
//		//zlib.MyPrint("mapKey:",mapKey)
//	}
//
//	//zlib.MyPrint("tmp : ",tmp)
//	playerStatusElement := &PlayerStatusElement{}
//	zlib.MapCovertStruct(tmp,playerStatusElement)
//
//	return *playerStatusElement
//}
//func (playerStatus *PlayerStatus)signTimeoutUpInfo(player Player){
//	zlib.MyPrint("signTimeoutUpInfo",player.Id)
//	playerStatusElement := playerStatus.GetOne(player)
//	playerStatus.delOne(playerStatusElement)
//}
//func (playerStatus *PlayerStatus) upStatus(playerId int ,status int){
//	player := Player {
//		Id :  playerId,
//	}
//	playerStatusElement := playerStatus.GetOne(player)
//	newPlayerStatusElement := playerStatusElement
//	newPlayerStatusElement.Status = status
//	mylog.Info("player up status , old status : ",playerStatusElement.Status , " new status : ",status)
//	playerStatus.upInfo(playerStatusElement,newPlayerStatusElement)
//}
////根据变量类型，统一全转换成string
//func StructCovertStr(playerStatusElement interface{})string{
//	t := reflect.TypeOf(playerStatusElement)
//	v := reflect.ValueOf(playerStatusElement)
//
//	commandValue := ""
//	for k := 0; k < t.NumField(); k++ {
//		val := v.Field(k).Interface()
//
//		itemVal := ""
//		typeStr := v.Field(k).Type().Name()
//		//fmt.Printf("%s -- %v \n", t.Field(k).Name, typeStr)
//		if typeStr  == "string"{
//			itemVal = val.(string)
//			if itemVal == ""{
//				itemVal = "''"
//			}
//		}else if typeStr == "int"{
//			itemVal = strconv.Itoa(val.(int))
//		}else{
//			zlib.ExitPrint("v.Field(k).Type().Name() : error")
//		}
//		commandValue += t.Field(k).Name + " " +itemVal + " "
//	}
//
//	return commandValue
//}


