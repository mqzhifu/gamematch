package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"time"
)

/*
	匹配类型 - 规则
	1. N人匹配 ，只要满足N人，即成功
	2. N人匹配 ，划分为2个队，A队满足N人，B队满足N人，即成功

	权重		：根据某个用户上的某个特定属性值，计算出权重，优先匹配
	组		：ABC是一个组，一起参与匹配，那这3个人匹配的时候是不能分开的
	游戏属性	：游戏类型，也可以有子类型，如：不同的赛制。最终其实是分队列。不同的游戏忏悔分类，有不同的分类
*/



const (
	separation 				= "#"		//redis 内容-字符串分隔符
	redisSeparation 		= "_"		//redis key 分隔符
	IdsSeparation 			= ","		//多个ID 分隔符
	redisPrefix 			= "match"	//整个服务的，redis 前缀
	PlayerMatchingMaxTimes 	= 3			//一个玩家，参与匹配机制的最大次数，超过这个次数，证明不用再匹配了，目前没用上，目前使用的还是绝对的超时时间为准


)
type Gamematch struct {
	RedisHost 			string
	RedisPort 			string
	RuleConfig 			*RuleConfig

	containerSign 		map[int]*QueueSign	//容器 报名类
	containerSuccess 	map[int]*QueueSuccess
	containerMatch	 	map[int]*Match
	containerPush		map[int]*Push
}

//玩家结构体，目前暂定就一个ID，其它的字段值，后期得加，主要是用于权重计算
type Player struct {
	Id	int
}

var redisConn 		redis.Conn	//redis 连接实例
var playerStatus 	*PlayerStatus	//玩家状态类
var log *zlib.Log
var err *zlib.Err

func NewSelf()*Gamematch{
	log = zlib.NewLog("/data/www/golang/src/logs",zlib.LEVEL_ALL,15)
	container := getErrorCode()
	err = zlib.NewErr(log,container)

	gamematch := new (Gamematch)

	gamematch.RedisHost = "127.0.0.1"
	gamematch.RedisPort = "6379"

	redisConn ,_= NewRedisConn(gamematch.RedisHost,gamematch.RedisPort)
	gamematch.RuleConfig,_= NewRuleConfig()
	//if errs != nil{
	//
	//}

	gamematch.containerPush		= make(map[int]*Push)
	gamematch.containerSign 	= make(	map[int]*QueueSign )	//容器 报名类
	gamematch.containerSuccess = make(	map[int]*QueueSuccess)
	gamematch.containerMatch	= make( 	map[int]*Match)


	log.Info("init container....")
	//实例化容器
	for _,rule := range gamematch.RuleConfig.getAll(){
		gamematch.containerPush[rule.Id] = NewPush(rule)
		gamematch.containerSign[rule.Id] = NewQueueSign(rule)
		gamematch.containerSuccess[rule.Id] = NewQueueSuccess(rule,gamematch.containerPush[rule.Id])
		gamematch.containerMatch[rule.Id] = NewMatch(rule,gamematch.containerSign[rule.Id],gamematch.containerSuccess[rule.Id],gamematch.containerPush[rule.Id])

	}
	playerStatus = NewPlayerStatus()

	return gamematch
}


func (gamematch *Gamematch)getContainerSignByRuleId(ruleId int)*QueueSign{
	return gamematch.containerSign[ruleId]
}
func (gamematch *Gamematch)getContainerSuccessByRuleId(ruleId int)*QueueSuccess{
	return gamematch.containerSuccess[ruleId]
}
func (gamematch *Gamematch)getContainerPushByRuleId(ruleId int)*Push{
	return gamematch.containerPush[ruleId]
}

