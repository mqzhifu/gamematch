package gamematch

import (
	"zlib"
	"strconv"
	"sync"
)

type Match struct {
	Mutex 	sync.Mutex
	Rule 	Rule
	QueueSign *QueueSign
	QueueSuccess *QueueSuccess
	Push 	*Push
	Gamematch	*Gamematch
	rangeStart	string
	rangeEnd	string
	Log *zlib.Log
}

func NewMatch(rule Rule , gamematch *Gamematch)*Match{
	match := new(Match)
	match.Rule = rule
	match.Gamematch = gamematch
	match.rangeStart = ""
	match.rangeEnd = ""

	match.QueueSign = match.Gamematch.getContainerSignByRuleId(rule.Id)
	match.QueueSuccess = match.Gamematch.getContainerSuccessByRuleId(rule.Id)
	match.Push = match.Gamematch.getContainerPushByRuleId(rule.Id)

	match.Log = getRuleModuleLogInc(rule.CategoryKey,"matching")
	return match
}
//一条rule 的匹配
func (match *Match) start(){
	mylog.Info("start one rule matching , ruleId :  ",match.Rule.Id, " category :" ,match.Rule.CategoryKey)
	//for{
	 	match.matching()
	 	//match.Log.Info("sleep 1")
		//mySleepSecond(1, "  matching start ")
	//}
}
//一次完整的匹配大流程
func  (match *Match)  matching() {
	playersTotal := match.QueueSign.getAllPlayersCnt()
	groupsTotal := match.QueueSign.getAllGroupsWeightCnt()
	mylog.Info("new once matching func , playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	match.Log.Info("new once matching func , playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	match.clearMemberRange( )

	if playersTotal == 0 || groupsTotal == 0{
		//mylog.Debug(" first total is empty ")
		match.Log.Info(" first total is empty ")
		return
	}
	//gamematch.getLock()
	//defer gamematch.unLock()

	//先做最大范围搜索：也就是整体看下集合全部元素
	matchingRange := []int{FilterFlagAll,FilterFlagBlock,FilterFlagBlockInc}
	for i:=0;i<len(matchingRange);i++{
		groupIds,isEmpty := match.matchingRange(matchingRange[i])
		if isEmpty == 1{
			mylog.Notice("signed players is empty ")
			return
		}
		if matchingRange[i] == FilterFlagAll{
			if len(groupIds) >0 {
				mylog.Info("FilterFlagAll hit...")
				return
			}
		}
	}

	finalPlayersTotal := match.QueueSign.getAllPlayersCnt()
	finalGroupsTotal := match.QueueSign.getAllGroupsWeightCnt()

	mylog.Info("once matching end, playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal,
		"finalPlayersTotal total:",finalPlayersTotal , " finalGroupsTotal : ",finalGroupsTotal)
	match.Log.Info("once matching end, playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal,
		"finalPlayersTotal total:",finalPlayersTotal , " finalGroupsTotal : ",finalGroupsTotal)

	match.clearMemberRange( )
}

func (match *Match) setMemberRange(rangeStart string,rangeEnd string){
	match.rangeStart = rangeStart
	match.rangeEnd = rangeEnd
	//mylog.Info(" set MemberVar Range ", rangeStart , " ", rangeEnd)
	match.Log.Info(" set MemberVar Range ", rangeStart , " ", rangeEnd)
}

func (match *Match) clearMemberRange( ){
	match.rangeStart = ""
	match.rangeEnd = ""
	//mylog.Info(" clear MemberVar : rangeStart  rangeEnd" )
	match.Log.Info(" clear MemberVar : rangeStart  rangeEnd")
}

/*
flag
	1:平均10等份
	2：逐渐 递增份
	3：全部
*/
//范围性匹配：搜索一定<权重>范围内的数据，一共是3次，由最粗到最细粒度
func  (match *Match)  matchingRange(flag int)(successGroupIds map[int]map[int]int,isEmpty int){
	//mylog.Info(" matchingRange , flag : ",flag)
	match.Log.Info("matchingRange , flag : ",flag)
	var tmpMin int
	var tmpMax float32
	rangeStart := ""
	rangeEnd := ""
	forStart := 0
	forEnd := 10
	if flag == FilterFlagAll{
		forEnd = 1
	}
	for i:=forStart;i<forEnd;i++{
		if flag == FilterFlagBlock{//区域性的
			tmpMin = i
			tmpMax = float32(i) + 0.9999

			rangeStart = strconv.Itoa(tmpMin)
			rangeEnd = zlib.FloatToString(tmpMax,3)
		}else if flag == FilterFlagBlockInc{//递增，范围更大
			if i == 0 {
				continue
			}
			if i == 1 {
				//因为0-1 区间上面已经计算过了，所以 从1开始
				continue
			}
			if i == 9 {
				//0-10 最开始全范围搜索已民经计算过了
				continue
			}
			tmpMin = 0
			tmpMax = float32(i) + 0.9999

			rangeStart = strconv.Itoa(tmpMin)
			rangeEnd = zlib.FloatToString(tmpMax,3)
		}else{//全部
			rangeStart = "-inf"
			rangeEnd = "+inf"
		}
		match.setMemberRange(rangeStart,rangeEnd)
		successGroupIds,isEmpty = match.searchByRange(flag)
		//mylog.Info("searchByRange rs : ",rangeStart , " ~ ",rangeStart , " , rs , cnt:",len(successGroupIds) , "isEmpty : ",isEmpty)
		match.Log.Debug("searchByRange rs : ",successGroupIds)
		match.Log.Info("searchByRange rs : ",rangeStart , " ~ ",rangeStart , " , rs , cnt:",len(successGroupIds) , "isEmpty : ",isEmpty)
		if isEmpty == 1 && flag == FilterFlagAll{
			//最大范围的查找没有数据，并且从redis读不出数据，证明就是数据太少，不可能成团
			mylog.Notice(" FilterFlagAll  isEmpty  = 1 ,break this foreach")
			match.Log.Info(" FilterFlagAll  isEmpty  = 1 ,break this foreach")
			return successGroupIds,isEmpty
		}
		//mylog.Debug("successGroupIds",successGroupIds)
		match.Log.Info("successGroupIds",successGroupIds)
		//将计算好的组ID，团队，插入到 成功 队列中
		if len(successGroupIds)> 0 {
			match.successConditions( successGroupIds)
			if flag == FilterFlagAll{
				//如果最大范围的匹配命中了数据，就证明，只有在最大范围才有数据，再细化的范围搜索已无用
				return successGroupIds,isEmpty
			}
		}
	}

	return successGroupIds,isEmpty
}
//根据具体的<权重>值，开始进行最真实的匹配
func  (match *Match) searchByRange(  flag int )(successGroupIds map[int]map[int]int,isEmpty int){
	//先做最大范围的搜索：所有报名匹配的玩家
	mylog.Info("searchByRange : ",  match.rangeStart , " , ",match.rangeEnd ,  "  , flag " ,flag)
	match.Log.Info("searchByRange : ",  match.rangeStart , " , ",match.rangeEnd ,  "  , flag " ,flag)
	playersTotal := 0
	if flag == FilterFlagAll{
		playersTotal = match.QueueSign.getAllPlayersCnt( )
	}else{
		playersTotal = match.QueueSign.getPlayersCntTotalByWeight(match.rangeStart,match.rangeEnd)
	}
	groupsTotal := match.QueueSign.getGroupsWeightCnt(match.rangeStart,match.rangeEnd)
	mylog.Info(" playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	match.Log.Info(" playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	if playersTotal <= 0 ||  groupsTotal <= 0 {
		mylog.Info("total is 0")
		match.Log.Info("total is 0")
		return successGroupIds,1
	}
	personCondition := match.Rule.PersonCondition
	if match.Rule.Flag == RuleFlagTeamVS { //组队/对战类
		personCondition = match.Rule.TeamVSPerson * 2
	}
	mylog.Info(" success condition when person =",personCondition)
	match.Log.Info(" success condition when person =",personCondition)
	if playersTotal < personCondition{
		mylog.Notice(" total < personCondition ")
		match.Log.Notice(" total < personCondition ")
		return successGroupIds,1
	}

	if flag  == FilterFlagAll && playersTotal > personCondition * 2 {
		mylog.Notice(" FilterFlagAll playersTotal > personCondition * 2 ,end this searchByRange ")
		match.Log.Info(" FilterFlagAll playersTotal > personCondition * 2 ,end this searchByRange ")
		return successGroupIds,0
	}
	//上面是基于<总数>的验证，没有问题后~,开始详情的计算
	successGroupIds = match.searchFilterDetail()
	//match.Log.Info(" matching success condition groupIds",successGroupIds)
	//zlib.MyPrint(" matching success condition groupIds",successGroupIds)
	return successGroupIds,0
}
//前面是基于各种<总数>汇总，的各种验证，都没有问题了，这里才算是正式的细节匹配
func   (match *Match) searchFilterDetail(  )(defaultSuccessGroupIds map[int]map[int]int){
	mylog.Info( "searchFilterDetail , rule flag :",RuleFlagTeamVS)
	match.Log.Info("searchFilterDetail , rule flag :",RuleFlagTeamVS)
	//实例化，使用地址引用，后面的子函数，不用返回值了
	successGroupIds := make(map[int]map[int]int)

	//N vs N ，会多一步，公平性匹配，也就是5人组优化匹配的对手也是5人组
	if match.Rule.Flag == RuleFlagTeamVS{
		/*
			共2轮过滤筛选
			1、只求互补数，为了公平：5人匹配，对手也是5人组队的，3人组队匹配到的对手里也应该有3人组队的
			   	如：5没有互补，4的互补是1，3的互补是2（2的互补是3，1互补是4，但前两步已经计算过了，这两步忽略）
			2、接上面处理过后的数据，正常排序组合
			   	上面一轮筛选过后，5人的：最多只剩下一个
				4人的，可能匹配光了，也可能剩下N个，因为取决于1人有多少，如果1人被匹配光了，4人组剩下多少个都没意义了，因为无法再组成一个团队了，所以4人组可直接忽略
				剩下 3 2 1 ，没啥快捷的办法了，只能单纯排列组合=5
		*/
		match.logarithmic( successGroupIds)
	}
	match.groupPersonCalculateNumberCombination( successGroupIds)
	//因用的是map ,预先make 下，地址就已经分配了，可能有时，其实一个值都没有用到，但是len 依然是 > 0
	if zlib.CheckMap2IntIsEmpty(successGroupIds)  {
		mylog.Notice("searchFilterDetail is empty")
		match.Log.Notice("searchFilterDetail is empty")
		return defaultSuccessGroupIds
	}
	//zlib.ExitPrint(-33)
	//for k,oneOkPersonCondition := range calculateNumberTotalRs{
	//	zlib.MyPrint(zlib.GetSpaceStr(4)+"condition " , k )
	//	zlib.ExitPrint(oneOkPersonCondition)
	//	//oneOkPersonCondition:一条可<成团>人数<分组>计算公式
	//	//循环，根据公式里要求的：某个组取几个，得出最终的groupIds
	//	oneConditionGroupIds := match.oneConditionConvertGroup(oneOkPersonCondition)
	//	successGroupIds[k] = oneConditionGroupIds
	//}

	//zlib.MyPrint(zlib.GetSpaceStr(4)+"successGroupIds",successGroupIds)

	return successGroupIds
}
func   (match *Match) groupPersonCalculateNumberCombination( successGroupIds map[int]map[int]int){
	//这里吧，按说取，一条rule最大的值就行，没必要取全局最大的5，但是吧，后面的算法有点LOW，外层循环数就是5 ，除了矩阵，太麻烦，回头我再想想
	groupPersonNum := match.QueueSign.getPlayersCntByWeight(match.rangeStart,match.rangeEnd)
	//mylog.Debug(zlib.GetSpaceStr(4)+"every group person total : ",groupPersonNum)
	match.Log.Info("every group person total : ",groupPersonNum)
	//根据组人数，做排列组合，得到最终自然数和
	calculateNumberTotalRs := match.calculateNumberTotal(match.Rule.PersonCondition,groupPersonNum)
	//mylog.Info(zlib.GetSpaceStr(4)+"calculateNumberTotalRs :",calculateNumberTotalRs)
	match.Log.Info("calculateNumberTotalRs :",calculateNumberTotalRs)
	//上面的函数，虽然计算完了，总人数是够了，但是也可能不会成团，比如：全是4人为一组报名，成团总人数却是奇数，偶数和是不可能出现奇数的
	if len(calculateNumberTotalRs) == 0{
		//mylog.Notice(zlib.GetSpaceStr(4)+"calculateNumberTotal is empty")
		match.Log.Notice("calculateNumberTotal is empty")
		return
	}
	/*
	calculateNumberTotal 函数是排列组合，它关心的是 N个数的总和，一个数可以使用N次，重复使用。而匹配：如果一个数使用了一次，下次组合就得少一个
	所以排列组合的结果，会有很多，而实际结果却会少很多
	接着分析，对两种匹配机制
	1、只要够多少人即成功，那么排列一次，随机，取其中一个算法，下一次，再排列一下
	2、N VS N PK，

	 */

	//这里是倒序，因为最大的数在最后，如5人组成立的条件，使用5最多
	inc := len(successGroupIds)
	for i:=len(calculateNumberTotalRs) - 1;i>=0;i--{
		//zlib.MyPrint(zlib.GetSpaceStr(4)+"condition " , i )
		oneConditionGroupIds := match.oneConditionConvertGroup(calculateNumberTotalRs[i])
		//successGroupIds[inc] = oneConditionGroupIds
		//inc++
		if len(oneConditionGroupIds) > 0{
			successGroupIds[inc] = oneConditionGroupIds
			inc++
		}else{
			mylog.Notice("this calculateNumberTotal condition not found ~")
		}
	}
	mylog.Debug("groupPersonCalculateNumberCombination rs :",successGroupIds)
}
func  (match *Match) oneConditionConvertGroup(oneOkPersonCondition [5]int)map[int]int{
	oneConditionGroupIds := make(map[int]int)
	inc := 0
	mylog.Debug(oneOkPersonCondition)
	someOneEmpty := 0	//redis 里取不出来数据了，或者取出的数据 小于 应取数据个数
	for index,num := range oneOkPersonCondition{
		person := index + 1
		zlib.MyPrint( zlib.GetSpaceStr(4)+" groupPerson : ",person , " get num : ",num )
		if num <= 0{
			zlib.MyPrint(zlib.GetSpaceStr(4)+" get num <= 0  ,continue")
			continue
		}
		//从redis , N人小组：集合里，取出几个groupId
		groupIds  := match.QueueSign.getGroupPersonIndexList(person ,match.rangeStart,match.rangeEnd,0,num,true)
		if len(groupIds) == 0{
			someOneEmpty = 1
			mylog.Notice("getGroupPersonIndexList empty")
			break
		}

		if len(groupIds) != num {
			someOneEmpty = 1
			mylog.Notice("getGroupPersonIndexList != num ", len(groupIds) , num)
			break
		}


		//zlib.MyPrint("groupIds",groupIds)
		for i:=0;i < len(groupIds);i++{
			oneConditionGroupIds[inc] = groupIds[i]
			inc++
			//match.QueueSign.delOneGroupIndex(groupIds[i])
		}
		//zlib.MyPrint("groupIds : ",groupIds)
	}
	if someOneEmpty == 1{
		//重置：该变量
		oneConditionGroupIds = make(map[int]int)
	}
	return oneConditionGroupIds
}
//N V N  ,互补数/对数
func  (match *Match)logarithmic( successGroupIds map[int]map[int]int){
	//mylog.Info(zlib.GetSpaceStr(3)+ " action RuleFlagTeamVS logarithmic :")
	match.Log.Info("logarithmic")
	groupPersonNum := match.QueueSign.getPlayersCntByWeight(match.rangeStart,match.rangeEnd)
	//mylog.Debug(zlib.GetSpaceStr(2),"groupPersonTotal , ",groupPersonNum)
	match.Log.Info("groupPersonTotal , ",groupPersonNum)
	successGroupIdsInc := 0
	//已处理过的互补数，如：4计算完了，1就不用算了，3计算完了，2其实也不用算了，
	var processedNumber []int
	wishSuccessGroups := 0 //预计应该成功的  团队~  用于统计debug
	for personNum,personTotal := range groupPersonNum{
		//mylog.Debug(zlib.GetSpaceStr(4)+"foreach groupPerson , person " , personNum, " ,  personTotal  ",personTotal , "successGroupIdsInc",successGroupIdsInc)
		match.Log.Info("foreach groupPerson , person " , personNum, " ,  personTotal  ",personTotal , "successGroupIdsInc",successGroupIdsInc)
		//判断是否已经处理过了
		elementInArrIndex := zlib.ElementInArrIndex(processedNumber,personNum)
		if elementInArrIndex != -1 {
			zlib.MyPrint(zlib.GetSpaceStr(4),"has processedNumber : ",personNum)
			continue
		}
		//5或者设置的最大值，已知最大的，且直接满足不做处理
		if personNum == match.Rule.TeamVSPerson{
			//mylog.Debug(zlib.GetSpaceStr(4),"in max TeamVSPerson , no need remainder number")
			match.Log.Info(zlib.GetSpaceStr(4),"in max TeamVSPerson , no need remainder number")
			if personTotal <= 1 {//<5人组>如果只有一个的情况，满足不了条件
				//mylog.Notice(zlib.GetSpaceStr(4),"in max TeamVSPerson 1 , but personTotal <= 1 , continue")
				match.Log.Notice("in max TeamVSPerson 1 , but personTotal <= 1 , continue")
				continue
			}
			maxNumber := match.getMinPersonNum(personTotal,personTotal)
			//mylog.Debug(zlib.GetSpaceStr(4),"maxNumber " , maxNumber)
			match.Log.Info("maxNumber " , maxNumber)
			if maxNumber <= 0{
				//mylog.Notice(zlib.GetSpaceStr(4),"in max TeamVSPerson 2 , but personTotal <= 1 , continue")
				match.Log.Notice("in max TeamVSPerson 2 , but personTotal <= 1 , continue")
				continue
			}
			wishSuccessGroups += maxNumber / 2
			//取出集合中，所有人数为5的组ids
			groupIds := match.QueueSign.getGroupPersonIndexList(match.Rule.TeamVSPerson,"-inf","+inf",0,maxNumber,true)
			for i:=0;i < maxNumber / 2;i++{
				tmp := make(map[int]int)
				tmp[0] = groupIds[i]
				tmp[1] = groupIds[i+1]
				successGroupIds[successGroupIdsInc] = tmp
				successGroupIdsInc++
			}
			//zlib.MyPrint("person 5 final groupIds",groupIds)
			continue
		}

		//团队最大值 - 当前人数 = 需要补哪个<组人数>   补数
		needRemainderNum := match.Rule.TeamVSPerson - personNum
		if groupPersonNum[needRemainderNum] <=0 {
			//互补值 不存在 ，或者 互补值 人数 为 0
			continue
		}
		maxNumber := match.getMinPersonNum(personTotal,groupPersonNum[needRemainderNum])
		mylog.Debug(zlib.GetSpaceStr(4)+"needNumber : ",needRemainderNum , "needNumberPersonTotal", groupPersonNum[needRemainderNum], " maxNumber : ",maxNumber  )
		if maxNumber <= 0{
			continue
		}
		setA := match.QueueSign.getGroupPersonIndexList(needRemainderNum,match.rangeStart,match.rangeEnd,0,maxNumber,true)
		setB := match.QueueSign.getGroupPersonIndexList(personNum,match.rangeStart,match.rangeEnd,0,maxNumber,true)
		//逐条合并 setA setB
		for k,_ := range setA{
			tmp := make(map[int]int)
			tmp[0] = setA[k]
			tmp[1] = setB[k]
			successGroupIds[successGroupIdsInc] = tmp
			successGroupIdsInc++
		}

		wishSuccessGroups += maxNumber

		processedNumber =  append(processedNumber,needRemainderNum)
	}
	//mylog.Debug("wishSuccessGroups :",wishSuccessGroups)
	//mylog.Info(zlib.GetSpaceStr(4),"logarithmic rs : ",successGroupIds)
	match.Log.Debug("logarithmic rs : ",successGroupIds)
	//zlib.ExitPrint(-11)
}
//互补数中，有一步是，取：两个互补数，人数的，最小的那个
func  (match *Match)getMinPersonNum(personTotal int,needRemainderNumPerson int)int{
	maxNumber := personTotal
	if needRemainderNumPerson < maxNumber{
		maxNumber = needRemainderNumPerson
	}
	divider := personTotal % 2
	if divider > 0 {//证明是奇数个
		maxNumber--
	}
	return maxNumber
}
//走到这里就证明，有匹配成功的玩家了
//当匹配成功最终，成功筛选出匹配的组/玩家后，该函数开始执行后续插入操作
//注：successGroupIds ，这里只是组ID，是从索引里拿出来的，只是单一删除了索引值，并没有删除真正的组信息
func  (match *Match)successConditions( successGroupIds map[int]map[int]int){
	length := len(successGroupIds)
	mylog.Info("successConditions  ...   len :   ",length)
	match.Log.Info("successConditions  ...   len :   ",length)
	if match.Rule.Flag == RuleFlagCollectPerson{//满足人数即开团
		match.Log.Debug("case : RuleFlagCollectPerson")
		//zlib.MyPrint("successGroupIds",successGroupIds)
		for _,oneCondition:=range successGroupIds{
			match.Log.Info("oneCondition : ",oneCondition)
			resultElement := match.QueueSuccess.NewResult()
			match.Log.Info("newr ResultElement struct")
			//zlib.MyPrint("new resultElement : ",resultElement)
			teamId := 1
			groupIdsArr := make( map[int]int)
			playerIdsArr:= make( map[int]int)
			for _,groupId := range oneCondition{
				match.successConditionAddOneGroup( resultElement.Id,groupId,teamId,groupIdsArr,playerIdsArr)
			}

			//zlib.MyPrint("groupIdsArr",groupIdsArr,"playerIdsArr",playerIdsArr)
			resultElement.GroupIds = zlib.MapCovertArr(groupIdsArr)
			resultElement.PlayerIds = zlib.MapCovertArr(playerIdsArr)
			resultElement.Teams = []int{teamId}
			//zlib.MyPrint("resultElement",resultElement)
			match.Log.Info("QueueSuccess.addOne",resultElement)
			match.QueueSuccess.addOne( resultElement,match.Push)

		}
	}else{//组队互相PK
		match.Log.Debug("case : RuleFlagTeamVS")
		if length == 1{
			match.groupPushBackCondition(successGroupIds[0])
			mylog.Notice("successGroupIds length = 1 , break")
			match.Log.Notice("successGroupIds length = 1 , break")
			return
		}
		if length % 2 > 0{
			match.Log.Notice("have single group!!!")
			//组队PK，肯定是至少有2个组，如果出现奇数，证明肯定最后一个不能用了
			//把最后一个数，塞回到redis里，再清空这个数
			index := len(successGroupIds)-1
			match.groupPushBackCondition(successGroupIds[index])
			successGroupIds[index] = nil
			length--
		}
		mylog.Info("final success cnt : ",successGroupIds , " length : ",length)
		var teamId int
		var resultElement Result

		var groupIdsArr map[int]int
		var playerIdsArr map[int]int
		for i:=0;i<length;i++ {
			//zlib.MyPrint(successGroupIds[i],i)
			if len(successGroupIds[i]) == 1{
				mylog.Info(" has a single")
				continue
			}
			if i % 2 == 0{
				resultElement = match.QueueSuccess.NewResult()
				teamId = 1
				groupIdsArr = make( map[int]int)
				playerIdsArr = make( map[int]int)

				mylog.Notice("groupId : ",resultElement.Id)
			}

			for _,groupId := range successGroupIds[i]{
				match.successConditionAddOneGroup(resultElement.Id,groupId,teamId,groupIdsArr,playerIdsArr)
			}

			resultElement.GroupIds = zlib.MapCovertArr(groupIdsArr)
			resultElement.PlayerIds = zlib.MapCovertArr(playerIdsArr)
			//resultElement.Teams = []int{teamId}
			teamId = 2
			if i % 2 == 1{
				teamIds := []int{1,2}
				resultElement.Teams = teamIds
				match.Log.Info("QueueSuccess.addOne",resultElement)
				match.QueueSuccess.addOne(resultElement ,match.Push)
			}
		}
		//zlib.ExitPrint(123123)
	}
	mylog.Info("finish successConditions  ...")
	match.Log.Info("finish successConditions  ...")
}
//取出来的groupIds 可能某些原因 最终并没有用上，但是得给塞回到redis里
//这里其实只是将index数据补充上即可，因为计算的时候，删的也只是索引值
func (match *Match)groupPushBackCondition(oneCondition map[int]int){
	mylog.Info("groupPushBackCondition")
	match.Log.Info("groupPushBackCondition", oneCondition)
	for _,v := range oneCondition{
		match.QueueSign.addOneGroupIndex(v)
	}
}
func (match *Match)successConditionAddOneGroup( resultId int,groupId int,teamId int,groupIdsArr  map[int]int ,playerIdsArr map[int]int)Group{
	mylog.Info("successConditionAddOneGroup")
	match.Log.Info("successConditionAddOneGroup")
	group := match.QueueSign.getGroupElementById( groupId )
	//mylog.Debug("getGroupElementById group ",group)
	//groupIdsArr = append(  (*groupIdsArr),groupId)
	groupIdsArr[len(groupIdsArr)] = groupId
	playerIdsArrInc := len(playerIdsArr)
	for _,player := range group.Players{
		playerIdsArr[playerIdsArrInc] = player.Id
		playerIdsArrInc++
	}

	SuccessGroup := group
	SuccessGroup.SuccessTimeout = zlib.GetNowTimeSecondToInt() + match.Rule.SuccessTimeout
	SuccessGroup.LinkId = resultId
	SuccessGroup.SuccessTime = zlib.GetNowTimeSecondToInt()
	SuccessGroup.TeamId = teamId
	//fmt.Printf("%+v",SuccessGroup)
	//zlib.ExitPrint(222)
	match.Log.Info("addOneGroup",SuccessGroup)
	match.QueueSuccess.addOneGroup(SuccessGroup)

	match.Log.Notice("delSingOldGroup",groupId)
	match.QueueSign.delOneRuleOneGroup(groupId,0)
	for _,v := range group.Players{
		PlayerStatusElement := playerStatus.GetOne(v)

		newPlayerStatusElement := PlayerStatusElement
		newPlayerStatusElement.Status = PlayerStatusSuccess
		newPlayerStatusElement.SuccessTimeout = group.SuccessTimeout
		playerStatus.upInfo(PlayerStatusElement,newPlayerStatusElement)
		match.Log.Info("playerStatus.upInfo ", "oldStatus : ",PlayerStatusElement.Status,"newStatus : ",newPlayerStatusElement.Status)
	}
	//zlib.MyPrint( "add one group : ")
	//fmt.Printf("%+v",SuccessGroup)
	return group
}
