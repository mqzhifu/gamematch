package gamematch


type HttpReqSign struct {
	MatchCode	string	`json:"matchCode"`
	GroupId		int		`json:"groupId"`
	CustomProp	string	`json:"customProp"`
	PlayerList	[]HttpReqPlayer	`json:"playerList"`
	Addition	string	`json:"addition"`
	RuleId 		int 	`json:"ruleId"`
}

type HttpReqPlayer struct {
	Uid 		int		`json:"uid"`
	MatchAttr	HttpReqPlayerAttr	`json:"matchAttr"`
}

type HttpReqPlayerAttr struct {

}

type HttpReqSignCancel struct {
	PlayerId	int
	GroupId 	int
	RuleId 		int
	MatchCode	string
}