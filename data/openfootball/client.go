package openfootball

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cedricblondeau/world-cup-2022-cli-dashboard/data"
)

type Client struct {
	httpClient  *http.Client
	tournament  string
	cache       *cache
	teams       map[string]TeamInfo
	groupTables []data.GroupTable
	matches     []data.Match
	mu          sync.RWMutex
}

type cache struct {
	data      []byte
	parsedAt  time.Time
	groups    map[string]*Group
	teams     map[string]TeamInfo
	matches   []Match
	teamMatch []data.Match
	standings map[string][]GroupTableTeam
	mu        sync.RWMutex
}

func NewClient(tournament string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		tournament: tournament,
		cache: &cache{
			groups:  make(map[string]*Group),
			teams:   make(map[string]TeamInfo),
			matches: make([]Match, 0),
		},
		teams: make(map[string]TeamInfo),
	}
}

func (c *Client) Name() string {
	return "openfootball"
}

func (c *Client) fetchData() ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/openfootball/worldcup/master/%s/cup.txt", c.tournament)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) parseAndCache(dataBytes []byte) error {
	parser, err := Parse(dataBytes, 2026)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	c.cache.data = dataBytes
	c.cache.parsedAt = time.Now()
	c.cache.groups = parser.groups

	for code, team := range parser.teams {
		c.cache.teams[code] = TeamInfo{
			Name:        team.Name,
			Group:       team.Group,
			FirstColor:  defaultTeamColors[code].FirstColor,
			SecondColor: defaultTeamColors[code].SecondColor,
		}
	}

	c.cache.matches = parser.GetMatches()
	c.cache.standings = ParseGroupStandings(parser.GetMatches())

	return nil
}

func (c *Client) ensureLoaded() error {
	c.cache.mu.RLock()
	if c.cache.data != nil && time.Since(c.cache.parsedAt) < 5*time.Minute {
		c.cache.mu.RUnlock()
		return nil
	}
	c.cache.mu.RUnlock()

	dataBytes, err := c.fetchData()
	if err != nil {
		return err
	}

	return c.parseAndCache(dataBytes)
}

