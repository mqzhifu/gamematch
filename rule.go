package gamematch

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"zlib"
)

//匹配规则 - 配置 ,像： 附加属性
type Rule struct {
	Id 				int
	AppId			int
	CategoryKey 	string	//分类名，严格来说是：队列的KEY的值，也就是说，该字符串，可以有多级，如：c1_c2_c3,gameId_gameType
	MatchTimeout 	int		//匹配 - 超时时间
	SuccessTimeout	int		//匹配成功后，一直没有人来取，超时时间
	IsSupportGroup 	int		//保留字段，是否支持，以组为单位报名，疑问：所有的类型按说都应该支持这以组为单位的报名
	Flag			int 	//匹配机制-类型，目前两种：1. N（TEAM） VS N（TEAM）  ，  2. N人够了就行(吃鸡模式)
	PersonCondition	int		//触发 满足人数的条件,即匹配成功,必须 flag = 2 ，该变量才有用
	TeamVSPerson	int		//如果是N VS N 的类型，得确定 每个队伍几个人,必须上面的flag = 1 ，该变量才有用,目前最大是5
	TeamVSNumber	int		//保留字段，还没想好，初衷：正常就是一个队伍跟一个队列PK，但是可能会有多队伍互相PK，不一定是N V N
	PlayerWeight	PlayerWeight	//权重，目前是以最小单位：玩家属性，如果是小组/团队，是计算平均值
	GroupPersonMax	int		//玩家以组为单位，一个小组最大报名人数,暂定最大为：5
}

type PlayerWeight struct {
	ScoreMin 	int		//权重值范围：最小值，范围： 1-10
	ScoreMax	int		//权重值范围：最大值，范围： 1-10
	AutoAssign	bool	//当权重值范围内，没有任何玩家，是否接收，自动调度分配，这样能提高匹配成功率
	Formula		string	//属性计算公式，由玩家的N个属性，占比，最终计算出权重值
	Flag		int		//1、计算权重平均的区间的玩家，2、权重越高的匹配越快
}

type GamesMatchConfig struct {
	Id 		int		`json:"id"`
	GamesId    int    `json:"games_id"`    //游戏ID
	Name       string `json:"name"`        //规则名称
	Status     int    `json:"status"`      //规则状态：1上线 2下线 3删除
	MatchCode  string `json:"match_code"`  //匹配代码
	TeamType   int    `json:"team_type"`   //团队类型 1各自为战 2对称团队战
	MaxPlayers int    `json:"max_players"` //匹配最大人数 如团队战代表每个队伍人数
	Rule       string `json:"rule"`        //表达式匹配规则
	Timeout    int    `json:"timeout"`     //匹配超时时间
	Fps        int    `json:"fps"`         //帧率
	SuccessTimeout int `json:"success_timeout"`
}

type RuleConfig struct {
	Data map[int]Rule
	gamematch *Gamematch
}

func (ruleConfig *RuleConfig) getRedisKey()string{
	return redisPrefix + redisSeparation + "rule"
}

func (ruleConfig *RuleConfig) getRedisIncKey()string{
	return redisPrefix + redisSeparation + "rule" + redisSeparation + "inc"
}

//初始化
func   NewRuleConfig (gamematch *Gamematch)(*RuleConfig,error){
	rule := new (RuleConfig)
	rule.Data = make(map[int]Rule)
	//zlib.MyPrint("NewRuleConfig",len(res))
	ruleList := rule.getByEtcd( )
	if len(ruleList) <= 0 {
		return rule,myerr.NewErrorCode(601)
	}

	mylog.Info("rule cnt : ",len(ruleList))
	for _,v := range ruleList{
		mylog.Info("match code : ",v.CategoryKey)
	}
	rule.gamematch = gamematch
	rule.Data = ruleList
	go rule.watchEtcdChange()
	return rule,nil
}

