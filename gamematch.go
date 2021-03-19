package gamematch

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"zlib"
)

type Gamematch struct {
	RuleConfig 			*RuleConfig
	PidFilePath			string
	containerSign 		map[int]*QueueSign	//容器 报名类
	containerSuccess 	map[int]*QueueSuccess
	containerMatch	 	map[int]*Match
	containerPush		map[int]*Push
	//signalChan			map[int] chan int
	signalChan			map[int] map[string] chan int
	HttpdRuleState		map[int]int
	PlayerStatus 	*PlayerStatus
}
////报名成功后，返回的结果
//type SignSuccessReturnData struct {
//	RuleId 		int
//	GroupId		int
//	PlayerIds 	string
//}
//玩家结构体，目前暂定就一个ID，其它的字段值，后期得加，主要是用于权重计算
type Player struct {
	Id			int
	MatchAttr	map[string]int
	Weight		float32
}

var mylog 		*zlib.Log
var myerr 		*zlib.Err
var myredis 	*zlib.MyRedis
var myservice	*zlib.Service
var myetcd		*zlib.MyEtcd
var mymetrics 	*zlib.Metrics
var playerStatus 	*PlayerStatus	//玩家状态类

type GamematchOption struct {
	Log 			*zlib.Log
	Redis 			*zlib.MyRedis
	Service 		*zlib.Service
	Etcd 			*zlib.MyEtcd
	Metrics 		*zlib.Metrics
	PidFilePath 	string
	MonitorRuleIds 	[]int
}

