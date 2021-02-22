package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
	"zlib"
	"strconv"
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


func (playerStatus *PlayerStatus) getRedisKey(playerId int)string{
	strPlayerId := strconv.Itoa( playerId )
	return playerStatus.GetCommonRedisPrefix() + strPlayerId
}

func (playerStatus *PlayerStatus) GetCommonRedisPrefix()string{
	return redisPrefix + redisSeparation + "status" + redisSeparation
}

func (playerStatus *PlayerStatus) GetOne(player Player)PlayerStatusElement{
	key := playerStatus.getRedisKey(player.Id)
	res ,err := redis.Values(myredis.RedisDo("HGETALL",key))
	//zlib.ExitPrint(res,err)
	if err != nil {
		zlib.ExitPrint("redis.Do error :",err.Error())
	}

	if res == nil || len(res) == 0{//证明，该KEY不存在
		mylog.Notice("PlayerStatusGetOne is empty!!!")
		playerStatusElement := PlayerStatusElement{
			PlayerId: player.Id,
			Status: PlayerStatusNotExist,
			ATime: zlib.GetNowTimeSecondToInt(),
			UTime: zlib.GetNowTimeSecondToInt(),
		}
		return playerStatusElement
	}
	//转换成  MapCovertStruct 函数  可用的类型 map[string]interface{}
	tmp := make(map[string]interface{})
	mapKey := ""
	for k,v := range res{
		val :=string(v.([]byte))
		//zlib.MyPrint(val)
		if k % 2 == 0{
			mapKey = val
			continue
		}else{
			tmp[mapKey] = val
		}
		//zlib.MyPrint("mapKey:",mapKey)
	}

	//zlib.MyPrint("tmp : ",tmp)
	playerStatusElement := &PlayerStatusElement{}
	zlib.MapCovertStruct(tmp,playerStatusElement)

	return *playerStatusElement
}
func (playerStatus *PlayerStatus)signTimeoutUpInfo(player Player){
	zlib.MyPrint("signTimeoutUpInfo",player.Id)
	playerStatusElement := playerStatus.GetOne(player)
	playerStatus.delOne(playerStatusElement)
}
func (playerStatus *PlayerStatus) checkSignTimeout(rule Rule,playerStatusElement PlayerStatusElement)(isTimeout bool ){
	now := zlib.GetNowTimeSecondToInt()
	if(now > playerStatusElement.SignTimeout){
		return true
	}
	return false
}
func  (playerStatus *PlayerStatus)  upInfo(playerStatusElement PlayerStatusElement,newPlayerStatusElement PlayerStatusElement)(bool ,error){
	//fmt.Printf("%+v , %+v \n",playerStatusElement,newPlayerStatusElement)
	mylog.Info("action Upinfo , old status :",playerStatusElement.Status , " new status :" ,newPlayerStatusElement.Status)
	newPlayerStatusElement.UTime = zlib.GetNowTimeSecondToInt()
	playerStatus.setNewOne(newPlayerStatusElement)
	return true,nil
}

func (playerStatus *PlayerStatus) upStatus(playerId int ,status int){
	player := Player {
		Id :  playerId,
	}
	playerStatusElement := playerStatus.GetOne(player)
	newPlayerStatusElement := playerStatusElement
	newPlayerStatusElement.Status = status
	mylog.Info("player up status , old status : ",playerStatusElement.Status , " new status : ",status)
	playerStatus.upInfo(playerStatusElement,newPlayerStatusElement)
}

func  (playerStatus *PlayerStatus) setNewOne(playerStatusElement PlayerStatusElement ){
	key := playerStatus.getRedisKey(playerStatusElement.PlayerId)
	//res,err := redisConn.Do("HMSET",redis.Args{}.Add(key).AddFlat(&playerStatusElement)...)
	res,err  := myredis.RedisDo("HMSET",redis.Args{}.Add(key).AddFlat(&playerStatusElement)...)
	if err != nil{
		zlib.ExitPrint("setNewOne redis error")
	}
	zlib.MyPrint("setNewOne",playerStatusElement,res)
}
//根据变量类型，统一全转换成string
func StructCovertStr(playerStatusElement interface{})string{
	t := reflect.TypeOf(playerStatusElement)
	v := reflect.ValueOf(playerStatusElement)

	commandValue := ""
	for k := 0; k < t.NumField(); k++ {
		val := v.Field(k).Interface()

		itemVal := ""
		typeStr := v.Field(k).Type().Name()
		//fmt.Printf("%s -- %v \n", t.Field(k).Name, typeStr)
		if typeStr  == "string"{
			itemVal = val.(string)
			if itemVal == ""{
				itemVal = "''"
			}
		}else if typeStr == "int"{
			itemVal = strconv.Itoa(val.(int))
		}else{
			zlib.ExitPrint("v.Field(k).Type().Name() : error")
		}
		commandValue += t.Field(k).Name + " " +itemVal + " "
	}

	return commandValue
}
func (playerStatus *PlayerStatus)  delOneById(playerId int){
	key := playerStatus.getRedisKey(playerId)
	res,_ := myredis.RedisDo("del",key)
	mylog.Notice("playerStatus delOneById , id : ",playerId , " , rs : ", res)
}

func (playerStatus *PlayerStatus)  delOne(playerStatusElement PlayerStatusElement){
	key := playerStatus.getRedisKey(playerStatusElement.PlayerId)
	res,_ := myredis.RedisDo("del",key)
	mylog.Notice("playerStatus delOne , id : ",playerStatusElement.PlayerId , " rs : ",res)
}
//删除所有玩家状态值
func  (playerStatus *PlayerStatus)  delAllPlayers(){
	mylog.Info("delAllPlayers ")
	key := playerStatus.GetCommonRedisPrefix()
	keys := key + "*"
	myredis.RedisDelAllByPrefix(keys)

}

