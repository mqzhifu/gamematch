package gamematch

import (
	"zlib"
)

var AddRuleFlag = 0

func Test(){
	//实例化-<日志>-组件
	logOption := zlib.LogOption{
		OutFilePath : LOG_BASE_DIR,
		OutFileName: LOG_FILE_NAME,
		Level : LOG_LEVEL,
		Target : LOG_TARGET,
	}
	mylog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}
	//实例化-<redis>-组件
	redisOption := zlib.RedisOption{
		Host: "127.0.0.1",
		Port: "6379",
		Log: mylog,
	}
	myredis , errs := zlib.NewRedisConnPool(redisOption)
	if errs != nil{
		zlib.ExitPrint("new redis err",errs.Error())
	}
	//实例化-<etcd>-组件
	url := "http://39.106.65.76:1234/system/etcd/cluster1/list/"
	etcdOption := zlib.EtcdOption{
		FindEtcdUrl: url,
		Log : mylog,
	}
	myetcd,errs := zlib.NewMyEtcdSdk(etcdOption)
	if errs != nil{
		zlib.ExitPrint("NewMyEtcdSdk err",errs.Error())
	}
	//实例化-<服务发现>-组件
	serviceOption := zlib.ServiceOption{
		Etcd: myetcd,
		Log: mylog,
		Prefix: SERVICE_PREFIX,
	}

	myHost := "192.168.31.148"
	//myHost := "192.168.192.170"
	myPort := "5678"

	myservice := zlib.NewService(serviceOption)
	//将自己注册成一个服务
	myservice.RegOne(SERVICE_MATCH_NAME,myHost+":"+myPort)
	//最后，终于，实例化：匹配机制
	myGamematch,_ := NewGamematch(mylog,myredis,myservice,myetcd)
	myHttpdOption :=  HttpdOption{
		Host: myHost,
		Port: myPort,
		Log : mylog,
	}
	go myGamematch.startHttpd(myHttpdOption)
	go myGamematch.DemonAll()
	deadLoopBlock(1 , " main ")

	//TestAddRuleData(*myGamematch)
	//TestSign(myGamematch,999)
	//TestMatching(*myGamematch)
	//TestUnitCase(*myGamematch)
	//zlib.ExitPrint(-999)
}

func clear(myGamematch Gamematch){
	myGamematch.DelAll()
	zlib.ExitPrint("clear all done. ")
}
func TestAddRuleData(myGamematch Gamematch){
	rule1 := Rule{
		Id: 1,
		AppId: 3,
		CategoryKey: "RuleFlagCollectPerson",
		MatchTimeout: 7,
		SuccessTimeout: 60,
		IsSupportGroup: 1,
		Flag: 2,
		PersonCondition: 4,
		GroupPersonMax:5,
		//TeamPerson: 5,
		PlayerWeight: PlayerWeight{
			ScoreMin:-1,
			ScoreMax:-1,
			AutoAssign:true,
			Formula:"",
			Flag:1,
		},
	}

	rule2 := Rule{
		Id: 2,
		AppId: 4,
		CategoryKey: "RuleFlagTeamVS",
		MatchTimeout: 10,
		SuccessTimeout: 70,
		IsSupportGroup: 1,
		Flag: 1,
		PersonCondition: 5,
		TeamVSPerson:5,
		GroupPersonMax:5,
		//TeamPerson: 5,
		PlayerWeight: PlayerWeight{
			ScoreMin:-1,
			ScoreMax:-1,
			AutoAssign:true,
			Formula:"",
			Flag:1,
		},
	}

	myGamematch.RuleConfig.AddOne(rule1)
	myGamematch.RuleConfig.AddOne(rule2)
}

func TestUnitCase(myGamematch Gamematch){
	//rule := 2
	//delOneRule(myGamematch,rule)
	//TestSign(myGamematch,rule)
	myGamematch.DemonAll()
}

//func delOneRule(myGamematch Gamematch ,ruleId int){
//	queueSign := myGamematch.getContainerSignByRuleId(ruleId)
//	queueSign.delOneRule()
//
//	queueSuccess := myGamematch.getContainerSuccessByRuleId(ruleId)
//	queueSuccess.delOneRule()
//
//	playerStatus.delAllPlayers()
//	mylog.Notice("testSignDel finish.")
//}

func getOneRandomPlayerUid()int{
	return zlib.GetRandIntNumRange(1000,9999)
}


//func TestSign(myGamematch *Gamematch,ruleId int){
//	//ruleId := 2
//	//playerIdInc := 1
//	signRuleArr := []int{1,5,4,5,3,3,3,2,2,1,1,1,1,5,4,4}
//	for _,playerNumMax := range signRuleArr{
//		var playerStructArr []Player
//		for i:=0;i<playerNumMax;i++{
//			//player := Player{Id:playerIdInc}
//			playerUid := getOneRandomPlayerUid()
//			player := Player{Id:playerUid}
//
//			playerStructArr = append(playerStructArr,player)
//			//playerIdInc++
//		}
//		//rule ,_ := myGamematch.RuleConfig.GetById(ruleId)
//		customProp := "im_customProp"
//		myGamematch.Sign(ruleId,9999,customProp,playerStructArr , "im_addition")
//	}
//}

