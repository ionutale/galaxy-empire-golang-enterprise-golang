package main

import "time"

type Expedition struct {
	ID              int
	PlayerID        int
	FleetID         int
	Galaxy          int
	System          int
	Position        int
	Status          string
	ShipsSent       map[string]int
	ShipsLost       map[string]int
	ShipsFound      map[string]int
	ResourcesFound  map[string]int
	DarkMatterFound int
	Outcome         string
	StartedAt       time.Time
	TravelDuration  int
	ExploreDuration int
	CompletedAt     *time.Time
}

type ExpeditionResponse struct {
	ID              int            `json:"id"`
	PlayerID        int            `json:"player_id"`
	Status          string         `json:"status"`
	ShipsSent       map[string]int `json:"ships_sent"`
	ShipsLost       map[string]int `json:"ships_lost,omitempty"`
	ShipsFound      map[string]int `json:"ships_found,omitempty"`
	ResourcesFound  map[string]int `json:"resources_found,omitempty"`
	DarkMatterFound int            `json:"dark_matter_found,omitempty"`
	Outcome         string         `json:"outcome,omitempty"`
	StartedAt       time.Time      `json:"started_at"`
	TravelDuration  int            `json:"travel_duration"`
	ExploreDuration int            `json:"explore_duration"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

type StartExpeditionRequest struct {
	PlanetID int            `json:"planet_id"`
	Ships    map[string]int `json:"ships"`
}

type ExpeditionOutcome struct {
	Outcome        string         `json:"outcome"`
	ResourcesFound map[string]int `json:"resources_found"`
	ShipsFound     map[string]int `json:"ships_found"`
	ShipsLost      map[string]int `json:"ships_lost"`
	DarkMatter     int            `json:"dark_matter"`
	EnemyFleet     map[string]int `json:"enemy_fleet,omitempty"`
}

type ShipNebulaConfig struct {
	Attack int
}

var NebulaShipStats = map[string]ShipNebulaConfig{
	"light_fighter":   {Attack: 50},
	"heavy_fighter":   {Attack: 150},
	"cruiser":         {Attack: 400},
	"battleship":      {Attack: 1000},
	"dreadnought":     {Attack: 4000},
	"bomber":          {Attack: 1000},
	"cargo":           {Attack: 3},
	"large_cargo":     {Attack: 5},
	"recycler":        {Attack: 1},
	"espionage_probe": {Attack: 1},
	"colony_ship":     {Attack: 15},
	"solar_satellite": {Attack: 1},
}

type PlayerDarkMatter struct {
	PlayerID    int `json:"player_id"`
	Balance     int `json:"balance"`
	TotalEarned int `json:"total_earned"`
}

type DMTransaction struct {
	ID           int       `json:"id"`
	PlayerID     int       `json:"player_id"`
	Amount       int       `json:"amount"`
	BalanceAfter int       `json:"balance_after"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

type PlayerCredits struct {
	PlayerID    int `json:"player_id"`
	Balance     int `json:"balance"`
	TotalEarned int `json:"total_earned"`
}

type CreditsTransaction struct {
	ID           int       `json:"id"`
	PlayerID     int       `json:"player_id"`
	Amount       int       `json:"amount"`
	BalanceAfter int       `json:"balance_after"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

type CommanderConfig struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DMCost      int    `json:"dm_cost"`
	Duration    int    `json:"duration_days"`
}

type CommanderEntry struct {
	ID            int       `json:"id"`
	PlayerID      int       `json:"player_id"`
	CommanderType string    `json:"commander_type"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Level         int       `json:"level"`
	HiredAt       time.Time `json:"hired_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	DaysRemaining int       `json:"days_remaining,omitempty"`
}

var Commanders = []CommanderConfig{
	{Type: "commander", Name: "Commander", Description: "Reduces build time, adds extra queues", DMCost: 25, Duration: 7},
	{Type: "engineer", Name: "Engineer", Description: "Increases fusion output, retains defenses after combat", DMCost: 50, Duration: 30},
	{Type: "technocrat", Name: "Technocrat", Description: "Boosts espionage, reduces research time", DMCost: 50, Duration: 30},
	{Type: "admiral", Name: "Admiral", Description: "Increases fleet slots, reduces defense cost", DMCost: 100, Duration: 30},
	{Type: "guider", Name: "Guider", Description: "Retains expedition fleet on pirate encounter", DMCost: 50, Duration: 30},
	{Type: "geologist", Name: "Geologist", Description: "Increases mine output", DMCost: 50, Duration: 30},
}

type DailyGiftStatus struct {
	StreakDay       int    `json:"streak_day"`
	ConsecutiveDays int    `json:"consecutive_days"`
	CanClaim        bool   `json:"can_claim"`
	GiftPreview     string `json:"gift_preview"`
}

type DailyGiftResult struct {
	StreakDay       int            `json:"streak_day"`
	ConsecutiveDays int            `json:"consecutive_days"`
	Rewards         map[string]int `json:"rewards"`
}

type DailyTask struct {
	ID             int            `json:"id"`
	PlayerID       int            `json:"player_id"`
	TaskType       string         `json:"task_type"`
	Description    string         `json:"description"`
	TargetAmount   int            `json:"target_amount"`
	Progress       int            `json:"progress"`
	RewardDM       int            `json:"reward_dm"`
	RewardResources map[string]int `json:"reward_resources"`
	Completed      bool           `json:"completed"`
	Claimed        bool           `json:"claimed"`
	AssignedDate   string         `json:"assigned_date"`
}

var dailyGiftRewards = []map[string]int{
	{"dm": 1, "metal": 500},
	{"dm": 2, "crystal": 1000},
	{"dm": 3, "gas": 2000},
	{"dm": 5, "metal": 5000},
	{"dm": 8, "crystal": 10000},
	{"dm": 12, "gas": 20000},
	{"dm": 15, "metal": 50000, "crystal": 25000, "gas": 10000, "cargo": 5},
}

var dailyGiftDescriptions = []string{
	"1 DM + 500 metal",
	"2 DM + 1000 crystal",
	"3 DM + 2000 gas",
	"5 DM + 5000 metal",
	"8 DM + 10000 crystal",
	"12 DM + 20000 gas",
	"15 DM + 50000 metal + 25000 crystal + 10000 gas + 5 cargo ships",
}

type TaskDefinition struct {
	Type        string
	Description string
	Target      int
	RewardDM    int
	RewardResources map[string]int
}

var dailyTaskPool = []TaskDefinition{
	{Type: "gather_resources", Description: "Gather 10000 metal", Target: 10000, RewardDM: 3, RewardResources: map[string]int{}},
	{Type: "build_ships", Description: "Build 5 ships", Target: 5, RewardDM: 3, RewardResources: map[string]int{}},
	{Type: "research", Description: "Research 1 level", Target: 1, RewardDM: 3, RewardResources: map[string]int{}},
	{Type: "expedition", Description: "Send 1 expedition", Target: 1, RewardDM: 5, RewardResources: map[string]int{}},
	{Type: "attack", Description: "Launch 1 attack", Target: 1, RewardDM: 5, RewardResources: map[string]int{}},
	{Type: "mine_resources", Description: "Have 20000 mine production", Target: 20000, RewardDM: 2, RewardResources: map[string]int{}},
}

type GalactoniteDiscoverer struct {
	PlayerID int `json:"player_id"`
	Level    int `json:"level"`
}

type dailyGiftRewardRow struct {
	StreakDay       int
	ConsecutiveDays int
	LastClaimDate   string
}

type dailyGiftClaimRow struct {
	StreakDay       int
	ConsecutiveDays int
}

type StoreItem struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Subtype     string         `json:"subtype,omitempty"`
	Count       int            `json:"count"`
	Cost        int            `json:"cost"`
	Currency    string         `json:"currency"`
	Description string         `json:"description,omitempty"`
}

type StorePurchase struct {
	ID          int       `json:"id"`
	PlayerID    int       `json:"player_id"`
	ItemID      string    `json:"item_id"`
	Cost        int       `json:"cost"`
	Currency    string    `json:"currency"`
	PurchasedAt time.Time `json:"purchased_at"`
}

var StoreItems = []StoreItem{
	{ID: "shard_pack_small", Name: "Shard Pack (5)", Type: "shards", Subtype: "flaming_crystal", Count: 5, Cost: 10, Currency: "dm", Description: "5 Flaming Crystal shards"},
	{ID: "shard_pack_medium", Name: "Shard Pack (20)", Type: "shards", Subtype: "concentrated_galactonite", Count: 20, Cost: 35, Currency: "dm", Description: "20 Concentrated Galactonite shards"},
	{ID: "commander_hire", Name: "Commander Extension", Type: "commander_extension", Count: 30, Cost: 50, Currency: "credits", Description: "+30 days commander duration"},
	{ID: "planet_shifter", Name: "Planet Shifter", Type: "planet_shifter", Count: 1, Cost: 100, Currency: "credits", Description: "Relocate a planet"},
	{ID: "resource_pack", Name: "Resource Pack", Type: "resources", Count: 50000, Cost: 25, Currency: "dm", Description: "50k of each resource"},
	{ID: "speed_up_3h", Name: "3h Speed-Up", Type: "speed_up", Count: 180, Cost: 12, Currency: "dm", Description: "Speed up by 3 hours"},
}

type dailyTaskRow struct {
	ID             int
	PlayerID       int
	TaskType       string
	Description    string
	TargetAmount   int
	Progress       int
	RewardDM       int
	RewardResources string
	Completed      bool
	Claimed        bool
	AssignedDate   string
	RerollsUsed    int
}
