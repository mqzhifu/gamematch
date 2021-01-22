package gamematch
//
//import (
//	"fmt"
//	"src/zlib"
//)
//
////使用队列，不支持权重
//func (gamematch *Gamematch) queueMatching(rule Rule){
//	queueName := queueSign.getRedisKey(rule)
//	fmt.Printf("%+v",rule)
//	for{
//		//先做 全局判定，如果队列里的玩家总数过少，直接等待
//		personsNum := queueSign.length(queueName)
//		zlib.MyPrint("personsNum wait matching : ",personsNum)
//		personsNumIf :=rule.PersonCondition
//		if rule.Flag == RuleFlagVS { //组队，对战类
//			//目前权支持两个队伍互相PK，不支持2个以上
//			personsNumIf = rule.PersonCondition * 2
//		}
//
//		if personsNumIf < rule.PersonCondition{
//			gamematch.matchingSleep(1)
//			continue
//		}
//
//		matchingTimes := 1
//		if rule.Flag == RuleFlagVS { //组队，对战类
//			matchingTimes = 2
//		}
//
//		successPlayers := make(map[int]QueueSignElement)
//		successPlayersInc := 0
//		for i:=0; i < matchingTimes;i++{
//			successPlayersOnce, queueIsEmpty := gamematch.matching(queueName,rule)
//			if(queueIsEmpty == 1){
//				gamematch.matchingSleep(1)
//				continue
//			}
//
//			if len(successPlayersOnce) != rule.PersonCondition{
//				//又是个极端情况 ，按说matching函数返回的，应该就是想要的人数，但是出错了
//				zlib.ExitPrint("matching person num is err ",successPlayersOnce)
//			}
//			for _,v := range successPlayersOnce {
//				//v.TeamId = i + 1
//				successPlayers[successPlayersInc] = v
//				successPlayersInc++
//			}
//		}
//
//		queueSuccess.PushOne(successPlayers,rule)
//		for _,v := range successPlayers {
//			playerStatus.upStatus(v.PlayerId,PlayerStatusSuccess)
//		}
//	}//end for dead loop
//}
//
////开始一次匹配，停止条件 ：1、成功，2、队列已经空了
//func (gamematch *Gamematch) matching(queueName string,rule Rule)( map[int]QueueSignElement, int){
//	successPlayers := make( map[int]QueueSignElement )//保存已匹配成功的玩家
//	successPlayersInc := 0//全局自增ID，用于map 的KEY
//	backPushPlayers := make( map[int]QueueSignElement )//取出来的数据，因为异常，不能使用，还得再PUSH回队列中
//	backPushPlayersInc := 0//全局自增ID，用于map 的KEY
//	isSuccess := 0	//标识，又来结束死循环，最终是否成功匹配出一个组队
//	queueIsEmpty := 0 //队列是否已经空了
//	queueSign.Mutex.Lock()
//	for{
//		//判断，当前取出的人，已经满足条件，终于死循环
//		if len(successPlayers) == rule.PersonCondition{
//			isSuccess = 1
//			break
//		}
//
//		//PS,注：只有上面这种情况，和队列已经无数据，可以使用break终止死循环，其余的出错，只可以使用config
//
//		groupPlayerMatchFlag := 0 //0 :正常
//		//1 当组里有任何一个超时，其它人就不用再做判断了，其它的均丢弃
//		//2 当组里有任何一个状态不为：等待匹配中，其它人就不用再做判断了，其它的均丢弃
//		//3 以组为单位，有时候会死循环一直匹配，如：3V3，但是报名的只有2个组，且每个组是两个人，这个永远都不会匹配成功，所以得加上匹配次数超时
//		//4 队列已经空了，以组为单位，缺失玩家了
//		//5 以组为单位，缺失玩家了
//		//6 正常已经匹配的玩家 + 新弹出的一组玩家 > 满足条件的人数
//		//7 玩家已匹配超过3次了
//
//		//队列，弹出一个组的玩家，可能是N个
//		queueSignElements,isEmpty,firstGroupPerson,firstGroupId := queueSign.PopOneByGroup(rule)
//		if isEmpty{//队列已经空了
//			queueIsEmpty = 1
//			if len(queueSignElements) > 0 {
//				//这里是个极端情况，正常，如果一次只读一条，是不会出现的
//				//只有以组为单位时，原本一个组应该有N个玩家，但是少数据，导致没有读全所有组内玩家
//				groupPlayerMatchFlag = 4
//			}
//			continue
//		}
//		if len(queueSignElements) <=0 {
//			//这种情况，目测不会发生，但是先保留吧
//			zlib.MyPrint("group len queueSignElements = 0")
//		}
//		//极端情况，以第一个玩家的组人数，弹出数据，但是最终取出的数组却与 组人数，对不上
//		if len(queueSignElements) != firstGroupPerson{
//			groupPlayerMatchFlag = 5
//			continue
//		}
//		//已成功匹配的玩家 + 刚取出来的一组玩家 > 满足条件的人数
//		if len(successPlayers) + len(queueSignElements) > rule.PersonCondition{
//			groupPlayerMatchFlag = 6
//			//虽然不满足，但是得把这组的数据给重新PUSH到回队列中
//			for _,v := range queueSignElements{
//				backPushPlayers[backPushPlayersInc] =v
//				backPushPlayersInc++
//			}
//			continue
//		}
//		//基础验证结束后，开始验证每个玩家的状态,以组为单位，只要有一个出错，即全错
//		for _,element := range queueSignElements{
//			//不能一直循环重复的匹配相同的玩家，有些玩家可能就是永远也匹配不上
//			if element.MatchTimes >= 3{
//				groupPlayerMatchFlag = 7
//				break
//			}
//			//又是极端情况，以组为弹出的时候，是第一个玩家的组信息为基础 ，向后面取，按说后面的玩家的组ID应该都一致的，但是有总是
//			if element.GroupId != firstGroupId{
//				groupPlayerMatchFlag = 3
//			}
//			//多人时：也是极端情况，按说一个小组的人，是同时报名，同时超时，如果有超时的应该是整组一起超时
//			isTimeout := queueSign.CheckTimeout(element)
//			if isTimeout{
//				groupPlayerMatchFlag = 1
//				break
//			}
//			//这里还要再判断一下玩家的 状态，因为期间可能发生变化，比如：玩家取消了匹配，玩家断线了等
//			tmpPlayer := Player{Id:element.PlayerId}
//			playerStatus := playerStatus.GetOne(tmpPlayer)
//			if playerStatus.Status != PlayerStatusSign{
//				groupPlayerMatchFlag = 2
//				break
//			}
//		}
//		//证明这一组玩家的数据，已经发生了错误，停止本次大循环
//		if groupPlayerMatchFlag != 0{
//			continue
//		}
//		//到这里就证明，这一组玩家数据是OK的，放入到全局  成功匹配玩家 MAP 中
//		for _,queueSignElement := range queueSignElements{
//			successPlayers[successPlayersInc] = queueSignElement
//			successPlayersInc++
//		}
//	}//matching for dead loop
//	if len(backPushPlayers) > 0{//将正确的数据，但是不满足条件的玩家，重新塞回到队列中
//		for _,v := range backPushPlayers{
//			//虽然放回去了，但是得标记一下：已匹配过一次了
//			v.MatchTimes++
//			//queueSign.PushOne(v,queueName)
//		}
//	}
//	queueSign.Mutex.Unlock()
//	zlib.MyPrint("isSuccess:",isSuccess)
//	return successPlayers ,queueIsEmpty
//}