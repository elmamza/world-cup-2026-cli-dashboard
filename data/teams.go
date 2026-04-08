package data

var teamInfoByCode map[string]TeamInfo = make(map[string]TeamInfo)

var TeamInfoByCode = teamInfoByCode

func SetTeamInfoByCode(m map[string]TeamInfo) {
	teamInfoByCode = m
	TeamInfoByCode = teamInfoByCode
}
