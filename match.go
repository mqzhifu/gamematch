package gamematch

import (
	"src/zlib"
	"strconv"
	"sync"
)

const (
	FilterFlagAll     =1
	FilterFlagBlock	  =2
	FilterFlagBlockInc=3
)

type Match struct {
	Mutex 	sync.Mutex
	Rule 	Rule
	QueueSign *QueueSign
	QueueSuccess *QueueSuccess
	Push 	*Push
	rangeStart	string
	rangeEnd	string
}

func NewMatch(rule Rule,queueSign *QueueSign,queueSuccess *QueueSuccess,push *Push)*Match{
	match := new(Match)
	match.Rule = rule
	match.QueueSign = queueSign
	match.QueueSuccess = queueSuccess
	match.Push = push
	match.rangeStart = ""
	match.rangeEnd = ""
	return match
}
//一条rule 的匹配
func (match *Match) start(){
	zlib.MyPrint("start one rule queueMatching : ",match.Rule.Id,match.Rule.CategoryKey)
	for{
	 	match.matching()
		zlib.ExitPrint(-777)
		mySleepSecond(1)
	}
}

func  (match *Match)  matching() {
	zlib.MyPrint("new once matching func :")
	match.clearMemberRange( )
	//gamematch.getLock()
	//defer gamematch.unLock()

	playersTotal := match.QueueSign.getAllPlayersCnt()
	groupsTotal := match.QueueSign.getAllGroupsWeightCnt()
	zlib.MyPrint(" final total start , playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)

	//先做最大范围搜索：也就是整体看下集合全部元素
	matchingRange := []int{FilterFlagAll,FilterFlagBlock,FilterFlagBlockInc}
	for i:=0;i<len(matchingRange);i++{
		groupIds,isEmpty := match.matchingRange(matchingRange[i])
		if isEmpty == 1{
			return
		}
		if matchingRange[i] == FilterFlagAll{
			if len(groupIds) >0 {
				return
			}
		}
	}

	playersTotal = match.QueueSign.getAllPlayersCnt()
	groupsTotal = match.QueueSign.getAllGroupsWeightCnt()
	zlib.MyPrint("final total end, playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	match.clearMemberRange( )
}

func (match *Match) setMemberRange(rangeStart string,rangeEnd string){
	match.rangeStart = rangeStart
	match.rangeEnd = rangeEnd
	zlib.MyPrint(" setMemberRange ", rangeStart , " ", rangeEnd)
}

func (match *Match) clearMemberRange( ){
	match.rangeStart = ""
	match.rangeEnd = ""
	zlib.MyPrint(" clearMemberRange " )
}

/*
flag
	1:平均10等份
	2：逐渐 递增份
	3：全部
*/
//匹配：范围性的，搜索与过滤
func  (match *Match)  matchingRange(flag int)(successGroupIds map[int]map[int]int,isEmpty int){
	zlib.MyPrint(" matchingRange , flag : ",flag)
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
		}else{
			rangeStart = "-inf"
			rangeEnd = "+inf"
		}
		match.setMemberRange(rangeStart,rangeEnd)
		successGroupIds,isEmpty = match.searchFilterAll(flag)
		zlib.MyPrint("search range : ",rangeStart , " ~ ",rangeStart , " , rs , cnt:",len(successGroupIds) , "isEmpty : ",isEmpty)
		if isEmpty == 1 && flag == FilterFlagAll{
			zlib.MyPrint(" isEmpty  = 1 ")
			return successGroupIds,isEmpty
		}
		zlib.MyPrint("successGroupIds",successGroupIds)
		//将计算好的组ID，团队，插入到 成功 队列中
		if len(successGroupIds)> 0 {
			match.successConditions( successGroupIds)
			if flag == FilterFlagAll{
				return successGroupIds,isEmpty
			}
		}
	}

	return successGroupIds,isEmpty
}
//匹配时，开始应该先做全局匹配
func  (match *Match) searchFilterAll(  flag int )(successGroupIds map[int]map[int]int,isEmpty int){
	//先做最大范围的搜索：所有报名匹配的玩家
	zlib.MyPrint(" searchFilterAll :  rangeStart ",  match.rangeStart , " , rangeEnd  ",match.rangeEnd ,  "  , flag " ,flag)
	playersTotal := 0
	if flag == FilterFlagAll{
		playersTotal = match.QueueSign.getAllPlayersCnt( )
	}else{
		playersTotal = match.QueueSign.getPlayersCntTotalByWeight(match.rangeStart,match.rangeEnd)
	}
	groupsTotal := match.QueueSign.getGroupsWeightCnt(match.rangeStart,match.rangeEnd)
	zlib.MyPrint(" playersTotal total:",playersTotal , " groupsTotal : ",groupsTotal)
	if playersTotal <= 0 ||  groupsTotal <= 0 {
		zlib.MyPrint("total is 0")
		return successGroupIds,1
	}
	personCondition := match.Rule.PersonCondition
	if match.Rule.Flag == RuleFlagTeamVS { //组队/对战类
		personCondition = match.Rule.TeamVSPerson * 2
	}
	zlib.MyPrint(" rule final personCondition:",personCondition)

	if playersTotal < personCondition{
		zlib.MyPrint(" total < personCondition ")
		return successGroupIds,1
	}

	if playersTotal > personCondition * 2 && flag  == FilterFlagAll {
		zlib.MyPrint(" FilterFlagAll playersTotal > personCondition * 2 ")
		return successGroupIds,0
	}
	//上面是粗略的验证，没有问题后~
	successGroupIds = match.searchFilterDetail()
	zlib.MyPrint(" matching success condition groupIds",successGroupIds)
	return successGroupIds,0
}
//前面是基于各种<总数>汇总，的各种验证，都没有问题了，这里才算是正式的细节匹配
func   (match *Match) searchFilterDetail(  )(defaultSuccessGroupIds map[int]map[int]int){
	zlib.MyPrint(zlib.GetSpaceStr(3)+"searchFilterDetail ： ")
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
		zlib.MyPrint(zlib.GetSpaceStr(3)+ " RuleFlagTeamVS logarithmic :")
		match.logarithmic( successGroupIds)
	}
	match.groupPersonCalculateNumberCombination( successGroupIds)
	//因用的是map ,预先make 下，地址就已经分配了，可能有时，其实一个值都没有用到，但是len 依然是 > 0
	if zlib.CheckMap2IntIsEmpty(successGroupIds)  {
		zlib.MyPrint("searchFilterDetail is empty")
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

	zlib.MyPrint(zlib.GetSpaceStr(4)+"successGroupIds",successGroupIds)

	return successGroupIds
}
func   (match *Match) groupPersonCalculateNumberCombination( successGroupIds map[int]map[int]int){
	//这里吧，按说取，一条rule最大的值就行，没必要取全局最大的5，但是吧，后面的算法有点LOW，外层循环数就是5，目前没想到太快的解决办法
	//除了矩阵，太麻烦，回头我再想想
	groupPersonNum := match.QueueSign.getPlayersCntByWeight(match.rangeStart,match.rangeEnd)
	zlib.MyPrint(zlib.GetSpaceStr(4)+"every group person total : ",groupPersonNum)
	//根据组人数，做排列组合，得到最终自然数和
	calculateNumberTotalRs := match.calculateNumberTotal(match.Rule.PersonCondition,groupPersonNum)
	zlib.MyPrint(zlib.GetSpaceStr(4)+"calculateNumberTotalRs :",calculateNumberTotalRs)
	//上面的函数，虽然计算完了，总人数是够了，但是也可能不会成团，比如：全是4人为一组报名，成团总人数却是奇数，偶数和是不可能出现奇数的
	if len(calculateNumberTotalRs) == 0{
		zlib.MyPrint(zlib.GetSpaceStr(4)+"calculateNumberTotal is empty")
		return
	}
	/*
	calculateNumberTotal 函数是排列组合，它关心的是 N个数的总和，一个数可以使用N次，重复使用。而匹配：如果一个数使用了一次，下次组合就得少一个
	所以排列组合的结果，会有很多，而实际结果却会少很多
	接着分析，对两种匹配机制
	1、只要够多少人即成功，那么排列一次，随机，取其中一个算法，下一次，再排列一下
	2、N VS N PK，

	 */

	for k,oneOkPersonCondition := range calculateNumberTotalRs{
		zlib.MyPrint(zlib.GetSpaceStr(4)+"condition " , k )
		oneConditionGroupIds := match.oneConditionConvertGroup(oneOkPersonCondition)
		successGroupIds[k] = oneConditionGroupIds
	}
}
func  (match *Match) oneConditionConvertGroup(oneOkPersonCondition [5]int)map[int]int{
	oneConditionGroupIds := make(map[int]int)
	inc := 0
	for index,num := range oneOkPersonCondition{
		person := index + 1
		zlib.MyPrint( zlib.GetSpaceStr(4)+" groupPerson : ",person , " get num : ",num )
		if num <= 0{
			zlib.MyPrint(zlib.GetSpaceStr(4)+" get num <= 0  ,continue")
			continue
		}
		//从redis , N人小组：集合里，取出几个groupId
		groupIds  := match.QueueSign.getGroupPersonIndexList(person ,match.rangeStart,match.rangeEnd,0,num)
		//zlib.MyPrint("groupIds",groupIds)
		for i:=0;i < len(groupIds);i++{
			oneConditionGroupIds[inc] = groupIds[i]
			inc++
			//match.QueueSign.delOneGroupIndex(groupIds[i])
		}
		//zlib.MyPrint("groupIds : ",groupIds)
	}
	return oneConditionGroupIds
}
//N V N  ,互补数/对数
func  (match *Match)logarithmic( successGroupIds map[int]map[int]int){

	groupPersonNum := match.QueueSign.getPlayersCntByWeight(match.rangeStart,match.rangeEnd)
	zlib.MyPrint(zlib.GetSpaceStr(4),"groupPersonNum , ",groupPersonNum)
	successGroupIdsInc := 0
	//已处理过的互补数，如：4计算完了，1就不用算了，3计算完了，2其实也不用算了，
	var processedNumber []int
	for personNum,personTotal := range groupPersonNum{
		zlib.MyPrint(zlib.GetSpaceStr(4)+" foreach personNum : " , personNum, " , personTotal  ",personTotal)
		//判断是否已经处理过了
		elementInArrIndex := zlib.ElementInArrIndex(processedNumber,personNum)
		if elementInArrIndex != -1 {
			zlib.MyPrint(zlib.GetSpaceStr(4),"has processedNumber : ",personNum)
			continue
		}
		//5或者设置的最大值，已知最大的，且直接满足不做处理
		if personNum == match.Rule.TeamVSPerson{
			zlib.MyPrint(zlib.GetSpaceStr(4),"in max TeamVSPerson")
			if personTotal <= 1 {//<5人组>如果只有一个的情况，满足不了条件
				zlib.MyPrint(zlib.GetSpaceStr(4)," personTotal <= 1 , continue")
				continue
			}
			//maxNumber := personTotal
			//divider := personTotal % 2
			//if divider > 0 {//证明是奇数个
			//	maxNumber--
			//}
			maxNumber := match.getMinPersonNum(personTotal,personTotal)
			zlib.MyPrint(zlib.GetSpaceStr(4)," maxNumber " , maxNumber)
			if maxNumber <= 0{
				continue
			}
			//取出集合中，所有人数为5的组ids
			groupIds := match.QueueSign.getGroupPersonIndexList(match.Rule.TeamVSPerson,"-inf","+inf",0,maxNumber)
			for i:=0;i < maxNumber;i++{
				tmp := make(map[int]int)
				tmp[0] = groupIds[i]
				successGroupIds[successGroupIdsInc] = tmp
				successGroupIdsInc++
			}
			continue
		}

		//团队最大值 - 当前人数 = 需要补哪个<组人数>   补数
		needRemainderNum := match.Rule.TeamVSPerson - personNum
		if groupPersonNum[needRemainderNum] <=0 {
			//互补值 不存在 ，或者 互补值 人数 为 0
			continue
		}
		maxNumber := match.getMinPersonNum(personTotal,groupPersonNum[needRemainderNum])
		zlib.MyPrint(zlib.GetSpaceStr(4)+"needRemainderNum : ",needRemainderNum , " maxNumber : ",maxNumber  )
		if maxNumber <= 0{
			continue
		}
		setA := match.QueueSign.getGroupPersonIndexList(needRemainderNum,match.rangeStart,match.rangeEnd,0,maxNumber)
		setB := match.QueueSign.getGroupPersonIndexList(personNum,match.rangeStart,match.rangeEnd,0,maxNumber)
		//逐条合并 setA setB
		for k,_ := range setA{
			tmp := make(map[int]int)
			tmp[0] = setA[k]
			tmp[1] = setB[k]
			successGroupIds[successGroupIdsInc] = tmp
			successGroupIdsInc++
		}
		processedNumber =  append(processedNumber,needRemainderNum)
	}
	zlib.MyPrint( zlib.GetSpaceStr(4),"logarithmic rs : ",successGroupIds)
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
func  (match *Match)successConditions( successGroupIds map[int]map[int]int){
	zlib.MyPrint(" in  successConditions  ...     ")
	if match.Rule.Flag == RuleFlagCollectPerson{
		zlib.MyPrint("successGroupIds",successGroupIds)
		for _,oneCondition:=range successGroupIds{
			zlib.MyPrint(oneCondition)
			resultElement := match.QueueSuccess.NewResult()
			zlib.MyPrint("new resultElement : ",resultElement)
			teamId := 1
			groupIdsArr := make( map[int]int)
			playerIdsArr:= make( map[int]int)
			for _,groupId := range oneCondition{
				match.successConditionAddOneGroup( resultElement.Id,groupId,teamId,groupIdsArr,playerIdsArr)
			}

			zlib.MyPrint(groupIdsArr,playerIdsArr)
			resultElement.GroupIds = zlib.MapCovertArr(groupIdsArr)
			resultElement.PlayerIds = zlib.MapCovertArr(playerIdsArr)
			resultElement.Teams = []int{teamId}
			zlib.MyPrint("resultElement",resultElement)
			match.QueueSuccess.addOne( resultElement,match.Push)

		}
	}else{//组队互相PK
		length := len(successGroupIds)
		zlib.MyPrint("length : ",length)
		if length == 1{
			zlib.MyPrint("successGroupIds length = 1")
			return
		}
		if length / 2 > 0{
			index := len(successGroupIds)-1
			//delete(successGroupIds,successGroupIds[index])
			successGroupIds[index] = nil
			//这里是把最后一个奇数置 空，但是还得把这个值 再给塞回到redis里
			//pushBack
			length--
		}
		var teamId int
		var resultElement Result

		var groupIdsArr map[int]int
		var playerIdsArr map[int]int
		for i:=0;i<length;i++ {
			if i % 2 == 0{
				resultElement = match.QueueSuccess.NewResult()
				teamId = 1
				groupIdsArr = make( map[int]int)
				playerIdsArr = make( map[int]int)
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
				match.QueueSuccess.addOne(resultElement ,match.Push)
			}
		}
		zlib.ExitPrint(123123)
	}
}
func (match *Match)successConditionAddOneGroup( resultId int,groupId int,teamId int,groupIdsArr  map[int]int ,playerIdsArr map[int]int)Group{
	group := match.QueueSign.getGroupElementById( groupId )
	zlib.MyPrint("group ",group)
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
	match.QueueSuccess.addOneGroup(SuccessGroup)

	//match.QueueSign.delOneRuleOneGroup(groupId)
	//for _,v := range group.Players{
	//	PlayerStatusElement := playerStatus.GetOne(v)
	//
	//	newPlayerStatusElement := PlayerStatusElement
	//	newPlayerStatusElement.Status = PlayerStatusSuccess
	//	newPlayerStatusElement.SuccessTimeout = group.SuccessTimeout
	//	playerStatus.upInfo(PlayerStatusElement,newPlayerStatusElement)
	//}
	//zlib.MyPrint( "add one group : ")
	//fmt.Printf("%+v",SuccessGroup)
	return group
}