func NewGamematch(gamematchOption GamematchOption)(gamematch *Gamematch ,errs error){
	mylog = gamematchOption.Log
	myredis = gamematchOption.Redis
	myservice = gamematchOption.Service
	myetcd = gamematchOption.Etcd
	mymetrics = gamematchOption.Metrics

	mymetrics.CreateOneNode("matchingSuccess")
	mymetrics.CreateOneNode("pushFailed")
	mymetrics.CreateOneNode("pushDrop")
	mymetrics.CreateOneNode("pushSuccess")

	mylog.Info("NewGamematch : ")

	//初始化-错误码
	container := getErrorCode()
	mylog.Info( " init ErrorCodeList , len : ",len(container))
	if   len(container) == 0{
		return gamematch,errors.New("getErrorCode len  = 0")
	}
	//初始化-错误/异常 类
	myerr = zlib.NewErr(mylog,container)
	gamematch = new (Gamematch)
	if gamematchOption.PidFilePath == ""{
		return gamematch,myerr.NewErrorCode(501)
	}
	gamematch.PidFilePath = gamematchOption.PidFilePath
	err := gamematch.initPid()
	if err != nil{
		return gamematch,err
	}
	//初始化 - 信号和管道
	gamematch.signalChan =  make( map[int]map[string] chan int )
	//mylog.Info("init container....")
	zlib.AddRoutineList("DemonSignal")
	go gamematch.DemonSignal()

	gamematch.RuleConfig, errs = NewRuleConfig(gamematch,gamematchOption.MonitorRuleIds)
	if errs != nil{
		return gamematch,errs
		//zlib.ExitPrint(" NewRuleConfig conn err.",errs.Error())
	}
	//初始化- 容器 报名、匹配、推送、报名成功
	gamematch.containerPush		= make(map[int]*Push)
	gamematch.containerSign 	= make(	map[int]*QueueSign )
	gamematch.containerSuccess = make(	map[int]*QueueSuccess)
	gamematch.containerMatch	= make( 	map[int]*Match)
	//实例化容器
	for _,rule := range gamematch.RuleConfig.getAll(){
		fmt.Printf("%+v",rule)
		zlib.MyPrint("------------")
		gamematch.containerPush[rule.Id] = NewPush(rule,gamematch)
		gamematch.containerSign[rule.Id] = NewQueueSign(rule,gamematch)
		gamematch.containerSuccess[rule.Id] = NewQueueSuccess(rule,gamematch)
		gamematch.containerMatch[rule.Id] = NewMatch(rule,gamematch)
	}

	containerTotal := len(gamematch.containerPush) + len(gamematch.containerSign) + len(gamematch.containerSuccess) + len(gamematch.containerMatch)
	mylog.Info("container total :",containerTotal)
	//zlib.ExitPrint(1111)
	playerStatus = NewPlayerStatus()
	gamematch.PlayerStatus = playerStatus
	//初始化 - 每个rule - httpd 状态
	gamematch.HttpdRuleState = make(map[int]int)
	mylog.Info("HTTPD_RULE_STATE_INIT")
	for ruleId ,_ := range gamematch.RuleConfig.getAll(){
		gamematch.HttpdRuleState[ruleId] = HTTPD_RULE_STATE_INIT
	}

	return gamematch,nil
}
//进程PID保存到文件
func (gamematch *Gamematch)initPid()error{
	pid := os.Getpid()
	fd, err  := os.OpenFile(gamematch.PidFilePath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC , 0777)
	defer fd.Close()
	if err != nil{
		rrr := myerr.MakeOneStringReplace(gamematch.PidFilePath + " " + err.Error())
		return myerr.NewErrorCodeReplace(502,rrr)
		//zlib.ExitPrint(" init log lib err ,  ",err.Error())
	}

	_, err = io.WriteString(fd, strconv.Itoa(pid))
	if err != nil{
		rrr := myerr.MakeOneStringReplace(gamematch.PidFilePath + " " + err.Error())
		return myerr.NewErrorCodeReplace(503,rrr)
	}

	mylog.Info("init pid :", pid, " , path : ",gamematch.PidFilePath)
	return nil
}
func (gamematch *Gamematch)getContainerSignByRuleId(ruleId int)*QueueSign{
	content ,ok := gamematch.containerSign[ruleId]
	if !ok{
		mylog.Error("getContainerSignByRuleId is null")
	}
	return content
}
func (gamematch *Gamematch)getContainerSuccessByRuleId(ruleId int)*QueueSuccess{
	content ,ok := gamematch.containerSuccess[ruleId]
	if !ok{
		mylog.Error("getContainerSuccessByRuleId is null")
	}
	return content
}
func (gamematch *Gamematch)getContainerPushByRuleId(ruleId int)*Push{
	content ,ok := gamematch.containerPush[ruleId]
	if !ok{
		mylog.Error("getContainerPushByRuleId is null")
	}
	return content
}
func (gamematch *Gamematch)getContainerMatchByRuleId(ruleId int)*Match{
	content ,ok := gamematch.containerMatch[ruleId]
	if !ok{
		mylog.Error("getContainerMatchByRuleId is null")
	}
	return content
}
//请便是方便记日志，每次要写两个FD的日志，太麻烦
func rootAndSingToLogInfoMsg(sign *QueueSign,a ...interface{}){
	mylog.Info(a)
	sign.Log.Info(a)
}
//报名 - 加入匹配队列
//此方法，得有前置条件：验证所有参数是否正确，因为使用者为http请求，数据的验证交由HTTP层做处理，如果是非HTTP，要验证一下
func (gamematch *Gamematch) Sign(httpReqSign HttpReqSign )( group Group , err error){
	ruleId := httpReqSign.RuleId
	outGroupId :=  httpReqSign.GroupId
	//这里只做最基础的验证，前置条件是已经在HTTP层作了验证
	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		return group,myerr.NewErrorCode(400)
	}
	lenPlayers := len(httpReqSign.PlayerList)
	if lenPlayers == 0{
		return group,myerr.NewErrorCode(401)
	}
	//groupsTotal := queueSign.getAllGroupsWeightCnt()	//报名 小组总数
	//playersTotal := queueSign.getAllPlayersCnt()	//报名 玩家总数
	//mylog.Info(" action :  Sign , players : " + strconv.Itoa(lenPlayers) +" ,queue cnt : groupsTotal",groupsTotal ," , playersTotal",playersTotal)
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	rootAndSingToLogInfoMsg(queueSign,"new sign :[ruleId : ",ruleId,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	//mylog.Info("new sign :[ruleId : ",ruleId,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	//queueSign.Log.Info("new sign :[ruleId : " ,  ruleId   ,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	now := zlib.GetNowTimeSecondToInt()

	if lenPlayers > queueSign.Rule.GroupPersonMax{
		msg := make(map[int]string)
		msg[0] = strconv.Itoa( queueSign.Rule.GroupPersonMax)
		return group,myerr.NewErrorCodeReplace(408,msg)
	}
	queueSign.Log.Info("start check player status :")
	//检查，所有玩家的状态
	var  players []Player
	for _,httpPlayer := range httpReqSign.PlayerList{
		player := Player{Id: httpPlayer.Uid,MatchAttr:httpPlayer.MatchAttr}
		playerStatusElement,isEmpty := playerStatus.GetById(player.Id)
		queueSign.Log.Info("player(",player.Id , ") GetById :  status = ", playerStatusElement.Status, " isEmpty:",isEmpty)
		if isEmpty == 1{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{//玩家已经匹配成功，并等待开始游戏
			queueSign.Log.Error(" player status = PlayerStatusSuccess ,demon not clean.")
			msg := make(map[int]string)
			msg[0] = strconv.Itoa(player.Id)
			return group,myerr.NewErrorCodeReplace(403,msg)
		}else if playerStatusElement.Status == PlayerStatusSign{//报名成功，等待匹配
			isTimeout := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{//未超时
				//queueSign.Log.Error(" player status = matching...  not timeout")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return group,myerr.NewErrorCodeReplace(402,msg)
			}else{//报名已超时，等待后台守护协程处理
				//这里其实也可以先一步处理，但是怕与后台协程冲突
				//queueSign.Log.Error(" player status is timeout ,but not clear , wait a moment!!!")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return group,myerr.NewErrorCodeReplace(407,msg)
			}
		}
		players  = append(players,player)
		//playerStatusElementMap[player.Id] = playerStatusElement
	}
	rootAndSingToLogInfoMsg(queueSign,"finish check player status.")
	//先计算一下权重平均值
	var groupWeightTotal float32
	groupWeightTotal = 0.00

	if rule.PlayerWeight.Formula != ""  {
		rootAndSingToLogInfoMsg(queueSign,"rule weight , Formula : ",rule.PlayerWeight.Formula )
		var weight float32
		weight = 0.00
		var playerWeightValue []float32
		for k,p := range players{
			onePlayerWeight := getPlayerWeightByFormula(rule.PlayerWeight.Formula ,p.MatchAttr ,queueSign)
			mylog.Debug("onePlayerWeight : ",onePlayerWeight)
			if onePlayerWeight > WeightMaxValue {
				onePlayerWeight = WeightMaxValue
			}
			weight += onePlayerWeight
			playerWeightValue = append(playerWeightValue,onePlayerWeight)
			players[k].Weight = onePlayerWeight
		}
		switch rule.PlayerWeight.Aggregation{
			case "sum":
				groupWeightTotal = weight
			case "min":
				groupWeightTotal = zlib.FindMinNumInArrFloat32(playerWeightValue)
			case "max":
				groupWeightTotal = zlib.FindMaxNumInArrFloat32(playerWeightValue)
			case "average":
				groupWeightTotal =  weight / float32(len(players))
			default:
				groupWeightTotal =  weight / float32(len(players))
		}
		//保留2位小数
		tmp := zlib.FloatToString(groupWeightTotal,2)
		groupWeightTotal = zlib.StringToFloat(tmp)
	}else{
		rootAndSingToLogInfoMsg(queueSign,"rule weight , Formula is empty!!!")
	}
	//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
	redisConnFD := myredis.GetNewConnFromPool()
	defer redisConnFD.Close()
	//开始多指令缓存模式
	myredis.Multi(redisConnFD)

	//超时时间
	expire := now + rule.MatchTimeout
	//创建一个新的小组
	group =  gamematch.NewGroupStruct(rule)
	//这里有偷个懒，还是用外部的groupId , 不想再给redis加 groupId映射outGroupId了
	mylog.Notice(" outGroupId replace groupId :",outGroupId,group.Id)
	group.Id = outGroupId
	group.Players = players
	group.SignTimeout = expire
	group.Person = len(players)
	group.Weight = groupWeightTotal
	group.OutGroupId = outGroupId
	group.Addition = httpReqSign.Addition
	group.CustomProp =  httpReqSign.CustomProp
	group.MatchCode = rule.CategoryKey
	rootAndSingToLogInfoMsg(queueSign,"newGroupId : ",group.Id , "player/group weight : " ,groupWeightTotal ," now : ",now ," expire : ",expire)
	//mylog.Info("newGroupId : ",group.Id , "player/group weight : " ,groupWeightTotal ," now : ",now ," expire : ",expire )
	//queueSign.Log.Info("newGroupId : ",group.Id , "player/group weight : " ,groupWeightTotal ," now : ",now ," expire : ",expire)
	queueSign.AddOne(group,redisConnFD)
	playerIds := ""
	for _,player := range players{

		newPlayerStatusElement := playerStatus.newPlayerStatusElement()
		newPlayerStatusElement.PlayerId = player.Id
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = player.Weight
		newPlayerStatusElement.SignTimeout = expire
		newPlayerStatusElement.GroupId = group.Id

		queueSign.Log.Info("playerStatus.upInfo:" ,PlayerStatusSign)
		playerStatus.upInfo(  newPlayerStatusElement , redisConnFD)

		playerIds += strconv.Itoa(player.Id) + ","
	}
	//提交缓存中的指令
	_,err = myredis.Exec(redisConnFD )
	if err != nil{
		queueSign.Log.Error("transaction failed : ",err)
	}
	queueSign.Log.Info(" sign finish ,total : newGroupId " , group.Id ," success players : ",len(players))
	mylog.Info(" sign finish ,total : newGroupId " , group.Id ," success players : ",len(players))

	//signSuccessReturnData = SignSuccessReturnData{
	//	RuleId: ruleId,
	//	GroupId: outGroupId,
	//	PlayerIds: playerIds,
	//
	//}

	return group,nil
}

func getPlayerWeightByFormula(formula string,MatchAttr map[string]int,sign *QueueSign)float32{
	//mylog.Debug("getPlayerWeightByFormula , formula:",formula)
	grep := FormulaFirst + "([\\s\\S]*?)"+FormulaEnd
	var imgRE = regexp.MustCompile(grep)
	findRs := imgRE.FindAllStringSubmatch(formula, -1)
	rootAndSingToLogInfoMsg(sign,"parse PlayerWeightByFormula : ",findRs)
	if len(findRs) == 0{
		return 0
	}
	for _,v := range findRs {
		//zlib.MyPrint(v[0],v[1])
		//fmt.Println(strconv.Itoa(i), out[i])
		val ,ok := MatchAttr[v[1]]
		if !ok {
			val = 0
		}
		formula = strings.Replace(formula,v[0],strconv.Itoa(val),-1)

	}
	rootAndSingToLogInfoMsg(sign,"final formula replaced str :",formula)
	rs, err := zlib.Eval(formula)
	if err != nil{
		return 0
	}
	f,_ :=rs.Float32()
	return f
}

func (gamematch *Gamematch)notifyRoutine(sign chan int,signType int){
	mylog.Notice("send routine : ",signType)
	sign <- signType
}
func (gamematch *Gamematch)countSignalChan(exceptRuleId int)int{
	cnt :=0
	for ruleId,set := range gamematch.signalChan{
		if ruleId == exceptRuleId{
			continue
		}
		cnt += len(set)
	}
	return cnt
}
func  (gamematch *Gamematch)closeOneRuleDemonRoutine( ruleId int )int{
	mylog.Info("closeOneRuleDemonRoutine ,ruleId :  ",ruleId , " start" )


	_,ok := gamematch.HttpdRuleState[ruleId]
	if !ok{
		mylog.Error("closeOneRuleDemonRoutine ,gamematch.HttpdRuleState[ruleId]  is null ",ruleId)
		return -1
	}
	mylog.Info("close httpd ,up state...")
	gamematch.HttpdRuleState[ruleId] = HTTPD_RULE_STATE_CLOSE

	_,ok = gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		mylog.Error("closeOneRuleDemonRoutine ,ruleId not in db  ... ",ruleId)
		return -1
	}

	_,ok = gamematch.signalChan[ruleId]
	if !ok{
		mylog.Error("closeOneRuleDemonRoutine ,gamematch.signalChan[ruleId]  is null ",ruleId)
		return -2
	}

	for title,mychann := range gamematch.signalChan[ruleId]{
		//mylog.Warning(SIGNAL_SEND_DESC  , ruleId,title )
		//mychann <- SIGN_GOROUTINE_EXEC_EXIT
		gamematch.signSend(mychann,SIGNAL_GOROUTINE_EXEC_EXIT,title)
	}

	closeRoutineCnt := 0
	for{
		if closeRoutineCnt == len(gamematch.signalChan[ruleId]){
			break
		}
		for title,chanLink := range gamematch.signalChan[ruleId]{
			select {
			case sign := <- chanLink:
				mylog.Warning("SIGNAL send" ,sign,"退出完成 , ruleId: ",ruleId,title)
				closeRoutineCnt++
			default:
				mylog.Info("closeDemonRoutine  waiting for goRoutine callback signal...")
			}
		}
	}

	for title,mychann := range gamematch.signalChan[ruleId]{
		mylog.Warning("close mychann ,",ruleId,title)
		close(mychann)
	}
	delete(gamematch.signalChan,ruleId)
	mylog.Warning( "delete gamematch.signalChan map key :",ruleId)

	mylog.Info("closeOneRuleDemonRoutine ,ruleId :  ",ruleId , " end" )
	return closeRoutineCnt
}
func  (gamematch *Gamematch)closeDemonRoutine(  )int{
	mylog.Info("closeDemonRoutine : start " )

	//要先关闭入口，也就是HTTPD
	httpdChan := gamematch.signalChan[0]["httpd"]
	//select {
	//	case httpdChan <- SIGN_GOROUTINE_EXEC_EXIT:
	//		mylog.Warning(SIGNAL_SEND_DESC ,SIGN_GOROUTINE_EXEC_EXIT,0,"httpd")
	//	case sign := <- httpdChan:
	//		mylog.Warning(SIGNAL_RECE_DESC ,sign,0,"httpd")
	//}
	gamematch.signSend(httpdChan,SIGNAL_GOROUTINE_EXEC_EXIT,"httpd")
	gamematch.signReceive(httpdChan,"httpd")

	mylog.Notice("gamematch.signalChan len:",len(gamematch.signalChan))
	for ruleId,_ := range gamematch.signalChan{
		if ruleId == 0{//0是特殊管理，仅给HTTPD使用
			continue
		}
		gamematch.closeOneRuleDemonRoutine(ruleId)
		//for title,chanLink := range set{
		//	mylog.Warning(SIGNAL_SEND_DESC  , ruleId,title )
		//	chanLink <- SIGN_GOROUTINE_EXEC_EXIT
		//}
	}
	mylog.Info("closeDemonRoutine : all routine quit success~~~")
	return 1

	//chanHasFinished := make(map[int] map[string]  int)
	//chanHasFinishedCnt := 0
	////signalChanLen := len(gamematch.signalChan)
	//countSignalChanNumber := gamematch.countSignalChan(0)
	//for{
	//	if chanHasFinishedCnt == countSignalChanNumber{
	//		break
	//	}
	//
	//	for ruleId,set := range gamematch.signalChan{
	//		if ruleId == 0{//0是特殊通道，留给httpd
	//			continue
	//		}
	//		for title,chanLink := range set{
	//			_,ok :=  chanHasFinished[ruleId][title]
	//			if ok{
	//				continue
	//			}
	//
	//			select {
	//				case sign := <- chanLink:
	//					mylog.Warning(SIGNAL_RECE_DESC ,sign,ruleId,title)
	//					_,ok := chanHasFinished[ruleId]
	//					if !ok {
	//						tmp := make(map[string] int)
	//						chanHasFinished[ruleId] = tmp
	//					}
	//					chanHasFinished[ruleId][title] = 1
	//					chanHasFinishedCnt++
	//				default:
	//					mylog.Info("closeDemonRoutine  waiting for goRoutine callback signal...")
	//			}
	//		}
	//	}
	//	//	_,ok :=  chanHasFinished[i]
	//	//	if ok{
	//	//		continue
	//	//	}
	//	//	select {
	//	//		case sign := <- gamematch.signalChan[i]:
	//	//			mylog.Warning(SIGNAL_RECE_DESC + strconv.Itoa(sign))
	//	//			chanHasFinished[i] = sign
	//	//		default:
	//	//			mylog.Info("closeDemonRoutine  waiting for goRoutine callback signal...")
	//	//	}
	//	//}
	//	mySleepSecond(1,"closeDemonRoutine")
	//}

}
//
func(gamematch *Gamematch) quit(source int){
	state := gamematch.closeDemonRoutine()
	os.Exit(state)
}
//接收一个管道的数据-会阻塞
func (gamematch *Gamematch)signReceive(mychan chan int,keyword string)int{
	mylog.Warning("SIGNAL receive blocking.... ")
	sign := <- mychan
	mylog.Warning("SIGNAL receive: ",sign,getSignalDesc(sign),keyword)
	return sign
}
func (gamematch *Gamematch)signSend(mychan chan int,data int,keyword string){
	mylog.Warning("SIGNAL send blocking.... ")
	mychan <- data
	mylog.Warning("SIGNAL send: ",data,getSignalDesc(data),keyword)
}
//
func (gamematch *Gamematch)getNewSignalChan(ruleId int,title string)chan int{
	mylog.Debug("getNewSignalChan: ",ruleId,title)
	signalChann := make(chan int)
	_,ok := gamematch.signalChan[ruleId]
	if !ok {
		tmp := make(map[string]chan int)
		gamematch.signalChan[ruleId] = tmp
	}
	gamematch.signalChan[ruleId][title] = signalChann
	return gamematch.signalChan[ruleId][title]
	//index := len(gamematch.signalChan)
	//gamematch.signalChan[index] = signalChann
	//mylog.Notice(SIGNAL_NEW_CHAN_DESC , " index: ",index)
	//return gamematch.signalChan[index]
}
//信号 处理
func (gamematch *Gamematch)DemonSignal(){
	mylog.Warning("SIGNAL init : ")
	c := make(chan os.Signal)
	signal.Notify(c,syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	prefix := "SIGNAL-DEMON :"
	for{
		sign := <- c
		mylog.Warning(prefix,sign)
		switch sign {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			mylog.Warning(prefix+" exit!!!")
			gamematch.quit(SIGNAL_QUIT_SOURCE)
		case syscall.SIGUSR1:
			mylog.Warning(prefix+" usr1!!!")
		case syscall.SIGUSR2:
			mylog.Warning(prefix+" usr2!!!")
		default:
			mylog.Warning(prefix+" unknow!!!")
		}
		mySleepSecond(1,prefix)
	}
}

func (gamematch *Gamematch)MyDemon( ruleId int ,title string,demonLog *zlib.Log, handler func( )){
	mylog.Warning("MyDemon : new one   , ruleId: ",ruleId, "title: " ,title)
	demonLog.Warning("MyDemon : new one   , ruleId: ",ruleId, "title: " ,title)
	signalChan := gamematch.getNewSignalChan(ruleId,title)
	for{
		select {
		case signal := <-signalChan:
			//demonLog.Warning(SIGNAL_RECE_DESC ,signal , " in MyDeomon (" +title+ ") goRuntine")
			//demonLog.Warning(SIGNAL_SEND_DESC , signal , " from ",title)

			mylog.Warning("SIGNAL receive: ",signal,getSignalDesc(signal),title)
			mylog.Warning("SIGNAL send: ",SIGNAL_GOROUTINE_EXIT_FINISH,getSignalDesc(SIGNAL_GOROUTINE_EXIT_FINISH),title)
			signalChan <- SIGNAL_GOROUTINE_EXIT_FINISH
			goto forEnd
		default:
			handler()
			mySleepSecond(1,"MyDemon : "+title)
		}
	}
	forEnd:
		demonLog.Notice("MyDemon end : ",title)
		mylog.Notice("MyDemon end : ",title)
}
//启动后台守护-协程
func (gamematch *Gamematch) DemonAll(){
	queueList := gamematch.RuleConfig.getAll()
	queueLen := len(queueList)
	mylog.Info("start DemonAll ,  rule total : ",queueLen)
	if queueLen <=0 {
		mylog.Error(" rule is zero , no need")
		return
	}
	//inc := 0
	for _,rule := range queueList{
		if rule.Id == 0{//0是特殊管理，仅给HTTPD使用
			continue
		}
		gamematch.startOneRuleDomon(rule)
		//inc++
	}
	//mylog.Info(" DemonAll : open goRoutine total : " ,inc)

	//for _,rule := range queueList{
	//	mylog.Info(" rule : ",rule.Id, " code : ",rule.CategoryKey)
	//	//if rule.Id != 999{//测试使用
	//	//	continue
	//	//}
	//	checkRs,_ := gamematch.RuleConfig.CheckRuleByElement(rule)
	//	mylog.Info(" check rule rs :",checkRs)
	//	if !checkRs{
	//		mylog.Error("check rule element ,ruleId :",rule.Id , " continue ")
	//		continue
	//	}
	//
	//	push := gamematch.getContainerPushByRuleId(rule.Id)
	//	go push.checkStatus()
	//
	//	//定时，检查，所有报名的组是否发生变化，将发生变化的组，删除掉
	//	queueSign := gamematch.getContainerSignByRuleId(rule.Id)
	//	go queueSign.CheckTimeout(push)
	//	//zlib.ExitPrint(12312333333)
	//	queueSuccess := gamematch.getContainerSuccessByRuleId(rule.Id)
	//	go queueSuccess.CheckTimeout(push)
	//
	//	match := gamematch.containerMatch[rule.Id]
	//	go match.start()
	//}
}

func  (gamematch *Gamematch)startOneRuleDomon(rule Rule){
	zlib.AddRoutineList("startOneRuleDomon push")
	push := gamematch.getContainerPushByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"push",push.Log,push.checkStatus)
	//报名超时
	zlib.AddRoutineList("startOneRuleDomon signTimeout")
	queueSign := gamematch.getContainerSignByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"signTimeout",queueSign.Log,queueSign.CheckTimeout)
	//报名成功，但3方迟迟推送失败，无人接收，超时
	zlib.AddRoutineList("startOneRuleDomon successTimeout")
	queueSuccess := gamematch.getContainerSuccessByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"successTimeout",queueSuccess.Log,queueSuccess.CheckTimeout)
	//匹配
	zlib.AddRoutineList("startOneRuleDomon matching")
	match := gamematch.containerMatch[rule.Id]
	go gamematch.MyDemon(rule.Id,"matching",match.Log,match.matching)

	mylog.Info("start httpd ,up state...")
	gamematch.HttpdRuleState[rule.Id] = HTTPD_RULE_STATE_OK
}

