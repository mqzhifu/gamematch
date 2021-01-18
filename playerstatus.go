package gamematch

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"reflect"
	"src/zlib"
	"strconv"
)

const (
	PlayerStatusNotExist = 1//redis中还没有该玩家信息
	PlayerStatusSign = 2	//已报名，等待匹配
	PlayerStatusSuccess = 3	//匹配成功，等待拿走
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
	res ,err := redis.Values(redisConn.Do("HGETALL",key))
	//zlib.ExitPrint(res,err)
	if err != nil {
		zlib.ExitPrint("redis.Do error :",err.Error())
	}

	if res == nil || len(res) == 0{//证明，该KEY不存在
		playerStatusElement := PlayerStatusElement{
			PlayerId: player.Id,
			Status: PlayerStatusNotExist,
			ATime: getNowTime(),
			UTime: getNowTime(),
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

	hasSignTimeout := playerStatus.CheckSignTimeout(*playerStatusElement)
	if hasSignTimeout{
		//newPlayerStatusElement := *playerStatusElement
		//newPlayerStatusElement.SignTimeout = 0
		//newPlayerStatusElement.SuccessTimeout = 0
		//newPlayerStatusElement.Status = PlayerStatusNotExist
		//playerStatus.upInfo(*playerStatusElement)
		//上面是走更新流程，不删除这个KEY，但得加一个：NONE 状态，下面是直接删除redis key，更新结构体状态，略简单
		playerStatusElement.Status = PlayerStatusNotExist
		playerStatusElement.SignTimeout = 0
		playerStatusElement.SuccessTimeout = 0
		playerStatus.delOne(*playerStatusElement)
	}
	//zlib.MyPrint(playerStatusElement)

	return *playerStatusElement
}

//
func (playerStatus *PlayerStatus) CheckSignTimeout(playerStatusElement PlayerStatusElement)bool{
	now := getNowTime()
	if(now >= playerStatusElement.SignTimeout){
		return true
	}
	return false
}
//
func  (playerStatus *PlayerStatus)  upInfo(playerStatusElement PlayerStatusElement,newPlayerStatusElement PlayerStatusElement)(bool ,error){
	fmt.Printf("%+v , %+v \n",playerStatusElement,newPlayerStatusElement)
	newPlayerStatusElement.UTime = getNowTime()
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
	playerStatus.upInfo(playerStatusElement,newPlayerStatusElement)
}

func  (playerStatus *PlayerStatus) setNewOne(playerStatusElement PlayerStatusElement ){
	key := playerStatus.getRedisKey(playerStatusElement.PlayerId)
	//res,err := redisConn.Do("HMSET",redis.Args{}.Add(key).AddFlat(&playerStatusElement)...)
	res,err  := redisDo("HMSET",redis.Args{}.Add(key).AddFlat(&playerStatusElement)...)
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

func (playerStatus *PlayerStatus)  delOne(playerStatusElement PlayerStatusElement){
	key := playerStatus.getRedisKey(playerStatusElement.PlayerId)
	redisDo("del",key)
}
//删除所有玩家状态值
func  (playerStatus *PlayerStatus)  delAllPlayers(){
	key := playerStatus.GetCommonRedisPrefix()
	keys := key + "*"
	redisDelAllByPrefix(keys)
}

