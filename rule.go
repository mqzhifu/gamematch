package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
	"strconv"
	"strings"
)
const(
	RuleFlagTeamVS = 1
	RuleFlagCollectPerson = 2
	RuleGroupPersonMax = 5
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
	groupPersonMax	int		//玩家以组为单位，一个小组最大报名人数,暂定最大为：5
}

type PlayerWeight struct {
	ScoreMin 	int		//权重值范围：最小值，范围： 1-10
	ScoreMax	int		//权重值范围：最大值，范围： 1-10
	AutoAssign	bool	//当权重值范围内，没有任何玩家，是否接收，自动调度分配，这样能提高匹配成功率
	Formula		string	//属性计算公式，由玩家的N个属性，占比，最终计算出权重值
	Flag		int		//1、计算权重平均的区间的玩家，2、权重越高的匹配越快
}

type RuleConfig struct {
	Data map[int]Rule
}

func (ruleConfig *RuleConfig) getRedisKey()string{
	return redisPrefix + redisSeparation + "rule"
}

func (ruleConfig *RuleConfig) getRedisIncKey()string{
	return redisPrefix + redisSeparation + "rule" + redisSeparation + "inc"
}

func   NewRuleConfig ()*RuleConfig{
	obj := new (RuleConfig)
	res,err := redis.StringMap(redisDo("HGETALL",obj.getRedisKey()))
	if err != nil{
		zlib.ExitPrint("rule confg HGETALL err ,",err.Error())
	}
	obj.Data = make(map[int]Rule)
	//zlib.MyPrint("NewRuleConfig",len(res))
	if AddRuleFlag == 0{
		if len(res) <= 0 {
			zlib.ExitPrint("RuleConfig is null")
		}
	}


	//mapKey := 0
	for k,v := range res{
		key := zlib.Atoi(k)
		myRule := obj.strToStruct(v)
		obj.Data[key] = myRule
	}
	//zlib.ExitPrint(obj.Data)
	return obj
}

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
		groupPersonMax : zlib.Atoi(strArr[9]),
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
		strconv.Itoa(rule.groupPersonMax)

		//rule.WeightRule + separation
	return str
}


func (ruleConfig *RuleConfig) GetById(id int ) (Rule,bool){
	rule,ok := ruleConfig.Data[id]
	return rule,ok
}

func (ruleConfig *RuleConfig) getAll()map[int]Rule{
	return ruleConfig.Data
}

func (ruleConfig *RuleConfig) getIncId( ) (int){
	key := ruleConfig.getRedisIncKey()
	res,_ := redis.Int(redisDo("INCR",key))
	return res
}

func (ruleConfig *RuleConfig) AddOne(rule Rule)int{
	if rule.groupPersonMax > RuleGroupPersonMax {
		zlib.ExitPrint("小组最大人数：",RuleGroupPersonMax)
	}

	if rule.groupPersonMax <= 0 {
		zlib.ExitPrint("小组最大人数不能 <= 0 ")
	}
	ruleOld ,ok := ruleConfig.Data[rule.Id]
	if ok{
		zlib.MyPrint("rule confg add one err,id has exist",ruleOld.Id)
	}
	key := ruleConfig.getRedisKey()
	//id := ruleConfig.getIncId()
	ruleStr := ruleConfig.structToStr(rule)
	zlib.MyPrint("ruleStr : ",ruleStr, " rule struct : ",rule)
	res ,err := redis.Int( redisDo("hset",redis.Args{}.Add(key).Add(rule.Id).Add(ruleStr)...))
	if err != nil{
		zlib.ExitPrint(" addRuleOne redis err ")
	}
	return res
}