//取消- 匹配
func cancelJoin(ruleId int,player Player){

}
func (gamematch *Gamematch)testSignDel(ruleId int){
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	queueSign.delOneRule()

	queueSuccess := gamematch.getContainerSuccessByRuleId(ruleId)
	queueSuccess.delOneRule()

	playerStatus.delAllPlayers()
	zlib.ExitPrint(-55)
}
//报名 - 加入匹配队列
func (gamematch *Gamematch) Sign(ruleId int ,players []Player  )error{
	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		return err.NewErrorCode(400)
	}
	lenPlayers := len(players)
	if lenPlayers == 0{
		return err.NewErrorCode(401)
	}
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	//gamematch.testSignDel(2)

	groupsTotal := queueSign.getAllGroupsWeightCnt()	//报名 小组总数
	playersTotal := queueSign.getAllPlayersCnt()	//报名 玩家总数
	log.Info(" action :  Sign , players : " + strconv.Itoa(lenPlayers) +" ,queue cnt : groupsTotal",groupsTotal ," , playersTotal",playersTotal)
	now := zlib.GetNowTimeSecondToInt()

	log.Debug("start get player status data :")
	//检查，所有玩家的状态
	playerStatusElementMap := make( map[int]PlayerStatusElement )
	for _,player := range players{
		playerStatusElement := playerStatus.GetOne(player)
		//fmt.Printf("k :%d playerStatus %+v",k,playerStatusElement)
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		if playerStatusElement.Status == PlayerStatusNotExist{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			msg := make(map[int]string)
			msg[0] = strconv.Itoa(player.Id)
			return err.NewErrorCodeReplace(403,msg)
		}else if playerStatusElement.Status == PlayerStatusSign{
			isTimeout := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{//未超时
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return err.NewErrorCodeReplace(402,msg)
			}
			groupInfo := queueSign.getGroupElementById(playerStatusElement.GroupId)
			if groupInfo.Person > 1{
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return err.NewErrorCodeReplace(405,msg)
			}
			log.Notice("players sign timeout : delete one group... , gid : ",playerStatusElement.GroupId)
			queueSign.delOneRuleOneGroup(playerStatusElement.GroupId,1)
			//删除一个组，会连带着组里所有玩家的状态信息也被删除，这里再重新拉取一下，保证数据完整性
			playerStatusElement = playerStatus.GetOne(player)
		}

		playerStatusElementMap[player.Id] = playerStatusElement
	}
	log.Info("playerStatus data check finish.")
	//先计算一下权重平均值
	var playerWeight float32
	playerWeight = 0.00
	if rule.PlayerWeight.Formula != ""{
		//weight := 0.00
		//for k := range players{
			//rule.PlayerWeight.Formula ,这个公式怎么用还没想好
			//weight += 0.00
		//}z
		//playerWeight = float64( weight / len(players) )
	}
	//超时时间
	expire := now + rule.MatchTimeout

	group :=  gamematch.NewGroupStruct(rule)
	group.Players = players
	group.SignTimeout = expire
	group.Person = len(players)
	group.Weight = playerWeight

	log.Info("newGroupId : ",group.Id , "player/group weight : " ,playerWeight ," now : ",now ," expire : ",expire )
	//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
	//queueSign.Mutex.Lock()
	queueSign.AddOne(group)
	//defer queueSign.Mutex.Unlock()

	for _,player := range players{
		newPlayerStatusElement := playerStatusElementMap[player.Id]
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = playerWeight
		newPlayerStatusElement.SignTimeout = expire
		newPlayerStatusElement.GroupId = group.Id

		playerStatus.upInfo( playerStatusElementMap[player.Id],newPlayerStatusElement)
	}

	log.Info(" once sign finish , success players : ",len(players))
	return nil
}


func (gamematch *Gamematch) DemonAll(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	log.Info("start DemonAllRuleCheckSuccessTimeout ,  total : ",queueLen)
	if queueLen <=0 {
		log.Error(" rule is zero , no need")
		return
	}

	for _,rule := range queueList{
		checkRs,_ := gamematch.RuleConfig.CheckRuleByElement(rule)
		log.Info(" check rule rs :",checkRs)
		if !checkRs{
			log.Error("check rule element ,ruleId :",rule.Id , " continue ")
			continue
		}

		push := gamematch.getContainerPushByRuleId(rule.Id)
		push.checkStatus()
		zlib.ExitPrint(-999)
		//定时，检查，所有报名的组是否发生变化，将发生变化的组，删除掉
		queueSign := gamematch.getContainerSignByRuleId(rule.Id)
		queueSign.CheckTimeout(push)

		queueSuccess := gamematch.getContainerSuccessByRuleId(rule.Id)
		queueSuccess.CheckTimeout(push)


		match := gamematch.containerMatch[rule.Id]
		match.start()
	}
}

func (gamematch *Gamematch) DelAll(){
	log.Notice(" action :  DelAll")
	keys := redisPrefix + "*"
	redisDelAllByPrefix(keys)
}



func   mySleepSecond(second time.Duration){
	zlib.MyPrint(" ...sleep second ", strconv.Itoa(int(second))," ,loop waiting...")
	time.Sleep(second * time.Second)
}

func deadLoopBlock(sleepSecond time.Duration){
	for {
		mySleepSecond(sleepSecond)
	}
}

//进程结束，要清理一些数据
//func processEndCleanData(){
//	ruleConfig := NewRuleConfig()
//	if len(ruleConfig.Data) <= 0 {
//		zlib.ExitPrint("data len is 0 ")
//	}
//	for _,v := range ruleConfig.Data{
//		queueSign.delAllPlayers(v)
//		playerStatus.delAllPlayers()
//
//	}
//
//}


//func (gamematch *Gamematch)getLock(){
//	zlib.MyPrint(" get lock...")
//	queueSign.Mutex.Lock()
//}
//
//func (gamematch *Gamematch)unLock(){
//	zlib.MyPrint(" un lock...")
//	queueSign.Mutex.Unlock()
//}


func logging(){

}

