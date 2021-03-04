package gamematch

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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
}
//报名成功后，返回的结果
type SignSuccessReturnData struct {
	RuleId 		int
	GroupId		int
	PlayerIds 	string
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

func NewGamematch(zlog *zlib.Log  ,zredis *zlib.MyRedis,zservice *zlib.Service ,zetcd *zlib.MyEtcd,pidFilePath string )(gamematch *Gamematch ,errs error){
	mylog = zlog
	myredis = zredis
	myservice = zservice
	myetcd = zetcd
	//初始化-错误码
	container := getErrorCode()
	mylog.Info("getErrorCode len : ",len(container))
	if   len(container) == 0{
		return gamematch,errors.New("getErrorCode len  = 0")
	}
	//初始化-错误/异常 类
	myerr = zlib.NewErr(mylog,container)
	gamematch = new (Gamematch)
	if pidFilePath == ""{
		return gamematch,myerr.NewErrorCode(501)
	}
	gamematch.PidFilePath = pidFilePath
	err := gamematch.initPid()
	if err != nil{
		return gamematch,err
	}

	gamematch.RuleConfig, errs = NewRuleConfig(gamematch)
	if errs != nil{
		return gamematch,errs
		//zlib.ExitPrint(" NewRuleConfig conn err.",errs.Error())
	}

	gamematch.containerPush		= make(map[int]*Push)
	gamematch.containerSign 	= make(	map[int]*QueueSign )	//容器 报名类
	gamematch.containerSuccess = make(	map[int]*QueueSuccess)
	gamematch.containerMatch	= make( 	map[int]*Match)
	//信号管道
	gamematch.signalChan =  make( map[int]map[string] chan int )
	mylog.Info("init container....")
	//实例化容器
	for _,rule := range gamematch.RuleConfig.getAll(){
		gamematch.containerPush[rule.Id] = NewPush(rule,gamematch)
		gamematch.containerSign[rule.Id] = NewQueueSign(rule,gamematch)
		gamematch.containerSuccess[rule.Id] = NewQueueSuccess(rule,gamematch)
		gamematch.containerMatch[rule.Id] = NewMatch(rule,gamematch)
	}
	playerStatus = NewPlayerStatus()

	gamematch.HttpdRuleState = make(map[int]int)
	for ruleId ,_ := range gamematch.RuleConfig.getAll(){
		gamematch.HttpdRuleState[ruleId] = HTTPD_RULE_STATE_INIT
	}

	go gamematch.DemonSignal()
	return gamematch,nil
}
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

	return nil
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
func (gamematch *Gamematch)getContainerMatchByRuleId(ruleId int)*Match{
	return gamematch.containerMatch[ruleId]
}

//报名 - 加入匹配队列
//此方法，得有前置条件：验证所有参数是否正确，因为使用者为http请求，数据的验证交由HTTP层做处理，如果是非HTTP，要验证一下
//func (gamematch *Gamematch) Sign(ruleId int ,outGroupId int, customProp string, players []Player ,addition string )( signSuccessReturnData SignSuccessReturnData , err error){
func (gamematch *Gamematch) Sign(httpReqSign HttpReqSign )( signSuccessReturnData SignSuccessReturnData , err error){
	ruleId := httpReqSign.RuleId
	outGroupId :=  httpReqSign.GroupId
	rule,ok := gamematch.RuleConfig.GetById(ruleId)
	if !ok{
		return signSuccessReturnData,myerr.NewErrorCode(400)
	}
	lenPlayers := len(httpReqSign.PlayerList)
	if lenPlayers == 0{
		return signSuccessReturnData,myerr.NewErrorCode(401)
	}
	queueSign := gamematch.getContainerSignByRuleId(ruleId)
	//groupsTotal := queueSign.getAllGroupsWeightCnt()	//报名 小组总数
	//playersTotal := queueSign.getAllPlayersCnt()	//报名 玩家总数
	//mylog.Info(" action :  Sign , players : " + strconv.Itoa(lenPlayers) +" ,queue cnt : groupsTotal",groupsTotal ," , playersTotal",playersTotal)
	mylog.Info("new sign :[ruleId : ",ruleId,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	queueSign.Log.Info("new sign :[ruleId : " ,  ruleId   ,"(",rule.CategoryKey,") , outGroupId : ",outGroupId," , playersCount : ",lenPlayers,"] ")
	now := zlib.GetNowTimeSecondToInt()

	queueSign.Log.Info("start check player status :")
	//检查，所有玩家的状态
	//playerStatusElementMap := make( map[int]PlayerStatusElement )
	var  players []Player
	for _,httpPlayer := range httpReqSign.PlayerList{
		player := Player{Id: httpPlayer.Uid}
		//playerStatusElement := playerStatus.GetOne(player)
		playerStatusElement,isEmpty := playerStatus.GetById(player.Id)
		//zlib.ExitPrint(playerStatusElement,isEmpty)
		//fmt.Printf("playerStatus : %+v",playerStatusElement)
		//zlib.MyPrint("k : ",k," , playerStatusElement GetOne: ",playerStatusElement)
		queueSign.Log.Info("player ( ",player.Id , " ) statusElement :  status = ", playerStatusElement.Status)
		//if playerStatusElement.Status == PlayerStatusNotExist{
		if isEmpty == 1{
			//这是正常
		}else if playerStatusElement.Status == PlayerStatusSuccess{
			queueSign.Log.Error(" player status = PlayerStatusSuccess ,demon not clean.")
			msg := make(map[int]string)
			msg[0] = strconv.Itoa(player.Id)
			return signSuccessReturnData,myerr.NewErrorCodeReplace(403,msg)
		}else if playerStatusElement.Status == PlayerStatusSign{
			isTimeout := playerStatus.checkSignTimeout(rule,playerStatusElement)
			if !isTimeout{//未超时
				//queueSign.Log.Error(" player status = matching...  not timeout")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return signSuccessReturnData,myerr.NewErrorCodeReplace(402,msg)
			}else{//报名已超时，等待后台守护协程处理
				//这里其实也可以先一步处理，但是怕与后台协程冲突
				//queueSign.Log.Error(" player status is timeout ,but not clear , wait a moment!!!")
				msg := make(map[int]string)
				msg[0] = strconv.Itoa(player.Id)
				return signSuccessReturnData,myerr.NewErrorCodeReplace(407,msg)
			}
		}

		players  = append(players,player)
		//playerStatusElementMap[player.Id] = playerStatusElement
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
	group.Addition = httpReqSign.Addition
	group.CustomProp =  httpReqSign.CustomProp
	group.MatchCode = rule.CategoryKey
	mylog.Info("newGroupId : ",group.Id , "player/group weight : " ,playerWeight ," now : ",now ," expire : ",expire )
	queueSign.Log.Info("newGroupId : ",group.Id , "player/group weight : " ,playerWeight ," now : ",now ," expire : ",expire)
	//下面两行必须是原子操作，如果pushOne执行成功，但是upInfo没成功会导致报名队列里，同一个用户能再报名一次
	//queueSign.Mutex.Lock()
	queueSign.AddOne(group)
	//defer queueSign.Mutex.Unlock()

	playerIds := ""
	for _,player := range players{
		newPlayerStatusElement := playerStatus.newPlayerStatusElement()
		newPlayerStatusElement.PlayerId = player.Id
		newPlayerStatusElement.Status = PlayerStatusSign
		newPlayerStatusElement.RuleId = ruleId
		newPlayerStatusElement.Weight = playerWeight
		newPlayerStatusElement.SignTimeout = expire
		newPlayerStatusElement.GroupId = group.Id

		playerStatus.upInfo(  newPlayerStatusElement)

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
	push := gamematch.getContainerPushByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"push",push.Log,push.checkStatus)
	//报名超时
	queueSign := gamematch.getContainerSignByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"signTimeout",queueSign.Log,queueSign.CheckTimeout)
	//报名成功，但3方迟迟推送失败，无人接收，超时
	queueSuccess := gamematch.getContainerSuccessByRuleId(rule.Id)
	go gamematch.MyDemon(rule.Id,"successTimeout",queueSuccess.Log,queueSuccess.CheckTimeout)
	//匹配
	match := gamematch.containerMatch[rule.Id]
	go gamematch.MyDemon(rule.Id,"matching",match.Log,match.start)

	mylog.Info("start httpd ,up state...")
	gamematch.HttpdRuleState[rule.Id] = HTTPD_RULE_STATE_OK
}

func (gamematch *Gamematch) startHttpd(httpdOption HttpdOption)error{
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
		OutFilePath : LOG_BASE_DIR ,
		OutFileName: moduleName + ".log",
		Level : LOG_LEVEL,
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
	dir := LOG_BASE_DIR + "/" + ruleCategory
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
		Level : LOG_LEVEL,
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
