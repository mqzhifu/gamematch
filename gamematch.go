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
	redisPrefix = "match"

)
type Gamematch struct {
	RedisHost string
	RedisPort string
	RuleConfig *RuleConfig
}


//
// 组    group ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值     ( = ,>= ,<=, > X < , >= X < ,>= X <= X )
// 单用户 normal ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值
// 团队  team   ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值


type Group struct {
	Id 		int
	Weight	int		//当前组的权重值
	Uids	string  //当前组包含的用户ID
}

//type Team struct {
//	Id 		int
//	Weight	int		//当前组的权重值
//	Uids	string  //当前组包含的用户ID
//}

type Player struct {
	Id	int
}

var redisConn redis.Conn
var queueSign *QueueSign
var queueSuccess *QueueSuccess
var playerStatus *PlayerStatus

//func getAllQueueRedisKey()string{
//	return redisPrefix + redisSeparation + "queues"
//}

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
func (gamematch *Gamematch) Sign(ruleId int ,players []*Player  ){
	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		zlib.ExitPrint("ruleId not in db")
	}
	queueKey := queueSign.getRedisKey(rule)
	playerNum := len(players)
	zlib.MyPrint("playerNum",playerNum ,"queueKey",queueKey)
	now := getNowTime()

	//检查，所有玩家的状态
	playerStatusElementMap := make( map[int]PlayerStatusElement )
	for k,player := range players{
		playerStatusElement := playerStatus.GetOne((*player))
		fmt.Printf("k :%d playerStatus %+v",k,playerStatusElement)
		zlib.MyPrint()
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		if playerStatusElement.Status == PlayerStatusNotExist{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			zlib.ExitPrint("sign err, Status:","PlayerStatusSuccess")
		}else if playerStatusElement.Status == PlayerStatusSign{
			zlib.ExitPrint("sign err, Status:","PlayerStatusSign")
		}
		playerStatusElementMap[(*player).Id] = playerStatusElement
	}
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

	expire := now + rule.MatchTimeout
	groupId := queueSign.GetGroupIncId()
	for _,player := range players{
		newPlayerStatusElement := playerStatusElementMap[(*player).Id]
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = playerWeight
		newPlayerStatusElement.SignTimeout = expire

		queueSignElement := QueueSignElement{
			PlayerId: (*player).Id,
			GroupId:groupId,
			Timeout:expire,
			GroupPerson:len(players),
			GroupWeight: playerWeight,
			MatchTimes: 0,
		}
		//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
		queueSign.PushOne(queueSignElement,rule)
		playerStatus.upInfo( playerStatusElementMap[(*player).Id],newPlayerStatusElement)
	}

}
//进程结束，要清理一些数据
func processEndCleanData(){
	ruleConfig := NewRuleConfig()
	if len(ruleConfig.Data) <= 0 {
		zlib.ExitPrint("data len is 0 ")
	}
	for _,v := range ruleConfig.Data{
		queueSign.delAllPlayers(v)
		playerStatus.delAllPlayers()

	}

}

func (gamematch *Gamematch)  StartAllQueueMatching(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	zlib.MyPrint("StartAllQueueMatching , queue total num : ",queueLen)
	if queueLen <=0 {
		zlib.MyPrint(" no need")
	}

	for _,rule := range queueList{
		//zlib.ExitPrint(queueName)
		gamematch.queueMatchingWeight(rule  )
	}
}

func checkCancel(player Player){
	//status := getPlayerStatus(player)
	//if status != PlayerStatusWait{
	//
	//}
}
//使用有序集合，支持权重
func (gamematch *Gamematch) queueMatchingWeight(rule Rule){
	//successPlayers := make( map[int]QueueSignElement )//保存已匹配成功的玩家
	//ScoreMax := rule.PlayerWeight.ScoreMax
	//ScoreMin := rule.PlayerWeight.ScoreMin

	ScoreMax := 10
	ScoreMin := 0

	for{
		//先做最精细的匹配，也就是10个档中，每个档都做匹配
		successPlayers,isEmpty := gamematch.matchingWeight(ScoreMin,ScoreMax,1,rule)
		if isEmpty == 1{
			gamematch.matchingSleep(1)
			continue
		}
		if len(successPlayers) > 0 {

			continue
		}
		zlib.ExitPrint(-400)
		//做粗略匹配，加大匹配权重范围
		//1-2,1-3,1-4,1-5 ... 加大搜索范围，直到1-10，所有元素
		//i:=ScoreMin+1, 因为1-1 在上面已经搜索过了，跳过1-1
		//successPlayers2,isEmpty2 := myFor(ScoreMin,ScoreMax,2,rule)
		//if isEmpty2 == 1{
		//	gamematch.matchingSleep(1)
		//	continue
		//}
		//
		//queueSuccess.PushOne(successPlayers,rule)
		//for _,v := range successPlayers {
		//	playerStatus.upStatus(v.PlayerId,PlayerStatusSuccess)
		//}
	}





	//return successPlayers,1
}

