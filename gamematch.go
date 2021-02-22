package gamematch

import (
	"os"
	"strconv"
	"time"
	"zlib"
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
//报名成功后，返回的结果
type SignSuccessReturnData struct {
	RuleId 		int
	GroupId		int
	PlayerIds 	string
}

type SignHttpData struct {
	GroupId int
	CustomProp	string
	PlayersList []Player
	RuleId int
	Addition string
}

type SignCancelHttpData struct {
	PlayerId	int
	GroupId 	int
	RuleId int
}


//玩家结构体，目前暂定就一个ID，其它的字段值，后期得加，主要是用于权重计算
type Player struct {
	Id	int
}

var mylog 		*zlib.Log
var myerr 		*zlib.Err
var myredis 	*zlib.MyRedis
var myservice	*zlib.Service
var myetcd		*zlib.MyEtcd


var playerStatus 	*PlayerStatus	//玩家状态类


//var logFlushFileDir string
//var logLevel int
//var logTarget int


//var configCenter	*configcenter.Configer
var roomServiceHost string
//var myEtcd *MyEtcd
//var myService *Service

func NewGamematch(zlog *zlib.Log  ,zredis *zlib.MyRedis,zservice *zlib.Service ,zetcd *zlib.MyEtcd )(gamematch *Gamematch ,errs error){
	mylog = zlog
	myredis = zredis
	myservice = zservice
	myetcd = zetcd
	//初始化-错误码
	container := getErrorCode()
	mylog.Info("getErrorCode len : ",len(container))
	if   len(container) == 0{
		//return gamematch,err.NewErrorCode(900)
		zlib.ExitPrint("getErrorCode len  = 0")
	}
	//初始化-错误/异常 类
	myerr = zlib.NewErr(mylog,container)
	gamematch = new (Gamematch)

	gamematch.RuleConfig, errs = NewRuleConfig()
	if errs != nil{
		zlib.ExitPrint(" New Redis conn err.",errs.Error())
	}

	gamematch.containerPush		= make(map[int]*Push)
	gamematch.containerSign 	= make(	map[int]*QueueSign )	//容器 报名类
	gamematch.containerSuccess = make(	map[int]*QueueSuccess)
	gamematch.containerMatch	= make( 	map[int]*Match)


	mylog.Info("init container....")
	//实例化容器
	for _,rule := range gamematch.RuleConfig.getAll(){
		gamematch.containerPush[rule.Id] = NewPush(rule)
		gamematch.containerSign[rule.Id] = NewQueueSign(rule)
		gamematch.containerSuccess[rule.Id] = NewQueueSuccess(rule,gamematch.containerPush[rule.Id])
		gamematch.containerMatch[rule.Id] = NewMatch(rule,gamematch.containerSign[rule.Id],gamematch.containerSuccess[rule.Id],gamematch.containerPush[rule.Id])

	}
	playerStatus = NewPlayerStatus()

	return gamematch,nil
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
//报名 - 加入匹配队列
//此方法，得有前置条件：验证所有参数是否正确，因为使用者为http请求，数据的验证交由HTTP层做处理，如果是非HTTP，要验证一下
func (gamematch *Gamematch) Sign(ruleId int ,outGroupId int, customProp string, players []Player ,addition string )( signSuccessReturnData SignSuccessReturnData , err error){
	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		return signSuccessReturnData,myerr.NewErrorCode(400)
	}
	lenPlayers := len(players)
	//if lenPlayers == 0{
	//	return myerr.NewErrorCode(401)
	//}
	queueSign := gamematch.getContainerSignByRuleId(ruleId)

	//groupsTotal := queueSign.getAllGroupsWeightCnt()	//报名 小组总数
	//playersTotal := queueSign.getAllPlayersCnt()	//报名 玩家总数
	//mylog.Info(" action :  Sign , players : " + strconv.Itoa(lenPlayers) +" ,queue cnt : groupsTotal",groupsTotal ," , playersTotal",playersTotal)
	mylog.Info("new sign :[ruleId : ",ruleId,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	queueSign.Log.Info("new sign :[ruleId : " ,  ruleId   ,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	now := zlib.GetNowTimeSecondToInt()

	queueSign.Log.Info("start check player status :")
	//检查，所有玩家的状态
	playerStatusElementMap := make( map[int]PlayerStatusElement )
	for _,player := range players{
		playerStatusElement := playerStatus.GetOne(player)
		//fmt.Printf("playerStatus : %+v",playerStatusElement)
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		queueSign.Log.Info("player ( ",player.Id , " ) statusElement :  status = ", playerStatusElement.Status)
		if playerStatusElement.Status == PlayerStatusNotExist{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			queueSign.Log.Error(" player status = PlayerStatusSuccess ,finish.")
			msg := make(map[int]string)
			msg[0] = strconv.Itoa(player.Id)
			return signSuccessReturnData,myerr.NewErrorCodeReplace(403,msg)
		}else if playerStatusElement.Status == PlayerStatusSign{
			isTimeout := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{//未超时
				queueSign.Log.Error(" player status = matching...  not timeout")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return signSuccessReturnData,myerr.NewErrorCodeReplace(402,msg)
			}else{
				queueSign.Log.Error(" player status is timeout ,but not clear , wait a moment!!!")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return signSuccessReturnData,myerr.NewErrorCodeReplace(402,msg)
			}
			//groupInfo := queueSign.getGroupElementById(playerStatusElement.GroupId)
			//if groupInfo.Person > 1{
			//	msg := make(map[int]string)
			//	msg[0] = strconv.Itoa(player.Id)
			//	return myerr.NewErrorCodeReplace(405,msg)
			//}
			//mylog.Notice("players sign timeout : delete one group... , gid : ",playerStatusElement.GroupId)
			//queueSign.delOneRuleOneGroup(playerStatusElement.GroupId,1)
			////删除一个组，会连带着组里所有玩家的状态信息也被删除，这里再重新拉取一下，保证数据完整性
			//playerStatusElement = playerStatus.GetOne(player)
		}

		playerStatusElementMap[player.Id] = playerStatusElement
	}
	mylog.Info("finish check player status.")
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
	//这里有偷个懒，还是用外部的groupId , 不想再给redis加 groupId映射outGroupId了
	mylog.Notice(" outGroupId replace groupId :",outGroupId,group.Id)
	group.Id = outGroupId
	group.Players = players
	group.SignTimeout = expire
	group.Person = len(players)
	group.Weight = playerWeight
	group.OutGroupId = outGroupId
	group.Addition = addition
	group.CustomProp =  customProp
	group.MatchCode = rule.CategoryKey
	mylog.Info("newGroupId : ",group.Id , "player/group weight : " ,playerWeight ," now : ",now ," expire : ",expire )
	queueSign.Log.Info("newGroupId : ",group.Id , "player/group weight : " ,playerWeight ," now : ",now ," expire : ",expire)
	//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
	//queueSign.Mutex.Lock()
	queueSign.AddOne(group)
	//defer queueSign.Mutex.Unlock()

	playerIds := ""
	for _,player := range players{
		newPlayerStatusElement := playerStatusElementMap[player.Id]
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = playerWeight
		newPlayerStatusElement.SignTimeout = expire
		newPlayerStatusElement.GroupId = group.Id

		queueSign.Log.Info("upInfo status , old :",playerStatusElementMap[player.Id].Status ,  " , new : ",newPlayerStatusElement.Status )
		playerStatus.upInfo( playerStatusElementMap[player.Id],newPlayerStatusElement)

		playerIds += strconv.Itoa(player.Id) + ","
	}

	queueSign.Log.Info("  finish : success players : ",len(players))
	mylog.Info("  finish : success players : ",len(players))

	signSuccessReturnData = SignSuccessReturnData{
		RuleId: ruleId,
		GroupId: outGroupId,
		PlayerIds: playerIds,
	}

	return signSuccessReturnData,nil
}

func (gamematch *Gamematch)CheckHttpRuleAddOneData(jsonDataMap map[string]string)(data SignCancelHttpData,errs error){
	//matchCode,ok := jsonDataMap["matchCode"]
	//if !ok {
	//	return data,myerr.NewErrorCode(450)
	//}
	//if matchCode == ""{
	//	return data,myerr.NewErrorCode(450)
	//}
	//
	//checkCodeRs := false
	//ruleId := 0
	//for _,rule := range gamematch.RuleConfig.getAll(){
	//	if  rule.CategoryKey == matchCode{
	//		ruleId = rule.Id
	//		checkCodeRs = true
	//		break
	//	}
	//}
	//if !checkCodeRs{
	//	return data,myerr.NewErrorCode(451)
	//}
	//
	//playerId,ok := jsonDataMap["playerId"]
	//if !ok {
	//	return data,myerr.NewErrorCode(452)
	//}
	//if playerId == ""{
	//	return data,myerr.NewErrorCode(452)
	//}
	//
	//data.PlayerId =  zlib.Atoi(playerId)
	//data.RuleId = ruleId
	return data,nil
}

func (gamematch *Gamematch)CheckHttpSignCancelData(jsonDataMap map[string]string)(data SignCancelHttpData,errs error){
	matchCode,ok := jsonDataMap["matchCode"]
	if !ok {
		return data,myerr.NewErrorCode(450)
	}
	if matchCode == ""{
		return data,myerr.NewErrorCode(450)
	}

	checkCodeRs := false
	ruleId := 0
	for _,rule := range gamematch.RuleConfig.getAll(){
		if  rule.CategoryKey == matchCode{
			ruleId = rule.Id
			checkCodeRs = true
			break
		}
	}
	if !checkCodeRs{
		return data,myerr.NewErrorCode(451)
	}

	groupId ,ok := jsonDataMap["groupId"]
	if !ok || groupId == ""{
		playerId,ok := jsonDataMap["playerId"]
		if !ok || playerId == "" {
			return data,myerr.NewErrorCode(457)
		}
		data.PlayerId = zlib.Atoi(playerId)
		data.GroupId =  0
	}else{
		data.PlayerId = 0
		data.GroupId =  zlib.Atoi(groupId)
	}

	data.RuleId = ruleId
	return data,nil

}
func (gamematch *Gamematch)CheckHttpSignData(jsonDataMap map[string]interface{})(data SignHttpData,errs error){
	mylog.Debug("CheckHttpSignData",jsonDataMap)
	matchCodeStr,ok := jsonDataMap["matchCode"]
	if !ok {
		return data,myerr.NewErrorCode(450)
	}
	matchCode := matchCodeStr.(string)
	if matchCode == ""{
		return data,myerr.NewErrorCode(450)
	}


	checkCodeRs := false
	ruleId := 0
	for _,rule := range gamematch.RuleConfig.getAll(){
		if  rule.CategoryKey == matchCode{
			ruleId = rule.Id
			checkCodeRs = true
			break
		}
	}
	if !checkCodeRs{
		return data,myerr.NewErrorCode(451)
	}

	groupIdStr,ok := jsonDataMap["groupId"]
	if !ok {
		return data,myerr.NewErrorCode(452)
	}
	//fmt.Printf("type:%T, value:%+v\n", groupIdStr,groupIdStr)
	//groupId := zlib.Atoi(groupIdStr.(string))
	groupId := int(groupIdStr.(float64))
	if groupId == 0{
		return data,myerr.NewErrorCode(452)
	}

	customProp := ""
	customPropStr,ok := jsonDataMap["customProp"]
	if !ok {
		//return data,err.NewErrorCode(453)
		customProp = ""
	}else{
		customProp = customPropStr.(string)
	}

	playerListMap,ok := jsonDataMap["playerList"]
	if !ok {
		return data,myerr.NewErrorCode(454)
	}

//[     map[matchAttr:map[age:10 sex:girl] uid:4] map[matchAttr:map[age:15 sex:man] uid:5]]
	//fmt.Printf("unexpected type %T", playerListMap)
	//fmt.Printf("playerListMap : %+v",playerListMap)
	playerList ,ok := playerListMap.([]interface{})
	if !ok{
		return data,myerr.NewErrorCode(455)
	}

	var playerListStruct []Player
	for _,v := range playerList{
		row := v.(map[string]interface{})
		zlib.MyPrint(row)
		id ,ok := row["uid"]
		if !ok {
			return data,myerr.NewErrorCode(456)
		}

		pid := int(id.(float64))
		playerListStruct = append(playerListStruct,Player{Id:pid})
	}

	addition := ""
	additionStr,ok := jsonDataMap["addition"]
	if ok {
		addition = additionStr.(string)
	}
	data.GroupId = groupId
	data.PlayersList = playerListStruct
	data.CustomProp = customProp
	data.RuleId = ruleId
	data.Addition = addition
	return data,nil
}

func (gamematch *Gamematch) DemonAll(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	mylog.Info("start DemonAll ,  total : ",queueLen)
	if queueLen <=0 {
		mylog.Error(" rule is zero , no need")
		return
	}

	for _,rule := range queueList{
		mylog.Info(" rule : ",rule.Id, " code : ",rule.CategoryKey)
		//if rule.Id != 999{//测试使用
		//	continue
		//}
		checkRs,_ := gamematch.RuleConfig.CheckRuleByElement(rule)
		mylog.Info(" check rule rs :",checkRs)
		if !checkRs{
			mylog.Error("check rule element ,ruleId :",rule.Id , " continue ")
			continue
		}

		push := gamematch.getContainerPushByRuleId(rule.Id)
		go push.checkStatus()

		//定时，检查，所有报名的组是否发生变化，将发生变化的组，删除掉
		queueSign := gamematch.getContainerSignByRuleId(rule.Id)
		go queueSign.CheckTimeout(push)
		//zlib.ExitPrint(12312333333)
		queueSuccess := gamematch.getContainerSuccessByRuleId(rule.Id)
		go queueSuccess.CheckTimeout(push)

		match := gamematch.containerMatch[rule.Id]
		go match.start()
	}
}

func (gamematch *Gamematch) DelAll(){
	mylog.Notice(" action :  DelAll")
	keys := redisPrefix + "*"
	myredis.RedisDelAllByPrefix(keys)
}

func (gamematch *Gamematch) startHttpd(httpdOption HttpdOption){
	httpd := NewHttpd(httpdOption,gamematch)
	httpd.Start()
}



func   mySleepSecond(second time.Duration , msg string){
	mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}

func deadLoopBlock(sleepSecond time.Duration,msg string){
	for {
		mySleepSecond(sleepSecond,  " deadLoopBlock: " +msg)
	}
}

func (gamematch *Gamematch)  delOneRule(  ruleId int){
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	queueSign.delOneRule()


	push :=   gamematch.getContainerPushByRuleId(ruleId)
	push.delOneRule()

	queueSuccess := gamematch.getContainerSuccessByRuleId(ruleId)
	queueSuccess.delOneRule()

	playerStatus.delAllPlayers()
	mylog.Notice("testSignDel finish.")
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

func getModuleLogInc( moduleName string)*zlib.Log{
	logOption := zlib.LogOption{
		OutFilePath : LOG_BASE_DIR ,
		OutFileName: moduleName + ".log",
		Level : LOG_LEVEL,
		Target : LOG_TARGET,
	}
	newLog,err := zlib.NewLog(logOption)
	if err != nil{
		zlib.ExitPrint(err.Error())
	}
	return newLog
}


func getRuleModuleLogInc(ruleCategory string,moduleName string)*zlib.Log{
	dir := LOG_BASE_DIR + "/" + ruleCategory
	exist ,err := zlib.PathExists(dir)
	if err != nil || exist{//证明目录存在
		mylog.Debug("dir has exist",dir)
	}else{
		err := os.Mkdir(dir, 0777)
		if err != nil{
			zlib.ExitPrint("create dir failed ",err.Error())
		}else{
			mylog.Debug("create dir success : ",dir)
		}
	}

	logOption := zlib.LogOption{
		OutFilePath : dir ,
		OutFileName: moduleName + ".log",
		Level : LOG_LEVEL,
		Target : LOG_TARGET,
	}
	newLog,err := zlib.NewLog(logOption)
	if err != nil{
		zlib.ExitPrint(err.Error())
	}
	return newLog
}