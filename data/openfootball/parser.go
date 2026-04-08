package openfootball

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Team struct {
	Code  string
	Name  string
	Group string
}

type Group struct {
	Letter string
	Teams  []Team
}

type Match struct {
	ID             int
	HomeTeamCode   string
	AwayTeamCode   string
	Date           time.Time
	Venue          string
	HomeTeamScore  uint64
	AwayTeamScore  uint64
	WinnerTeamCode string
	Minute         string
	Status         string
	Stage          string
	GroupLetter    string
	Matchday       int
}

type Parser struct {
	groups         map[string]*Group
	teams          map[string]Team
	matches        []Match
	matchID        int
	tournamentYear int
	currentGroup   string
	currentDate    time.Time
}

func NewParser(year int) *Parser {
	return &Parser{
		groups:         make(map[string]*Group),
		teams:          make(map[string]Team),
		matches:        make([]Match, 0),
		matchID:        1,
		tournamentYear: year,
	}
}

func (p *Parser) Parse(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	p.currentDate = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	p.currentGroup = ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "Group ") && !strings.HasPrefix(line, "▪ Group") {
			if err := p.parseGroupLine(line); err != nil {
				return fmt.Errorf("parse group line: %w", err)
			}
		} else if strings.HasPrefix(line, "▪ Group") {
			groupRe := regexp.MustCompile(`▪\s+Group\s+([A-L])`)
			if m := groupRe.FindStringSubmatch(line); len(m) > 1 {
				p.currentGroup = m[1]
			}
		} else if isDateLine(line) {
			if newDate := parseDateLine(line, p.tournamentYear); !newDate.IsZero() {
				p.currentDate = newDate
			}
		} else if isMatchLine(line) {
			match, err := p.parseMatchLine(line)
			if err != nil {
				return fmt.Errorf("parse match line: %w", err)
			}
			if match != nil {
				match.Date = match.Date.In(time.UTC)
				p.matches = append(p.matches, *match)
			}
		}
	}

	return nil
}

func isDateLine(line string) bool {
	datePattern := regexp.MustCompile(`^[A-Z][a-z]{2}\s+June\s+\d{1,2}$`)
	return datePattern.MatchString(line)
}

func parseDateLine(line string, year int) time.Time {
	dateRe := regexp.MustCompile(`([A-Z][a-z]{2})\s+June\s+(\d{1,2})`)
	matches := dateRe.FindStringSubmatch(line)
	if len(matches) < 3 {
		return time.Time{}
	}

	day, err := strconv.Atoi(matches[2])
	if err != nil {
		return time.Time{}
	}

	return time.Date(year, time.June, day, 0, 0, 0, 0, time.UTC)
}