func (ruleConfig *RuleConfig)watchEtcdChange( ){
	prefix := "etcd watching"
	watchChann := myetcd.Watch(RuleEtcdConfigPrefix)
	mylog.Notice(prefix , " , new key : ",RuleEtcdConfigPrefix)
	//watchChann := myetcd.Watch("/testmatch")
	for wresp := range watchChann{
		for _, ev := range wresp.Events{
			action := ev.Type.String()
			key := string(ev.Kv.Key)
			val := string(ev.Kv.Value)
			mylog.Warning(prefix , " chan has event : ",action)
			mylog.Warning(prefix , " key : ",key)
			mylog.Warning(prefix , " val : ",val)
			//zlib.MyPrint(ev.Type.String(), string(ev.Kv.Key), string(ev.Kv.Value))
			//fmt.Printf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)

			matchCode := strings.Replace(key,RuleEtcdConfigPrefix,"",-1)
			matchCode = strings.Trim(matchCode," ")
			matchCode = strings.Trim(matchCode,"/")
			mylog.Warning(prefix , " matchCode : ",matchCode)

			rule,ruleErr := ruleConfig.getByCategory(matchCode)
			//fmt.Printf("%+v",rule)
			if ev.Type.String() == "PUT"{
				mylog.Warning(prefix ," action PUT : ruleErr ,",ruleErr)
				if ruleErr != nil{
					mylog.Error("etcd event DELETE rule ,but Value no match rule , maybe add new rule~")
				}else{
					ruleConfig.gamematch.closeOneRuleDemonRoutine(rule.Id)
					ruleConfig.delOne(rule.Id)
				}
				newRule,err := ruleConfig.parseOneConfigByEtcd(key,val)
				if err != nil{
					mylog.Error("etcd monitor:",err.Error())
				}else{
					//新添加一条rule
					ruleConfig.Data[newRule.Id] = newRule
					ruleConfig.gamematch.startOneRuleDomon(newRule)
					//ruleConfig.gamematch.HttpdRuleState[newRule.Id] = HTTPD_RULE_STATE_OK
					//mySleepSecond(3,"testtest")
					//zlib.ExitPrint(111111)
				}
			}else if ev.Type.String() == "DELETE"{
				mylog.Warning(prefix ," dvent = DELETE : ruleErr ,",ruleErr)
				if ruleErr != nil{
					mylog.Error("etcd event DELETE rule ,but Value no match rule!!!")
				}else{
					ruleConfig.gamematch.closeOneRuleDemonRoutine(rule.Id)
					ruleConfig.delOne(rule.Id)
				}
			}
		}
	}
	//zlib.ExitPrint(watchChann)
}

func (ruleConfig *RuleConfig)delOne(ruleId int){
	_, ok  := ruleConfig.GetById(ruleId)
	if !ok {
		mylog.Error("ruleConfig.GetByI is empty~")
		return
	}
	delete(ruleConfig.Data,ruleId)

	queueSign := ruleConfig.gamematch.getContainerSignByRuleId(ruleId)
	queueSign.delOneRule()

	push := ruleConfig.gamematch.getContainerPushByRuleId(ruleId)
	push.delOneRule()

	queueSuccess := ruleConfig.gamematch.getContainerSuccessByRuleId(ruleId)
	queueSuccess.delOneRule()

	playerIds := playerStatus.getOneRuleAllPlayer(ruleId)
	for _,playerId := range playerIds{
		playerStatus.delOneById(zlib.Atoi(playerId))
	}
}

func (ruleConfig *RuleConfig)parseOneConfigByEtcd(k string ,v string)(rule Rule,err error){
	if k == ""{
		return rule,myerr.NewErrorCode(620)
	}
	if v == ""{
		return rule,myerr.NewErrorCode(621)
	}
	//zlib.MyPrint(v)
	gamesMatchConfig := GamesMatchConfig{}
	err = json.Unmarshal( []byte(v), & gamesMatchConfig)
	if err != nil{
		msg := myerr.MakeOneStringReplace(err.Error())
		myerr.NewErrorCodeReplace(622,msg)
		return rule,err
	}

	if gamesMatchConfig.Status != RuleStatusOnline{
		return rule,myerr.NewErrorCode(624)
	}

	playerWeightRow := PlayerWeight{}
	rule  = Rule{
		Id: gamesMatchConfig.Id,
		AppId: gamesMatchConfig.Id,
		CategoryKey : gamesMatchConfig.MatchCode,
		MatchTimeout: gamesMatchConfig.Timeout,
		SuccessTimeout: gamesMatchConfig.SuccessTimeout,
		IsSupportGroup: 1,

		Flag:gamesMatchConfig.TeamType,
		TeamVSPerson:gamesMatchConfig.MaxPlayers / 2,
		PersonCondition: gamesMatchConfig.MaxPlayers,
		GroupPersonMax : gamesMatchConfig.MaxPlayers / 2,
		PlayerWeight: playerWeightRow,
	}

	return rule,err
}

