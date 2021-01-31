package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
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
	separation = "#"		//redis 内容-字符串分隔符
	redisSeparation = "_"	//redis key 分隔符
	IdsSeparation = ","		//多个ID 分隔符
	redisPrefix = "match"	//整个服务的，redis 前缀
	PlayerMatchingMaxTimes = 3	//一个玩家，参与匹配机制的最大次数，超过这个次数，证明不用再匹配了，目前没用上，目前使用的还是绝对的超时时间为准

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

type  Group struct {
	Id				int
	LinkId			int		//关联ID，匹配成功后关联成功的那条记录的ID，正常报名用不上
	Person			int		//小组人数
	Weight			float32	//小组权重
	MatchTimes		int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃，不过没用到
	SignTimeout		int		//多少秒后无人来取，即超时，更新用户状态，删除数据
	SuccessTimeout	int
	SignTime		int		//报名时间
	SuccessTime		int		//匹配成功时间
	Players 		[]Player
	Addition		string	//请求方附加属性值
	TeamId			int		//组队互相PK的时候，得成两个队伍
}
//玩家结构体，目前暂定就一个ID，其它的字段值，后期得加，主要是用于权重计算
type Player struct {
	Id	int
}

var redisConn 		redis.Conn	//redis 连接实例
var playerStatus 	*PlayerStatus	//玩家状态类

