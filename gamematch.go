package gamematch

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"src/zlib"
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
	RuleFlagVS = 1	//N人 对战 N人，对战略的匹配，以N人为一个队伍
	RuleFlagGroup = 2 //满足N人，即可
)

const (
	separation = "#"
	redisSeparation = "_"
	IdsSeparation = ","
	redisPrefix = "match"
	PlayerMatchingMaxTimes = 3	//一个玩家，参与匹配机制的最大次数，超过这个次数，证明不用再匹配了

)
type Gamematch struct {
	RedisHost string
	RedisPort string
	RuleConfig *RuleConfig
}

type Group struct {
	Id 		int
	Weight	int		//当前组的权重值
	Uids	string  //当前组包含的用户ID
}

type Player struct {
	Id	int
}

var redisConn redis.Conn
var queueSign *QueueSign
var queueSuccess *QueueSuccess
var playerStatus *PlayerStatus

func NewSelf()*Gamematch{
	Gamematch := new (Gamematch)

	Gamematch.RedisHost = "127.0.0.1"
	Gamematch.RedisPort = "6379"

	redisConn = InitConn(Gamematch.RedisHost,Gamematch.RedisPort)
	Gamematch.RuleConfig = NewRuleConfig()

	queueSign = NewQueueSign()
	queueSuccess = NewQueueSuccess()
	playerStatus = NewPlayerStatus()

	return Gamematch
}