func (ruleConfig *RuleConfig)getByEtcd()  map[int]Rule{
	etcdRuleList := myetcd.GetListByPrefix(RuleEtcdConfigPrefix)
	ruleList := make(map[int]Rule)
	if len(etcdRuleList) == 0{
		return ruleList
	}
	//i := 1
	for k,v := range etcdRuleList{
		//matchCode := strings.Replace(k,RuleEtcdConfigPrefix,"",-1)
		//matchCode = strings.Trim(matchCode," ")
		//matchCode = strings.Trim(matchCode,"/")
		//if k == ""{
		//	myerr.NewErrorCode(620)
		//	continue
		//}
		//if v == ""{
		//	myerr.NewErrorCode(621)
		//	continue
		//}
		////zlib.MyPrint(v)
		//gamesMatchConfig := GamesMatchConfig{}
		//err := json.Unmarshal( []byte(v), & gamesMatchConfig)
		ruleRow,err := ruleConfig.parseOneConfigByEtcd(k,v)
		if err != nil{
			//msg := myerr.MakeOneStringReplace(err.Error())
			//myerr.NewErrorCodeReplace(622,msg)
			continue
		}
		//fmt.Printf("%+v",gamesMatchConfig)
		//playerWeightRow := PlayerWeight{}
		//ruleRow := Rule{
		//	Id: gamesMatchConfig.Id,
		//	AppId: gamesMatchConfig.Id,
		//	CategoryKey : gamesMatchConfig.MatchCode,
		//	MatchTimeout: gamesMatchConfig.Timeout,
		//	SuccessTimeout: gamesMatchConfig.SuccessTimeout,
		//	IsSupportGroup: 1,
		//
		//	Flag:gamesMatchConfig.TeamType,
		//	TeamVSPerson:gamesMatchConfig.MaxPlayers / 2,
		//	PersonCondition: gamesMatchConfig.MaxPlayers,
		//	GroupPersonMax : gamesMatchConfig.MaxPlayers / 2,
		//	PlayerWeight: playerWeightRow,
		//}
		//fmt.Printf("%+v",gamesMatchConfig)
		//zlib.ExitPrint(gamesMatchConfig)
		_,err =  ruleConfig.CheckRuleByElement(ruleRow)
		//zlib.ExitPrint(err)
		if err != nil{
			//myerr.NewErrorCode(621)
			continue
		}
		ruleList[ruleRow.Id] = ruleRow
		//i++
	}
	//zlib.ExitPrint(ruleList)
	return ruleList

}
//func (ruleConfig *RuleConfig)getByRedis(){
//	res,errs := redis.StringMap(myredis.RedisDo("HGETALL",ruleConfig.getRedisKey()))
//	if errs != nil{
//		msg := make(map[int]string)
//		msg[0] = errs.Error()
//		return obj,myerr.NewErrorCodeReplace(600,msg)
//	}
//}
func (ruleConfig *RuleConfig)strToStruct(redisStr string)Rule{

	strArr := strings.Split(redisStr,separation)
	element := Rule{
		Id 				:	zlib.Atoi(strArr[0]),
		AppId 			:	zlib.Atoi(strArr[1]),
		CategoryKey 	:	strArr[2],
		MatchTimeout 	:	zlib.Atoi(strArr[3]),
		SuccessTimeout 	:	zlib.Atoi(strArr[4]),
		PersonCondition :	zlib.Atoi(strArr[5]),
		IsSupportGroup 	:	zlib.Atoi(strArr[6]),
		Flag 			:	zlib.Atoi(strArr[7]),
		TeamVSPerson 		:	zlib.Atoi(strArr[8]),
		GroupPersonMax : zlib.Atoi(strArr[9]),
		//WeightRule 		:	strArr[0],
	}
	return element
}

func (ruleConfig *RuleConfig)structToStr(rule Rule)string{
	//groupPersonMax	int		//玩家以组为单位，一个小组最大报名人数,暂定最大为：5

	str := strconv.Itoa(rule.Id) + separation +
		strconv.Itoa(rule.AppId) + separation +
		rule.CategoryKey + separation +
		strconv.Itoa(rule.MatchTimeout) + separation +
		strconv.Itoa(rule.SuccessTimeout) + separation +
		strconv.Itoa(rule.PersonCondition) + separation +
		strconv.Itoa(rule.IsSupportGroup) + separation +
		strconv.Itoa(rule.Flag) + separation +
		strconv.Itoa(rule.TeamVSPerson) + separation +
		strconv.Itoa(rule.GroupPersonMax)

		//rule.WeightRule + separation
	return str
}


func (ruleConfig *RuleConfig) GetById(id int ) (Rule,bool){
	if id == 0{
		return Rule{},false
	}
	rule,ok := ruleConfig.Data[id]
	return rule,ok
}

