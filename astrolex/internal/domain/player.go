package domain

type Player struct {
    Credits        int64          `json:"credits"`
    Reputation     Reputation     `json:"reputation"`
    UnlockedParts  []string       `json:"unlocked_parts"`
    UnlockedBases  []string       `json:"unlocked_bases"`
    Notoriety      map[string]int `json:"notoriety"`
}

type Reputation struct {
    Safety  int            `json:"safety"`
    Speed   int            `json:"speed"`
    Politic map[string]int `json:"politic"`
}