func (gamematch *Gamematch) StartHttpd(httpdOption HttpdOption)error{
	httpd,err  := NewHttpd(httpdOption,gamematch)
	if err != nil{
		return err
	}
	httpd.Start()
	return nil
}
//睡眠 - 协程
func   mySleepSecond(second time.Duration , msg string){
	mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}
//让出当前协程执行时间
func myGosched(msg string){
	mylog.Info(msg + " Gosched ..." )
	runtime.Gosched()
}
//死循环
func deadLoopBlock(sleepSecond time.Duration,msg string){
	for {
		mySleepSecond(sleepSecond,  " deadLoopBlock: " +msg)
	}
}
//实例化，一个LOG类，基于模块
func getModuleLogInc( moduleName string)(newLog *zlib.Log ,err error){
	logOption := zlib.LogOption{
		OutFilePath : myetcd.GetAppConfByKey("log_base_path") ,
		OutFileName: moduleName + ".log",
		Level : zlib.Atoi(myetcd.GetAppConfByKey("log_level")),
		Target : 6,
	}
	newLog,err = zlib.NewLog(logOption)
	if err != nil{
		return newLog,err
	}
	return newLog,nil
}
//实例化，一个LOG类，基于RULE+模块
func getRuleModuleLogInc(ruleCategory string,moduleName string)*zlib.Log{
	dir := myetcd.GetAppConfByKey("log_base_path") + "/" + ruleCategory
	exist ,err := zlib.PathExists(dir)
	if err != nil || exist{//证明目录存在
		//mylog.Debug("dir has exist",dir)
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
		Level : zlib.Atoi(myetcd.GetAppConfByKey("log_level")),
		Target : 6,
	}
	newLog,err := zlib.NewLog(logOption)
	if err != nil{
		zlib.ExitPrint(err.Error())
	}
	return newLog
}
func (gamematch *Gamematch) DelOneRuleById(ruleId int){
	gamematch.RuleConfig.delOne(ruleId)
}

