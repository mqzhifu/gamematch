package gamematch

import "src/zlib"

/*
	0. 加锁
	0. 先，从最大的范围搜索，扫整个集合，如果元素过少，直接在这个维度就结束了
	0. 缩小搜索范围，把整个集合，划分成10个维度，每个维度单纯计算，如果成功，那就结束了，如果人数过多，还会再划分一次最细粒度的匹配
	0. 这种是介于上面两者中间~即不是全部集合，也不是单独计算一个维度，而是逐步，放大到:最大90%集合，1-1，1-2....1-9
	0. 解锁

	总结：以上的算法，其实就是不断切换搜索范围（由最大到中小，再到中大），加速匹配时间
*/


//
// 组    group ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值     ( = ,>= ,<=, > X < , >= X < ,>= X <= X )
// 单用户 normal ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值
// 团队  team   ( 属性A * 百分比 + 属性B * 百分比 ... )  = 最终权重值

/*
fof :

1元		2张
2元		5张
3元		40张
4元		30张
5元		60张

有序集合_小组人数

	weight groupId
有序集合
	小组权重	小组ID
	weight 	groupId
hash_groupId
	小组详情信息
	groupId
		weight
		person
		timeout
		ATime
hash_groupId_players
	小组成员

集合
	groupPlayers	playerId
*/

func (gamematch *Gamematch)calculateNumberTotal(sum int,groupPerson map[int]int)map[int][5]int{
	//1人组：1-个
	//2人组：4个
	//3人组：2个
	//4人组：6个
	//5人组：3个

	result := make(map[int][5]int)
	inc := 0
	for a:=0;a<=groupPerson[1];a++{
		for b:=0;b<=groupPerson[2];b++{
			for c:=0;c<=groupPerson[3];c++{
				for d:=0;d<=groupPerson[4];d++{
					for e:=0;e<=groupPerson[5];e++{
						if a + 2 * b + 3 * c + 4 * d + 5 * e == sum {
							ttt := [5]int{a,b,c,d,e}
							result[inc] = ttt
							zlib.MyPrint("1 x ",a," + 2 x",b,"+ 3 x ",c," + 4 x ",d , " 5 x ",e,"=",sum)
							inc++
						}
					}
				}
			}
		}
	}
	return result

	//aCnt := groupPerson[0]
	//bCnt := 2
	//cCnt := 20
	//
	//inc := 0
	//for a:=0;a<=aCnt;a++{
	//	for b:=0;b<=bCnt;b++{
	//		for c:=0;c<=cCnt;c++{
	//			if a + 2 * b + 5 * c == sum {
	//				inc++
	//				zlib.MyPrint("1 x ",a," + 2 x",b,"+ 5 x ",c,"=",sum)
	//			}
	//		}
	//	}
	//}
	//zlib.ExitPrint(inc)
}