func (gamematch *Gamematch)getLock(){
	zlib.MyPrint(" get lock...")
	queueSign.Mutex.Lock()
}

func (gamematch *Gamematch)unLock(){
	zlib.MyPrint(" un lock...")
	queueSign.Mutex.Unlock()
}

func (gamematch *Gamematch)matchingWeight(start int ,end int ,flag int,rule Rule)(successPlayers map[int]QueueSignElement,isEmpty int){
	zlib.MyPrint("start matching func :")
	gamematch.getLock()
	defer gamematch.unLock()
	//先做最大范围的搜索：所有报名匹配的玩家
	zlib.MyPrint("max range search start:")
	total := queueSign.ZCOUNTTOTAL(rule)
	zlib.MyPrint("sign players total:",total)
	if total <= 0 {
		zlib.MyPrint("total is 0")
		return successPlayers,1
	}
	personCondition := rule.PersonCondition
	if rule.Flag == RuleFlagVS { //组队/对战类
		personCondition = personCondition * 2
	}
	zlib.MyPrint("personCondition:",personCondition)

	if total < personCondition{
		zlib.MyPrint("total <personCondition ")
		return successPlayers,1
	}
	//集合中的：总玩家数<小于>满足条件玩家数 2倍~，没必要再走算法了，直接算就行了
	if total <= personCondition * 2 {
		//一次取出所有玩家，虽然是多取了一些，为避免有些玩家状态出现异常
		queueSign.ZREVRANGEBYSCORETOTAL(rule)
		zlib.ExitPrint(" <= *2 ")
	}
	zlib.MyPrint("max range search end, no match")
	zlib.MyPrint("middle range search start:")
	var tmpMin int
	var tmpMax float32
	for i:=start;i<end;i++{
		if flag == 1{//区域性的
			tmpMin = i
			tmpMax = float32(i) + 0.9999
		}else{//递增，范围更大
			tmpMin = 1
			tmpMax = float32(i) + 0.9999
		}

		cnt := queueSign.ZCOUNT(rule,tmpMin,tmpMax)
		zlib.MyPrint("search ",tmpMin , " - ",tmpMax , "cnt:",cnt)
		//这里应该再 +N 人，一次取出的玩家要冗余几个人，防止符合条件 这些玩家里面有人异常
		//如：状态错误、数据丢失、取消匹配等，还有一种情况是其它队列也在算，这种情况我还得想想
		if cnt < personCondition { //人数不够，放弃本次，计算下一个阶梯
			zlib.MyPrint("cnt < personCondition , continue")
			continue
		}
		//虽然人数够，但是符合条件的人数，也还没超过4倍，再写算法，再做更精细的匹配意义就不大了，按添加时间顺序挑选即可
		if(cnt <= personCondition * 2){
			zlib.MyPrint(" cnt <= personCondition * 2 ")
		}else{//这种情况，是人数至少4倍以上了，那就得写算法，找出距离当前玩家权重最近的对手
			//
		}
		zlib.ExitPrint(-500)
	}
	return successPlayers,isEmpty
}

func  (gamematch *Gamematch) matchingSleep(second time.Duration){
	zlib.MyPrint(" no enough persons...sleep 1 ,loop waiting...")
	time.Sleep(second * time.Second)
}
/*
	3.1
	3.2
	3.3
	4.5
	5.4

 */

func (gamematch *Gamematch) DelAll(){
	keys := redisPrefix + "*"
	redisDelAllByPrefix(keys)
}


//取消- 匹配
func cancelJoin(player Player){

}

func logging(){

}