//删除全部数据
func (gamematch *Gamematch) DelAll(){
	mylog.Notice(" action :  DelAll")
	keys := redisPrefix + "*"
	myredis.RedisDelAllByPrefix(keys)
}

func (gamematch *Gamematch)RedisMetrics()(rulelist map[int]Rule ,list map[int]map[string]int,playerCnt map[string]int,rulePersonNum map[int]map[int]int){
	rulelist = gamematch.RuleConfig.getAll()

	playerList,_ := playerStatus.getAllPlayers()
	playerCnt = make(map[string]int)
	playerCnt["total"] = 0
	playerCnt["signTimeout"] = 0
	playerCnt["successTimeout"] = 0
	//下面是状态
	playerCnt["sign"] = 0
	playerCnt["success"] = 0
	playerCnt["int"] = 0
	playerCnt["unknow"] = 0

	if(len(playerList) > 0){
		now := zlib.GetNowTimeSecondToInt()
		for _,playerStatusElement := range playerList{
			if playerStatusElement.Status == PlayerStatusSign{
				playerCnt["sign"]++
			}else if playerStatusElement.Status == PlayerStatusSuccess{
				playerCnt["success"]++
			}else if playerStatusElement.Status == PlayerStatusInit{
				playerCnt["int"]++
			}else{
				playerCnt["unknow"]++
			}

			if now - playerStatusElement.SignTimeout >  rulelist[playerStatusElement.RuleId].MatchTimeout{
				playerCnt["signTimeout"]++
			}

			if playerStatusElement.SuccessTimeout > 0 {
				if now - playerStatusElement.SuccessTimeout >  rulelist[playerStatusElement.RuleId].SuccessTimeout{
					playerCnt["successTimeout"]++
				}
			}


			playerCnt["total"]++
		}
	}
	list = make(map[int]map[string]int)

	rulePersonNum = make(map[int]map[int]int)
	for ruleId,_ := range rulelist{
		//prefix := "rule("+strconv.Itoa(ruleId) + ")"

		row := make(map[string]int)
		row["player"] = playerStatus.getOneRuleAllPlayerCnt(ruleId)
		//playerCnt := playerStatus.getOneRuleAllPlayerCnt(ruleId)
		//zlib.ExitPrint(playerCnt)
		push := gamematch.getContainerPushByRuleId(ruleId)
		row["push"] = push.getAllCnt()
		//zlib.ExitPrint(pushCnt,prefix,rule)
		row["pushWaitStatut"] = push.getStatusCnt(PushStatusWait)
		row["pushRetryStatut"] = push.getStatusCnt(PushStatusRetry)

		sign := gamematch.getContainerSignByRuleId(ruleId)
		row["signGroup"] = sign.getAllGroupsWeightCnt()
		//row["groupPersonCnt"] =
		//map[int]int
		playersByPerson := sign.getPlayersCntByWeight("0","100")
		rulePersonNum[ruleId] = playersByPerson

		success := gamematch.getContainerSuccessByRuleId(ruleId)
		row["success"] = success.GetAllTimeoutCnt()
	//
	//	//match := httpd.Gamematch.getContainerMatchByRuleId(ruleId)
		list[ruleId] = row
	}
	return rulelist,list,playerCnt,rulePersonNum
}