func (c *Client) GroupTables() ([]data.GroupTable, error) {
	if err := c.ensureLoaded(); err != nil {
		return nil, err
	}

	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	var tables []data.GroupTable
	for letter, teams := range c.cache.standings {
		table := data.GroupTable{
			Letter: letter,
			Table:  make([]data.GroupTableTeam, len(teams)),
		}
		for i, t := range teams {
			table.Table[i] = data.GroupTableTeam{
				Code:              t.Code,
				Points:            t.Points,
				Wins:              t.Wins,
				Draws:             t.Draws,
				Losses:            t.Losses,
				MatchesPlayed:     t.MatchesPlayed,
				GoalsFor:          t.GoalsFor,
				GoalsAgainst:      t.GoalsAgainst,
				GoalsDifferential: t.GoalsDifferential,
			}
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func (c *Client) SortedMatches() ([]data.Match, error) {
	if err := c.ensureLoaded(); err != nil {
		return nil, err
	}

	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	matches := make([]data.Match, 0, len(c.cache.matches))
	for _, m := range c.cache.matches {
		matches = append(matches, data.Match{
			ID:           m.ID,
			HomeTeamCode: m.HomeTeamCode,
			AwayTeamCode: m.AwayTeamCode,
			Date:         m.Date,
			Venue:        m.Venue,
			Status:       data.Status(m.Status),
			Stage:        m.Stage,
		})
	}

	return matches, nil
}

func (c *Client) GetTeams() map[string]TeamInfo {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	result := make(map[string]TeamInfo)
	for k, v := range c.cache.teams {
		result[k] = v
	}
	return result
}

var defaultTeamColors = map[string]struct {
	Name        string
	Group       string
	FirstColor  string
	SecondColor string
}{
	"MEX": {"Mexico", "A", "#006847", "#CE1126"},
	"RSA": {"South Africa", "A", "#007749", "#DEB887"},
	"KOR": {"South Korea", "A", "#CD2E3A", "#0047A5"},
	"CZE": {"Czech Republic", "A", "#D7141A", "#11457E"},
	"CAN": {"Canada", "B", "#FF0000", "#FFFFFF"},
	"BIH": {"Bosnia & Herzegovina", "B", "#002395", "#FECE00"},
	"QAT": {"Qatar", "B", "#8A1538", "#FFFFFF"},
	"SUI": {"Switzerland", "B", "#FF0000", "#FFFFFF"},
	"BRA": {"Brazil", "C", "#FFDF00", "#009C3B"},
	"MAR": {"Morocco", "C", "#C1272D", "#000000"},
	"HAI": {"Haiti", "C", "#00209F", "#D21034"},
	"SCO": {"Scotland", "C", "#005EB8", "#FFFFFF"},
	"USA": {"USA", "D", "#BF0D3E", "#FFFFFF"},
	"PAR": {"Paraguay", "D", "#D52B1E", "#FFFFFF"},
	"AUS": {"Australia", "D", "#00843D", "#FFC220"},
	"TUR": {"Turkey", "D", "#E30A17", "#FFFFFF"},
	"GER": {"Germany", "E", "#000000", "#DD0000"},
	"CUW": {"Curaçao", "E", "#002395", "#FFFFFF"},
	"CIV": {"Ivory Coast", "E", "#FF8200", "#009450"},
	"ECU": {"Ecuador", "E", "#FFD100", "#EF3340"},
	"NED": {"Netherlands", "F", "#F68A00", "#FFFFFF"},
	"JPN": {"Japan", "F", "#BC002D", "#FFFFFF"},
	"SWE": {"Sweden", "F", "#FECC02", "#006AA7"},
	"TUN": {"Tunisia", "F", "#E70013", "#FFFFFF"},
	"BEL": {"Belgium", "G", "#FF0000", "#FAE042"},
	"EGY": {"Egypt", "G", "#CE1124", "#FFFFFF"},
	"IRN": {"Iran", "G", "#DA0000", "#FFFFFF"},
	"NZL": {"New Zealand", "G", "#000000", "#FFFFFF"},
	"ESP": {"Spain", "H", "#AA151B", "#F1BF00"},
	"CPV": {"Cape Verde", "H", "#002395", "#FFFFFF"},
	"KSA": {"Saudi Arabia", "H", "#006C35", "#FFFFFF"},
	"URU": {"Uruguay", "H", "#FFFFFF", "#000000"},
	"FRA": {"France", "I", "#0055A4", "#FFFFFF"},
	"SEN": {"Senegal", "I", "#00853F", "#FDEF42"},
	"IRQ": {"Iraq", "I", "#007A3D", "#FFFFFF"},
	"NOR": {"Norway", "I", "#BA0C2F", "#FFFFFF"},
	"ARG": {"Argentina", "J", "#75AADB", "#FFFFFF"},
	"ALG": {"Algeria", "J", "#007A33", "#FFFFFF"},
	"AUT": {"Austria", "J", "#ED2939", "#FFFFFF"},
	"JOR": {"Jordan", "J", "#CE1126", "#FFFFFF"},
	"POR": {"Portugal", "K", "#006600", "#FFFFFF"},
	"COD": {"DR Congo", "K", "#FCDC04", "#007A3D"},
	"UZB": {"Uzbekistan", "K", "#00A0DC", "#FFF700"},
	"COL": {"Colombia", "K", "#FCD116", "#003893"},
	"ENG": {"England", "L", "#FFFFFF", "#CE1124"},
	"CRO": {"Croatia", "L", "#FF0000", "#FFFFFF"},
	"GHA": {"Ghana", "L", "#FCD116", "#000000"},
	"PAN": {"Panama", "L", "#D21034", "#FFFFFF"},
}

func (c *Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Tournament string    `json:"tournament"`
		CachedAt   time.Time `json:"cached_at"`
		Teams      int       `json:"teams"`
		Matches    int       `json:"matches"`
	}{
		Tournament: c.tournament,
		CachedAt:   c.cache.parsedAt,
		Teams:      len(c.cache.teams),
		Matches:    len(c.cache.matches),
	})
}

func ParseLocalData(dataBytes []byte) (*Parser, map[string]TeamInfo, error) {
	parser, err := Parse(dataBytes, 2026)
	if err != nil {
		return nil, nil, err
	}

	teams := make(map[string]TeamInfo)
	for code, team := range parser.teams {
		teams[code] = TeamInfo{
			Name:        team.Name,
			Group:       team.Group,
			FirstColor:  defaultTeamColors[code].FirstColor,
			SecondColor: defaultTeamColors[code].SecondColor,
		}
	}

	return parser, teams, nil
}

func GenerateGroupStageMatches(groups map[string]*Group) []Match {
	var matches []Match
	matchID := 1

	for _, group := range groups {
		teams := group.Teams
		if len(teams) != 4 {
			continue
		}

		matchups := []struct{ home, away int }{
			{0, 1}, {2, 3},
			{0, 2}, {1, 3},
			{0, 3}, {1, 2},
		}

		matchday := 1
		for round := 0; round < 3; round++ {
			for _, m := range matchups[round*2:] {
				match := Match{
					ID:           matchID,
					HomeTeamCode: teams[m.home].Code,
					AwayTeamCode: teams[m.away].Code,
					GroupLetter:  group.Letter,
					Matchday:     matchday,
					Status:       "Scheduled",
					Stage:        "Group",
				}
				matches = append(matches, match)
				matchID++
			}
			matchday++
		}
	}

	return matches
}

func ConvertToDataMatch(matches []Match) []data.Match {
	result := make([]data.Match, len(matches))
	for i, m := range matches {
		result[i] = data.Match{
			ID:             m.ID,
			HomeTeamCode:   m.HomeTeamCode,
			AwayTeamCode:   m.AwayTeamCode,
			Date:           m.Date,
			Venue:          m.Venue,
			HomeTeamScore:  m.HomeTeamScore,
			AwayTeamScore:  m.AwayTeamScore,
			WinnerTeamCode: m.WinnerTeamCode,
			Minute:         m.Minute,
			Status:         data.Status(m.Status),
			Stage:          m.Stage,
		}
	}
	return result
}

func ConvertToGroupTables(standings map[string][]GroupTableTeam) []data.GroupTable {
	var tables []data.GroupTable
	for letter, teams := range standings {
		table := data.GroupTable{
			Letter: letter,
			Table:  make([]data.GroupTableTeam, len(teams)),
		}
		for i, t := range teams {
			table.Table[i] = data.GroupTableTeam{
				Code:              t.Code,
				Points:            t.Points,
				Wins:              t.Wins,
				Draws:             t.Draws,
				Losses:            t.Losses,
				MatchesPlayed:     t.MatchesPlayed,
				GoalsFor:          t.GoalsFor,
				GoalsAgainst:      t.GoalsAgainst,
				GoalsDifferential: t.GoalsDifferential,
			}
		}
		tables = append(tables, table)
	}
	return tables
}

func (c *Client) Refresh() error {
	dataBytes, err := c.fetchData()
	if err != nil {
		return err
	}
	return c.parseAndCache(dataBytes)
}

func (c *Client) CacheStats() (time.Time, int, int) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()
	return c.cache.parsedAt, len(c.cache.teams), len(c.cache.matches)
}

type TeamInfo struct {
	Name        string
	Group       string
	FirstColor  string
	SecondColor string
}

func init() {
	m := make(map[string]data.TeamInfo)
	for code, tc := range defaultTeamColors {
		m[code] = data.TeamInfo{
			Name:        tc.Name,
			Group:       tc.Group,
			FirstColor:  tc.FirstColor,
			SecondColor: tc.SecondColor,
		}
	}
	data.SetTeamInfoByCode(m)
}
