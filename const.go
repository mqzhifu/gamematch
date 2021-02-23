package gamematch

import "zlib"

/*
	匹配类型 - 规则
	1. N人匹配 ，只要满足N人，即成功
	2. N人匹配 ，划分为2个队，A队满足N人，B队满足N人，即成功

	权重		：根据某个用户上的某个特定属性值，计算出权重，优先匹配
	组		：ABC是一个组，一起参与匹配，那这3个人匹配的时候是不能分开的
	游戏属性	：游戏类型，也可以有子类型，如：不同的赛制。最终其实是分队列。不同的游戏忏悔分类，有不同的分类
*/
//rule规格配置表
const(
	RuleFlagTeamVS 			= 1		//对战类型
	RuleFlagCollectPerson 	= 2		//满足人数即可
	RuleGroupPersonMax 		= 5		//一个小组允许最大人数
	RuleTeamVSPersonMax		= 10  //组队互相PK，每个队最多人数
	RulePersonConditionMax 	= 100	//N人组团，最大人数

	RuleMatchTimeoutMax 	= 400	//报名，最大超时时间
	RuleMatchTimeoutMin 	= 3		//报名，最短切尔西时间
	RuleSuccessTimeoutMax 	= 600	//匹配成功后，最大超时时间
	RuleSuccessTimeoutMin 	= 60	//匹配成功后，最短超时时间

	RuleEtcdConfigPrefix = "/v1/conf/matches/"	//etcd中  ， 存放 rule  集合的前缀
)
//微服务
const(
	SERVICE_PREFIX = "/v1/service"		//微服务前缀

	SERVICE_MSG_SERVER		="msgServer"
	SERVICE_MATCH_NAME		="gamematch"
)
//公共
const (
	LOG_BASE_DIR 			= "/data/www/golang/src/logs"
	LOG_FILE_NAME			= "gamematch"
	LOG_LEVEL				= zlib.LEVEL_ALL
	LOG_TARGET 				= 15

	separation 				= "#"		//redis 内容-字符串分隔符
	PayloadSeparation		= "%"		//push时的内容，缓存进redis时
	redisSeparation 		= "_"		//redis key 分隔符
	IdsSeparation 			= ","		//多个ID 分隔符
	redisPrefix 			= "match"	//整个服务的，redis 前缀
	PlayerMatchingMaxTimes 	= 3			//一个玩家，参与匹配机制的最大次数，超过这个次数，证明不用再匹配了，目前没用上，目前使用的还是绝对的超时时间为准

)
//匹配-筛选策略
const (
	FilterFlagAll     =1	//全匹配
	FilterFlagBlock	  =2	//块-匹配
	FilterFlagBlockInc=3	//递增块切尔西
)

//玩家状态
const (
	PlayerStatusNotExist = 1//redis中还没有该玩家信息
	PlayerStatusSign = 2	//已报名，等待匹配
	PlayerStatusSuccess = 3	//匹配成功，等待拿走
)

//HTTP推送
const(
	PushCategorySignTimeout 	= 1	//报名超时
	PushCategorySuccess 		= 2	//匹配成功
	PushCategorySuccessTimeout	= 3	//匹配成功超时

	PushStatusWait		= 1	//等待推送
	PushStatusRetry 	= 2	//已推送过，但失败了，等待重试
)


