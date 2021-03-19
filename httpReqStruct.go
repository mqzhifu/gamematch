package gamematch


type HttpReqSign struct {
	MatchCode	string	`json:"matchCode"`
	GroupId		int		`json:"groupId"`
	CustomProp	string	`json:"customProp"`
	PlayerList	[]HttpReqPlayer	`json:"playerList"`
	Addition	string	`json:"addition"`
	RuleId 		int 	`json:"ruleId"`
	Rule_ver	int		`json:"rule_ver"`
}

type HttpReqPlayer struct {
	Uid 		int		`json:"uid"`
	MatchAttr	map[string]int	`json:"matchAttr"`
}

//type HttpReqPlayerAttr struct {
//
//}

type HttpReqSignCancel struct {
	PlayerId	int
	GroupId 	int
	RuleId 		int
	MatchCode	string
}