func (ruleConfig *RuleConfig) getAll()map[int]Rule{
	return ruleConfig.Data
}

func  (ruleConfig *RuleConfig) getByCategory(category string) (rule Rule ,err error){
	if category == ""{
		return rule,myerr.NewErrorCode(450)
	}

	for _,rule := range ruleConfig.getAll(){
		if  rule.CategoryKey == category{
			return rule,nil
		}
	}
	return rule,myerr.NewErrorCode(451)
}

func (ruleConfig *RuleConfig) getIncId( ) (int){
	key := ruleConfig.getRedisIncKey()
	res,_ := redis.Int(myredis.RedisDo("INCR",key))
	return res
}

func (ruleConfig *RuleConfig) AddOne(rule Rule)(bool,error){
	checkRs,errs := ruleConfig.CheckRuleByElement(rule)
	if !checkRs{
		return false,errs
	}
	key := ruleConfig.getRedisKey()
	//id := ruleConfig.getIncId()
	ruleStr := ruleConfig.structToStr(rule)
	mylog.Info("ruleStr : ",ruleStr, " rule struct : ",rule)
	_ ,errs = redis.Int( myredis.RedisDo("hset",redis.Args{}.Add(key).Add(rule.Id).Add(ruleStr)...))
	if errs != nil{
		return false,myerr.NewErrorCode(603)
	}
	return true,nil
}

func (ruleConfig *RuleConfig) CheckRuleByElement(rule Rule)(bool,error){
	if rule.Id <= 0{
		return false,myerr.NewErrorCode(604)
	}
	if rule.AppId <= 0{
		return false,myerr.NewErrorCode(605)
	}
	if rule.CategoryKey == ""{
		return false,myerr.NewErrorCode(616)
	}
	if rule.Flag <= 0{
		return false,myerr.NewErrorCode(606)
	}
	if rule.Flag == RuleFlagTeamVS{
		if rule.TeamVSPerson <= 0{
			return false,myerr.NewErrorCode(608)
		}

		if rule.TeamVSPerson > RuleTeamVSPersonMax{
			return false,myerr.NewErrorCodeReplace(609,myerr.MakeOneStringReplace(strconv.Itoa(RuleTeamVSPersonMax)))
		}
		//TeamVSPerson	int		//如果是N VS N 的类型，得确定 每个队伍几个人,必须上面的flag = 1 ，该变量才有用,目前最大是5
	}else if rule.Flag == RuleFlagCollectPerson{
		if rule.PersonCondition <= 0{
			return false,myerr.NewErrorCode(610)
		}

		if rule.PersonCondition > RulePersonConditionMax{
			return false,myerr.NewErrorCodeReplace(611,myerr.MakeOneStringReplace(strconv.Itoa(RuleTeamVSPersonMax)))
		}
	}else{
		return false,myerr.NewErrorCode(607)
	}
	if rule.MatchTimeout < RuleMatchTimeoutMin || rule.MatchTimeout > RuleMatchTimeoutMax{
		msg := make(map[int]string)
		msg[0] = strconv.Itoa(RuleMatchTimeoutMin)
		msg[1] = strconv.Itoa(RuleMatchTimeoutMax)
		return false,myerr.NewErrorCodeReplace(612,msg)
	}

	if rule.SuccessTimeout < RuleSuccessTimeoutMin || rule.SuccessTimeout > RuleSuccessTimeoutMax{
		msg := make(map[int]string)
		msg[0] = strconv.Itoa(RuleSuccessTimeoutMin)
		msg[1] = strconv.Itoa(RuleSuccessTimeoutMax)
		return false,myerr.NewErrorCodeReplace(613,msg)
	}

	if rule.GroupPersonMax <= 0{
		return false,myerr.NewErrorCode(614)
	}

	if rule.GroupPersonMax > RuleGroupPersonMax{
		return false,myerr.NewErrorCodeReplace(615,myerr.MakeOneStringReplace(strconv.Itoa(RuleGroupPersonMax)))
	}

	//PlayerWeight	PlayerWeight	//权重，目前是以最小单位：玩家属性，如果是小组/团队，是计算平均值
	return true,nil
}

func (ruleConfig *RuleConfig) CheckRuleById(ruleId int)(bool,error){
	rule , ok := ruleConfig.GetById(ruleId)
	if !ok {
		msg := make(map[int]string)
		msg[0] = strconv.Itoa(ruleId)
		return false,myerr.NewErrorCodeReplace(602,msg)
	}

	return ruleConfig.CheckRuleByElement(rule)


}