func NewSelf()*Gamematch{
	gamematch := new (Gamematch)

	gamematch.RedisHost = "127.0.0.1"
	gamematch.RedisPort = "6379"

	redisConn = NewRedisConn(gamematch.RedisHost,gamematch.RedisPort)
	gamematch.RuleConfig = NewRuleConfig()

	gamematch.containerPush		= make(map[int]*Push)
	gamematch.containerSign 	= make(	map[int]*QueueSign )	//容器 报名类
	gamematch.containerSuccess = make(	map[int]*QueueSuccess)
	gamematch.containerMatch	= make( 	map[int]*Match)


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

func  (gamematch *Gamematch) NewGroupStruct(rule Rule)Group{
	group := Group{}
	group.Id = gamematch.GetGroupIncId(rule.Id)
	group.LinkId = 0
	group.Person = 0
	group.Weight = 0
	group.MatchTimes = 0
	group.SignTimeout = zlib.GetNowTimeSecondToInt()
	group.SuccessTimeout = 0
	group.SignTime = 0
	group.SuccessTime = 0
	group.Addition = ""
	group.Players = nil

	return group
}

//组自增ID，因为匹配最小单位是基于组，而不是基于一个人，组ID就得做到全局唯一，很重要
func  (gamematch *Gamematch) getRedisGroupIncKey(ruleId int)string{
	signClass := gamematch.getContainerSignByRuleId(ruleId)
	return signClass.getRedisCatePrefixKey()  + redisSeparation + "group_inc_id"
}

//获取并生成一个自增GROUP-ID
func (gamematch *Gamematch) GetGroupIncId( ruleId int )  int{
	key := gamematch.getRedisGroupIncKey(ruleId)
	res,_ := redis.Int(redisDo("INCR",key))
	return res
}
func (gamematch *Gamematch)getContainerSignByRuleId(ruleId int)*QueueSign{
	return gamematch.containerSign[ruleId]
}
func (gamematch *Gamematch)getContainerSuccessByRuleId(ruleId int)*QueueSuccess{
	return gamematch.containerSuccess[ruleId]
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
func (gamematch *Gamematch) Sign(ruleId int ,players []Player  ){

	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		zlib.ExitPrint("ruleId not in redis db")
	}
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	//gamematch.testSignDel(2)

	groupsTotal := queueSign.getAllGroupsWeightCnt()	//报名 小组总数
	playersTotal := queueSign.getAllPlayersCnt()	//报名 玩家总数
	zlib.MyPrint("func : queue Sign groupsTotal",groupsTotal ,"playersTotal",playersTotal)
	now := zlib.GetNowTimeSecondToInt()

	zlib.MyPrint("start get player status data :")
	//检查，所有玩家的状态
	playerStatusElementMap := make( map[int]PlayerStatusElement )
	for _,player := range players{
		playerStatusElement := playerStatus.GetOne(player)
		//fmt.Printf("k :%d playerStatus %+v",k,playerStatusElement)
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		if playerStatusElement.Status == PlayerStatusNotExist{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			zlib.ExitPrint("sign err, Status:","PlayerStatusSuccess")
		}else if playerStatusElement.Status == PlayerStatusSign{
			isTimeout := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{
				zlib.ExitPrint("sign err, Status:","PlayerStatusSign")
			}

			zlib.MyPrint("sign timeout : need delete one group...")
			//zlib.MyPrint("playerStatus.CheckSignTimeout : ",hasSignTimeout)
			groupInfo := queueSign.getGroupElementById(playerStatusElement.GroupId)
			if groupInfo.Person > 1{
				//newPlayerStatusElement := *playerStatusElement
				//newPlayerStatusElement.SignTimeout = 0
				//newPlayerStatusElement.SuccessTimeout = 0
				//newPlayerStatusElement.Status = PlayerStatusNotExist
				//playerStatus.upInfo(*playerStatusElement)
				//上面是走更新流程，不删除这个KEY，但得加一个：NONE 状态，下面是直接删除redis key，更新结构体状态，略简单
				//playerStatusElement.Status = PlayerStatusNotExist
				//playerStatusElement.SignTimeout = 0
				//playerStatusElement.SuccessTimeout = 0
				queueSign.delOneRuleOneGroup(playerStatusElement.GroupId)
				playerStatus.delOne(playerStatusElement)
				//zlib.MyPrint(playerStatusElement)
				//if !isClear{//超时，但是没敢清除，因为此玩家所在组，还有其它人
				//	zlib.ExitPrint("sign err, have one player matching timeout : ",player.Id)
				//}

			}
			//证明，该玩家状态为报名，但已经被删了：playerStatus 和 group sign 信息
			playerStatusElement = playerStatus.GetOne(player)
		}

		playerStatusElementMap[player.Id] = playerStatusElement
	}
	zlib.MyPrint("playerStatus data check finish.")
	//先计算一下权重平均值
	var playerWeight float32
	playerWeight = 0.00
	if rule.PlayerWeight.Formula != ""{
		//weight := 0.00
		//for k := range players{
			//rule.PlayerWeight.Formula ,这个公式怎么用还没想好
			//weight += 0.00
		//}
		//playerWeight = float64( weight / len(players) )
	}
	//超时时间
	expire := now + rule.MatchTimeout

	group :=  gamematch.NewGroupStruct(rule)
	group.Players = players
	group.SignTimeout = expire
	group.Person = len(players)
	group.Weight = playerWeight

	zlib.MyPrint(" weight : " ,playerWeight ," now : ",now ," expire : ",expire , "new groupId : ",group.Id)
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

	zlib.MyPrint(" one sign finish,")
}
//开启，后台，所有需要匹配的rule
func (gamematch *Gamematch)  DemonStartAllRuleMatching(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	zlib.MyPrint("StartAllQueueMatching , queue total num : ",queueLen)
	if queueLen <=0 {
		zlib.MyPrint(" no need")
	}

	for _,rule := range queueList{
		if rule.Id == 1{
			continue
		}
		gamematch.checkRule(rule.Id)
		match := gamematch.containerMatch[rule.Id]
		match.start()
		zlib.ExitPrint(-888)
		//gamematch.checkMating(rule)
	}
}

func (gamematch *Gamematch) checkRule(ruleId int){

}

//定时，检查，所有报名的组是否发生变化，将发生变化的组，删除掉
func (gamematch *Gamematch) DemonAllRuleCheckSignTimeout(rule Rule){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	zlib.MyPrint("StartAllQueueMatching , queue total num : ",queueLen)
	if queueLen <=0 {
		zlib.MyPrint(" no need")
	}

	for _,rule := range queueList{
		 gamematch.CheckSignTimeout(rule)
		//gamematch.checkMating(rule)
	}
}

func (gamematch *Gamematch) DemonAllRuleCheckSuccessTimeout(rule Rule){

}
func (gamematch *Gamematch) DemonAllRuleCheckResultPush(rule Rule){

}

func (gamematch *Gamematch) DelAll(){
	keys := redisPrefix + "*"
	redisDelAllByPrefix(keys)
}



func  (gamematch *Gamematch) CheckSignTimeout(rule Rule){
	queueSign := gamematch.getContainerSignByRuleId(rule.Id)
	keys := queueSign.getRedisKeyGroupSignTimeout()
	now := zlib.GetNowTimeSecondToInt()

	res,err := redis.IntMap(  redisDo("ZREVRANGEBYSCORE",redis.Args{}.Add(keys).Add(now).Add("-inf").Add("WITHSCORES")...))
	zlib.MyPrint(res)
	if err != nil{
		zlib.ExitPrint("redis keys err :",err.Error())
	}

	zlib.MyPrint("checkMating timeout group total : ",len(res))
	if len(res) == 0{
		zlib.MyPrint(" empty , no need process")
		return
	}
	for groupId,_ := range res{
		queueSign.delOneRuleOneGroup(zlib.Atoi(groupId))
		//zlib.ExitPrint(-199)
	}
}

func  GroupStrToStruct(redisStr string)Group{
	//zlib.MyPrint("redis str : ",redisStr)
	strArr := strings.Split(redisStr,separation)
	//zlib.ExitPrint("redisStr",redisStr)
	playersArr := strings.Split(strArr[9],IdsSeparation)
	var players []Player
	for _,v := range playersArr{
		players = append(players,Player{Id : zlib.Atoi(v)})
	}

	element := Group{
		Id 				:	zlib.Atoi(strArr[0]),
		LinkId			: 	zlib.Atoi(strArr[1]),
		Person 			:	zlib.Atoi(strArr[2]),
		Weight			:	zlib.StringToFloat(strArr[3]),
		MatchTimes 		:	zlib.Atoi(strArr[4]),
		SignTimeout 	:	zlib.Atoi(strArr[5]),
		SuccessTimeout	: 	zlib.Atoi(strArr[6]),
		SignTime		: 	zlib.Atoi(strArr[7]),
		SuccessTime		: 	zlib.Atoi(strArr[8]),
		Players 		:	players,
		Addition		:	strArr[10],
		TeamId			:	zlib.Atoi(strArr[11]),
	}

	return element
}

func  GroupStructToStr(group Group)string{
	//Weight	float32	//小组权重
	//MatchTimes	int		//已匹配过的次数，超过3次，证明该用户始终不能匹配成功，直接丢弃

	playersIds := ""
	for _,v := range group.Players{
		playersIds +=  strconv.Itoa(v.Id) + IdsSeparation
	}
	playersIds = playersIds[0:len(playersIds)-1]
	Weight := zlib.FloatToString(group.Weight,3)

	content :=
		strconv.Itoa(  group.Id )+ separation +
		strconv.Itoa(  group.LinkId )+ separation +
		strconv.Itoa(  group.Person )+ separation +
		Weight + separation +
		strconv.Itoa(  group.MatchTimes )+ separation +
		strconv.Itoa(  group.SignTimeout )+ separation +
		strconv.Itoa(  group.SuccessTimeout )+ separation +
		strconv.Itoa(  group.SignTime )+ separation +
		strconv.Itoa(  group.SuccessTime )+ separation +
	 	playersIds + separation +
		group.Addition +separation +
		strconv.Itoa(  group.TeamId)

	return content
}

func   mySleepSecond(second time.Duration){
	zlib.MyPrint(" no enough persons...sleep 1 ,loop waiting...")
	time.Sleep(second * time.Second)
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