func getNowTime()int{
	return int( time.Now().Unix() )
}
//报名 - 加入匹配队列
func (gamematch *Gamematch) Sign(ruleId int ,players []Player  ){

	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		zlib.ExitPrint("ruleId not in db")
	}
	//queueSign.delOneRuleAll(rule)
	//playerStatus.delAllPlayers()
	//zlib.ExitPrint(-100)

	groupsTotal := queueSign.getGroupsTotal(rule)
	playersTotal := queueSign.getPlayersTotal(rule)
	zlib.MyPrint("func : queue Sign groupsTotal",groupsTotal ,"playersTotal",playersTotal)
	now := getNowTime()

	zlib.MyPrint("start get player status data :")
	//检查，所有玩家的状态
	playerStatusElementMap := make( map[int]PlayerStatusElement )
	for k,player := range players{
		playerStatusElement := playerStatus.GetOne(player)
		fmt.Printf("k :%d playerStatus %+v",k,playerStatusElement)
		//zlib.ExitPrint(-100)
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		if playerStatusElement.Status == PlayerStatusNotExist{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			zlib.ExitPrint("sign err, Status:","PlayerStatusSuccess")
		}else if playerStatusElement.Status == PlayerStatusSign{
			isTimeout,isClear := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{
				zlib.ExitPrint("sign err, Status:","PlayerStatusSign")
			}
			if !isClear{//超时，但是没敢清除，因为此玩家所在组，还有其它人
				zlib.ExitPrint("sign err, have one player matching timeout : ",player.Id)
			}
			//证明，该玩家状态为报名，但已经被删了：playerStatus 和 group sign 信息
			playerStatusElement = playerStatus.GetOne(player)
		}

		playerStatusElementMap[player.Id] = playerStatusElement
	}
	zlib.MyPrint("start get player status data finish.")
	//先计算一下权重平均值
	var playerWeight float32
	playerWeight= 0.00
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
	//内部生成 自增GroupId，外部请求也会给个groupId,我还没想好怎么处理
	groupId := queueSign.GetGroupIncId(rule)

	zlib.MyPrint(" now : ",now ," expire : ",expire , "new groupId : ",groupId)
	queueSignElement := QueueSignGroupElement{
		Players: players,
		GroupId:groupId,
		Timeout:expire,
		GroupPerson:len(players),
		GroupWeight: playerWeight,
		MatchTimes: 0,
	}
	//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
	//queueSign.Mutex.Lock()
	queueSign.PushOneGroup(queueSignElement,rule)
	//defer queueSign.Mutex.Unlock()

	for _,player := range players{
		newPlayerStatusElement := playerStatusElementMap[player.Id]
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = playerWeight
		newPlayerStatusElement.SignTimeout = expire
		newPlayerStatusElement.GroupId = groupId

		playerStatus.upInfo( playerStatusElementMap[player.Id],newPlayerStatusElement)
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

//开启，后台，所有需要匹配的rule
func (gamematch *Gamematch)  StartAllQueueMatching(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	zlib.MyPrint("StartAllQueueMatching , queue total num : ",queueLen)
	if queueLen <=0 {
		zlib.MyPrint(" no need")
	}

	for _,rule := range queueList{
		gamematch.queueMatching(rule  )
		//gamematch.checkMating(rule)
	}
}

func checkCancel(player Player){
	//status := getPlayerStatus(player)
	//if status != PlayerStatusWait{
	//
	//}
}
//一条rule 的匹配
func (gamematch *Gamematch) queueMatching(rule Rule){
	zlib.MyPrint("start one rule queueMatching : ")
	for{
		//先做最精细的匹配，也就是10个档中，每个档都做匹配
		successPlayers,isEmpty := gamematch.matching(rule)
		if isEmpty == 1{
			gamematch.matchingSleep(1)
			continue
		}
		if len(successPlayers) > 0 {

			continue
		}
		zlib.ExitPrint(-400)
	}
}

func (gamematch *Gamematch)getLock(){
	zlib.MyPrint(" get lock...")
	queueSign.Mutex.Lock()
}

func (gamematch *Gamematch)unLock(){
	zlib.MyPrint(" un lock...")
	queueSign.Mutex.Unlock()
}
//一条rule的-全匹配过程
func (gamematch *Gamematch)matching(rule Rule)(successPlayers map[int]QueueSignGroupElement,isEmpty int){
	zlib.MyPrint("start matching func :")
	gamematch.getLock()
	defer gamematch.unLock()
	successGroupIds,isEmpty := gamematch.matchingRangeAll(rule)
	if isEmpty == 1{

	}
	if len(successGroupIds)> 0 {
		//for k,oneCondition:=range successGroupIds{
		//
		//}
	}
	//上面都是基于最大范围：全部玩家搜索，下面是做精细化搜索
	//zlib.MyPrint("max range search end, no match")
	//zlib.MyPrint("middle range search start:")
	//var tmpMin int
	//var tmpMax float32
	//flag := 1
	//for i:=0;i<10;i++{
	//	if flag == 1{//区域性的
	//		tmpMin = i
	//		tmpMax = float32(i) + 0.9999
	//	}else{//递增，范围更大
	//		tmpMin = 1
	//		tmpMax = float32(i) + 0.9999
	//	}
	//
	//	cnt := queueSign.getGroupWeightRangeCnt(rule,tmpMin,tmpMax)
	//	zlib.MyPrint("search ",tmpMin , " - ",tmpMax , "cnt:",cnt)
	//	//这里应该再 +N 人，一次取出的玩家要冗余几个人，防止符合条件 这些玩家里面有人异常
	//	//如：状态错误、数据丢失、取消匹配等，还有一种情况是其它队列也在算，这种情况我还得想想
	//	if cnt < personCondition { //人数不够，放弃本次，计算下一个阶梯
	//		zlib.MyPrint("cnt < personCondition , continue")
	//		continue
	//	}
	//	zlib.ExitPrint(-500)
	//}
	return successPlayers,isEmpty
}
//匹配时，开始应该先做全局匹配
func (gamematch *Gamematch)matchingRangeAll(rule Rule)(successGroupIds map[int]map[int]int,isEmpty int){
	//先做最大范围的搜索：所有报名匹配的玩家
	zlib.MyPrint("max range search start:")
	playersTotal := queueSign.getPlayersTotal(rule)
	groupsTotal := queueSign.getGroupsTotal(rule)
	zlib.MyPrint("playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	if playersTotal <= 0 ||  groupsTotal <= 0 {
		zlib.MyPrint("total is 0")
		return successGroupIds,1
	}
	personCondition := rule.PersonCondition
	if rule.Flag == RuleFlagVS { //组队/对战类
		personCondition = rule.TeamVSPerson * 2
	}
	zlib.MyPrint("personCondition:",personCondition)

	if playersTotal < personCondition{
		zlib.MyPrint("total < personCondition ")
		return successGroupIds,1
	}
	//测试，先注释掉了
	//集合中的：总玩家数<小于>满足条件玩家数 2倍~，没必要再走算法了，直接算就行了
	//if playersTotal > personCondition * 2 {
	//	return successGroupIds,0
	//}
	zlib.MyPrint(" total <= *2 ")
	successGroupIds = gamematch.matchingCalculateNumberTotal(rule,"-inf","+inf")

	return successGroupIds,2
}
func  (gamematch *Gamematch) matchingCalculateNumberTotal(rule Rule,rangeStart string,rangeEnd string)(successGroupIds map[int]map[int]int){
	//以小组为单位，取出每种人数对应的小组数
	//1=>n ,2=>n,3=>n,4=>n,5=>n
	groupPersonNum := make(map[int]int)
	//这里吧，按说取，一条rule最大的值就行，没必要取全局最大的5，但是吧，后面的算法有点LOW，外层循环数就是5，目前没想到太好的解决办法 ，回头我再想想
	for i:=1;i<=rule.groupPersonMax;i++{
		groupPersonNum[i] = queueSign.getGroupPersonTotal(rule,i)
	}
	zlib.MyPrint("groupPersonNum ",groupPersonNum)
	if rule.Flag != RuleFlagVS{//集齐人数即可
		calculateNumberTotalRs := gamematch.calculateNumberTotal(rule.PersonCondition,groupPersonNum)
		zlib.MyPrint(calculateNumberTotalRs)
		//总人数虽然够了，但是没有成团，可能有些极端的情况 ，比如：全是4人为一组报名，成团人数却是10
		if len(calculateNumberTotalRs) == 0{
			return successGroupIds
		}

		for k,oneOkPersonCondition := range calculateNumberTotalRs{
			for person,num := range oneOkPersonCondition{
				if num <= 0{
					zlib.MyPrint("person : ",person , "num : ",num ," >=0  ,continue")
					continue
				}
				groupIds  := queueSign.getGroupPersonRangeListAll(rule,person + 1,0,num)
				successGroupIds[k] = groupIds
				//for _,v := range groupIds{
				//	groupElement := queueSign.getGroupElementById(rule,v)
				//}
			}
		}
		return successGroupIds

	}else{//组队，互相PK

		//第一轮，过滤，只求互补数，另外也是为了公平：5人匹配，对手也是5人组队的
		//如：5没有互补，4的互补是1，3的互补是2，2的互补是3，1互补是4
		//5已知最大的，且直接满足不做处理，其实4计算完了，1就不用算了，3计算完了，2其实也不用算了，
		//最终：实际上需要处理的是N / 2 个数

		//for k,v := range groupPersonNum{
		//	if v == 0{
		//		continue
		//	}
		//	if k == RuleGroupPersonMax{//这是最大值的情况，属于直接即满足条件，不需要再计算了
		//		if v <= 1{
		//			continue
		//		}
		//		limitOffset := 0
		//		var limitCnt int
		//		divider := v / 2
		//		if divider > 0{//两个互相PK，出现偶数，证明，小组数刚好
		//			limitCnt = int(v / 2)
		//		}else{
		//			limitCnt = v / 2
		//		}
		//		groupIds  := queueSign.getGroupPersonRangeListAll(rule,v,limitOffset,limitCnt)
		//		//取出的数组，要立刻删除~避免计算混乱，且要原子 操作
		//	}
		//}
		////上面一轮筛选过后，5人的：最多只剩下一个
		////4人的，可能匹配光了，也可能剩下N个，因为取决于1人有多少，如果1人被匹配光了，4人组剩下多少个都没意义了，因为无法再组成一个团队了，所以4人组即就直接忽略
		////剩下 3 2 1 ，没啥快捷的办法了，只能单纯的计算组成5人了
		//
		////重新再拉一遍数据，上面计算完后，会删除掉一些匹配成功的数据
		//groupPersonNum := make(map[int]int)
		////这里吧，按说取，一条rule最大的值就行，没必要取全局最大的5，但是吧，后面的算法有点LOW，外层循环数就是5，目前没想到太好的解决办法 ，回头我再想想
		//for i:=1;i<=rule.groupPersonMax;i++{
		//	groupPersonNum[i] = queueSign.getGroupPersonTotal(rule,i)
		//}
		//
		//calculateNumberTotalRs := gamematch.calculateNumberTotal(rule.TeamVSPerson,groupPersonNum)
		//if len(calculateNumberTotalRs) == 0{
		//
		//}
		//
		//for i:=0;i<=len(calculateNumberTotalRs[0]);i++{
		//	groupIds  := queueSign.getGroupPersonRangeListAll(rule,i+1,0,calculateNumberTotalRs[0][i])
		//	if len(groupIds) / 2 == 0{
		//
		//	}
		//	for k,v := range groupIds{
		//		groupElement := queueSign.getGroupElementById(rule,v)
		//	}
		//}

		//取出的数组，要立刻删除~避免计算混乱，且要原子 操作
	}
	return successGroupIds
}
//定时，检查，所有报名的组是否发生变化，将发生变化的组，删除掉
func (gamematch *Gamematch) checkMating(rule Rule){
	keys := queueSign.getRedisKeyGroupTimeout(rule)
	now := getNowTime()

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
		queueSign.delOneRuleOneGroup(rule,zlib.Atoi(groupId))
		//zlib.ExitPrint(-199)
	}
}
//errCode :  0 正常
func (gamematch *Gamematch) matchingWeightCheckOnePlayer(playerId int) int {
	errCode := 0
	////不能一直循环重复的匹配相同的玩家，有些玩家可能就是永远也匹配不上
	//if element.MatchTimes > PlayerMatchingMaxTimes{
	//	errCode = -1
	//	return errCode
	//}
	//这里还要再判断一下玩家的 状态，因为期间可能发生变化，比如：玩家取消了匹配，玩家断线了等
	tmpPlayer := Player{Id:playerId}
	playerStatus := playerStatus.GetOne(tmpPlayer)
	if playerStatus.Status != PlayerStatusSign{
		errCode = -3
		return errCode
	}
	return errCode
}

func  (gamematch *Gamematch) matchingSleep(second time.Duration){
	zlib.MyPrint(" no enough persons...sleep 1 ,loop waiting...")
	time.Sleep(second * time.Second)
}

func (gamematch *Gamematch) DelAll(){
	keys := redisPrefix + "*"
	redisDelAllByPrefix(keys)
}


//取消- 匹配
func cancelJoin(player Player){

}

func logging(){

}