func (p *Parser) parseGroupLine(line string) error {
	re := regexp.MustCompile(`Group\s+([A-L])\s*\|\s*(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 3 {
		return nil
	}

	letter := matches[1]
	teamNamesStr := matches[2]

	teamNames := extractTeamNames(teamNamesStr)
	if len(teamNames) == 0 {
		return nil
	}

	group := &Group{
		Letter: letter,
		Teams:  make([]Team, 0),
	}

	for _, name := range teamNames {
		code := countryToCode(name)
		team := Team{
			Code:  code,
			Name:  name,
			Group: letter,
		}
		group.Teams = append(group.Teams, team)
		p.teams[code] = team
	}

	p.groups[letter] = group
	return nil
}

func extractTeamNames(s string) []string {
	s = strings.ReplaceAll(s, "\t", " ")
	for {
		old := s
		s = strings.ReplaceAll(s, "  ", " ")
		if s == old {
			break
		}
	}
	s = strings.TrimSpace(s)

	var result []string
	words := strings.Split(s, " ")
	current := ""

	for _, word := range words {
		current += " " + word
		current = strings.TrimSpace(current)

		if _, ok := countryCodeMap[current]; ok {
			result = append(result, current)
			current = ""
			continue
		}

		found := false
		for i := len(current); i >= 2; i-- {
			partial := current[:i]
			if _, ok := countryCodeMap[partial]; ok {
				result = append(result, partial)
				remaining := strings.TrimSpace(current[i:])
				current = remaining
				found = true
				break
			}
		}
		if !found && len(current) > 20 {
			result = append(result, current)
			current = ""
		}
	}

	if len(result) == 0 {
		result = strings.Fields(s)
	}

	return result
}

func isMatchLine(line string) bool {
	timePattern := regexp.MustCompile(`^\s*\d{1,2}:\d{2}\s+UTC`)
	return timePattern.MatchString(line)
}

func (p *Parser) parseMatchLine(line string) (*Match, error) {
	timeRe := regexp.MustCompile(`(\d{1,2}:\d{2})\s+(UTC[+-]\d+)\s+(.+?)\s+v\s+(.+?)\s+@(.+)`)
	matches := timeRe.FindStringSubmatch(line)

	if len(matches) < 6 {
		return nil, nil
	}

	timeStr := strings.TrimSpace(matches[1])
	tz := strings.TrimSpace(matches[2])
	homeTeamName := strings.TrimSpace(matches[3])
	awayTeamName := strings.TrimSpace(matches[4])
	venue := strings.TrimSpace(matches[5])

	homeTeamCode := countryToCode(homeTeamName)
	awayTeamCode := countryToCode(awayTeamName)

	groupLetter := ""
	for gLetter, group := range p.groups {
		for _, team := range group.Teams {
			if team.Code == homeTeamCode || team.Code == awayTeamCode {
				groupLetter = gLetter
				break
			}
		}
		if groupLetter != "" {
			break
		}
	}
	if groupLetter == "" {
		groupLetter = p.currentGroup
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	hourMin := strings.Split(timeStr, ":")
	hour, _ := strconv.Atoi(hourMin[0])
	min, _ := strconv.Atoi(hourMin[1])

	baseDate := p.currentDate
	dateWithTime := baseDate.Add(time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute)
	dateWithTime = dateWithTime.In(loc)

	matchday := p.getMatchdayForGroup(groupLetter, dateWithTime)

	return &Match{
		ID:           p.matchID,
		HomeTeamCode: homeTeamCode,
		AwayTeamCode: awayTeamCode,
		Date:         dateWithTime,
		Venue:        venue,
		Status:       "Scheduled",
		Stage:        "Group",
		GroupLetter:  groupLetter,
		Matchday:     matchday,
	}, nil
}

func (p *Parser) getGroupStartDate(group string) time.Time {
	dates := map[string]time.Time{
		"A": time.Date(2026, time.June, 11, 0, 0, 0, 0, time.UTC),
		"B": time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC),
		"C": time.Date(2026, time.June, 13, 0, 0, 0, 0, time.UTC),
		"D": time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC),
		"E": time.Date(2026, time.June, 14, 0, 0, 0, 0, time.UTC),
		"F": time.Date(2026, time.June, 14, 0, 0, 0, 0, time.UTC),
		"G": time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		"H": time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		"I": time.Date(2026, time.June, 16, 0, 0, 0, 0, time.UTC),
		"J": time.Date(2026, time.June, 16, 0, 0, 0, 0, time.UTC),
		"K": time.Date(2026, time.June, 17, 0, 0, 0, 0, time.UTC),
		"L": time.Date(2026, time.June, 17, 0, 0, 0, 0, time.UTC),
	}
	if d, ok := dates[group]; ok {
		return d
	}
	return time.Date(2026, time.June, 11, 0, 0, 0, 0, time.UTC)
}

func (p *Parser) getMatchdayForGroup(group string, matchDate time.Time) int {
	groupStartDate := p.getGroupStartDate(group)
	daysDiff := int(matchDate.Sub(groupStartDate).Hours() / 24)
	return daysDiff/7 + 1
}

func (p *Parser) GetTeams() map[string]Team {
	return p.teams
}

func (p *Parser) GetGroups() map[string]*Group {
	return p.groups
}

func (p *Parser) GetMatches() []Match {
	sorted := make([]Match, len(p.matches))
	copy(sorted, p.matches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})
	return sorted
}

func Parse(data []byte, year int) (*Parser, error) {
	p := NewParser(year)
	if err := p.Parse(data); err != nil {
		return nil, err
	}
	return p, nil
}

var countryCodeMap = map[string]string{
	"Mexico":               "MEX",
	"South Africa":         "RSA",
	"South Korea":          "KOR",
	"Czech Republic":       "CZE",
	"Canada":               "CAN",
	"Bosnia & Herzegovina": "BIH",
	"Qatar":                "QAT",
	"Switzerland":          "SUI",
	"Brazil":               "BRA",
	"Morocco":              "MAR",
	"Haiti":                "HAI",
	"Scotland":             "SCO",
	"USA":                  "USA",
	"Paraguay":             "PAR",
	"Australia":            "AUS",
	"Turkey":               "TUR",
	"Germany":              "GER",
	"Curaçao":              "CUW",
	"Ivory Coast":          "CIV",
	"Ecuador":              "ECU",
	"Netherlands":          "NED",
	"Japan":                "JPN",
	"Sweden":               "SWE",
	"Tunisia":              "TUN",
	"Belgium":              "BEL",
	"Egypt":                "EGY",
	"Iran":                 "IRN",
	"New Zealand":          "NZL",
	"Spain":                "ESP",
	"Cape Verde":           "CPV",
	"Saudi Arabia":         "KSA",
	"Uruguay":              "URU",
	"France":               "FRA",
	"Senegal":              "SEN",
	"Iraq":                 "IRQ",
	"Norway":               "NOR",
	"Argentina":            "ARG",
	"Algeria":              "ALG",
	"Austria":              "AUT",
	"Jordan":               "JOR",
	"Portugal":             "POR",
	"DR Congo":             "COD",
	"Uzbekistan":           "UZB",
	"Colombia":             "COL",
	"England":              "ENG",
	"Croatia":              "CRO",
	"Ghana":                "GHA",
	"Panama":               "PAN",
}

func countryToCode(name string) string {
	name = strings.TrimSpace(name)
	if code, ok := countryCodeMap[name]; ok {
		return code
	}
	words := strings.Split(name, " ")
	if len(words) > 1 {
		for i := len(words); i > 0; i-- {
			partial := strings.Join(words[:i], " ")
			if code, ok := countryCodeMap[partial]; ok {
				return code
			}
		}
		lastWord := words[len(words)-1]
		if len(lastWord) >= 3 {
			return strings.ToUpper(lastWord[:3])
		}
		return strings.ToUpper(lastWord)
	}
	if len(name) >= 3 {
		return strings.ToUpper(name[:3])
	}
	return strings.ToUpper(name)
}

type GroupTableTeam struct {
	Code              string `json:"Code"`
	Points            int    `json:"Points"`
	Wins              int    `json:"Wins"`
	Draws             int    `json:"Draws"`
	Losses            int    `json:"Losses"`
	MatchesPlayed     int    `json:"MatchesPlayed"`
	GoalsFor          int    `json:"GoalsFor"`
	GoalsAgainst      int    `json:"GoalsAgainst"`
	GoalsDifferential int    `json:"GoalsDifferential"`
}

type GroupTable struct {
	Letter string           `json:"Letter"`
	Table  []GroupTableTeam `json:"Table"`
}

func ParseGroupStandings(matches []Match) map[string][]GroupTableTeam {
	standings := make(map[string][]GroupTableTeam)
	groupTeams := make(map[string]map[string]*GroupTableTeam)

	for _, m := range matches {
		if m.Stage != "Group" {
			continue
		}
		group := m.GroupLetter
		if group == "" {
			continue
		}

		if _, ok := groupTeams[group]; !ok {
			groupTeams[group] = make(map[string]*GroupTableTeam)
		}

		if _, ok := groupTeams[group][m.HomeTeamCode]; !ok {
			groupTeams[group][m.HomeTeamCode] = &GroupTableTeam{Code: m.HomeTeamCode}
		}
		if _, ok := groupTeams[group][m.AwayTeamCode]; !ok {
			groupTeams[group][m.AwayTeamCode] = &GroupTableTeam{Code: m.AwayTeamCode}
		}

		homeTeam := groupTeams[group][m.HomeTeamCode]
		awayTeam := groupTeams[group][m.AwayTeamCode]

		homeTeam.MatchesPlayed++
		awayTeam.MatchesPlayed++

		homeTeam.GoalsFor += int(m.HomeTeamScore)
		homeTeam.GoalsAgainst += int(m.AwayTeamScore)
		awayTeam.GoalsFor += int(m.AwayTeamScore)
		awayTeam.GoalsAgainst += int(m.HomeTeamScore)

		if m.HomeTeamScore > m.AwayTeamScore {
			homeTeam.Wins++
			homeTeam.Points += 3
			awayTeam.Losses++
		} else if m.AwayTeamScore > m.HomeTeamScore {
			awayTeam.Wins++
			awayTeam.Points += 3
			homeTeam.Losses++
		} else {
			homeTeam.Draws++
			awayTeam.Draws++
			homeTeam.Points++
			awayTeam.Points++
		}
	}

	for group, teams := range groupTeams {
		standings[group] = make([]GroupTableTeam, 0, len(teams))
		for _, team := range teams {
			team.GoalsDifferential = team.GoalsFor - team.GoalsAgainst
			standings[group] = append(standings[group], *team)
		}
		sortStandings(standings[group])
	}

	return standings
}

func sortStandings(teams []GroupTableTeam) {
	for i := 0; i < len(teams)-1; i++ {
		for j := i + 1; j < len(teams); j++ {
			if shouldSwap(teams[i], teams[j]) {
				teams[i], teams[j] = teams[j], teams[i]
			}
		}
	}
}

func shouldSwap(a, b GroupTableTeam) bool {
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	if a.GoalsDifferential != b.GoalsDifferential {
		return a.GoalsDifferential > b.GoalsDifferential
	}
	return a.GoalsFor > b.GoalsFor
}

func ParseReader(r io.Reader, year int) (*Parser, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse(data, year)
}
