package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type NebulaService struct {
	repo          Repository
	planetBaseURL string
	httpClient    *http.Client
}

func NewNebulaService(repo Repository, planetBaseURL string) *NebulaService {
	return &NebulaService{
		repo:          repo,
		planetBaseURL: planetBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *NebulaService) StartExpedition(ctx context.Context, playerID, planetID int, ships map[string]int) (Expedition, error) {
	if len(ships) == 0 {
		return Expedition{}, fmt.Errorf("no ships selected")
	}

	for shipType := range ships {
		if _, ok := validShipTypes[shipType]; !ok {
			return Expedition{}, fmt.Errorf("unknown ship type: %s", shipType)
		}
	}

	planetInfo, err := s.getPlanetInfo(ctx, planetID)
	if err != nil {
		return Expedition{}, fmt.Errorf("planet service: %w", err)
	}

	if planetInfo.PlayerID != playerID {
		return Expedition{}, fmt.Errorf("planet does not belong to player")
	}

	// Check expedition cooldown
	lastExpedition, err := s.repo.GetLastExpeditionTime(ctx, playerID)
	if err == nil && time.Since(lastExpedition) < 60*time.Second {
		return Expedition{}, fmt.Errorf("expedition cooldown: please wait before starting another")
	}

	for shipType, qty := range ships {
		if planetInfo.Ships[shipType] < qty {
			return Expedition{}, fmt.Errorf("insufficient %s: have %d, need %d", shipType, planetInfo.Ships[shipType], qty)
		}
	}

	if err := s.deductShips(ctx, planetID, ships); err != nil {
		return Expedition{}, fmt.Errorf("deduct ships: %w", err)
	}

	espionageTechLevel := s.getEspionageTechLevel(ctx, planetInfo.PlayerID)

	outcome := generateOutcome(ships, espionageTechLevel)

	if len(outcome.ResourcesFound) > 0 {
		for resource, amount := range outcome.ResourcesFound {
			if err := s.addResourceToPlanet(ctx, planetID, resource, amount); err != nil {
				slog.Error("add resource failed", "planet", planetID, "resource", resource, "error", err)
			}
		}
	}

	if len(outcome.ShipsFound) > 0 {
		if err := s.addShipsToPlanet(ctx, planetID, outcome.ShipsFound); err != nil {
			slog.Error("add ships failed", "planet", planetID, "error", err)
		}
	}

	returnShips := make(map[string]int)
	for shipType, qty := range ships {
		lost := outcome.ShipsLost[shipType]
		returned := qty - lost
		if returned > 0 {
			returnShips[shipType] = returned
		}
	}
	if len(returnShips) > 0 {
		if err := s.addShipsToPlanet(ctx, planetID, returnShips); err != nil {
			slog.Error("return ships failed", "planet", planetID, "error", err)
		}
	}

	now := time.Now()
	exp := Expedition{
		PlayerID:        playerID,
		Status:          "completed",
		ShipsSent:       ships,
		ShipsLost:       outcome.ShipsLost,
		ShipsFound:      outcome.ShipsFound,
		ResourcesFound:  outcome.ResourcesFound,
		DarkMatterFound: outcome.DarkMatter,
		Outcome:         outcome.Outcome,
		TravelDuration:  300,
		ExploreDuration: 1800,
		CompletedAt:     &now,
	}

	created, err := s.repo.CreateExpedition(ctx, exp)
	if err != nil {
		slog.Error("save expedition failed", "error", err)
		return exp, nil
	}

	if outcome.DarkMatter > 0 {
		if err := s.addDarkMatter(ctx, playerID, outcome.DarkMatter); err != nil {
			slog.Error("add dark matter failed", "player", playerID, "amount", outcome.DarkMatter, "error", err)
		}
	}

	return created, nil
}

var validShipTypes = map[string]bool{
	"cargo": true, "large_cargo": true, "recycler": true,
	"espionage_probe": true, "colony_ship": true, "solar_satellite": true,
	"light_fighter": true, "heavy_fighter": true, "cruiser": true,
	"battleship": true, "dreadnought": true, "bomber": true,
}

func generateOutcome(ships map[string]int, espionageTechLevel int) ExpeditionOutcome {
	totalShips := 0
	for _, qty := range ships {
		totalShips += qty
	}
	if totalShips == 0 {
		return ExpeditionOutcome{Outcome: "nothing"}
	}

	resourcesProb := 35 + espionageTechLevel*2
	shipsProb := 10 + espionageTechLevel*2
	darkMatterProb := 5
	piratesProb := 10
	aliensProb := 10
	nothingProb := 30 - espionageTechLevel*1

	total := resourcesProb + shipsProb + darkMatterProb + piratesProb + aliensProb + nothingProb
	roll := rand.Intn(total)

	switch {
	case roll < resourcesProb:
		return generateResourcesOutcome(totalShips)
	case roll < resourcesProb+shipsProb:
		return generateShipsOutcome(totalShips)
	case roll < resourcesProb+shipsProb+darkMatterProb:
		return generateDarkMatterOutcome(totalShips)
	case roll < resourcesProb+shipsProb+darkMatterProb+piratesProb:
		return generatePiratesOutcome(ships)
	case roll < resourcesProb+shipsProb+darkMatterProb+piratesProb+aliensProb:
		return generateAliensOutcome(ships, totalShips)
	default:
		return ExpeditionOutcome{Outcome: "nothing"}
	}
}

func generateResourcesOutcome(totalShips int) ExpeditionOutcome {
	ratio := math.Min(1.0, float64(totalShips)/667.0)
	maxAmount := int(2000000.0 * ratio)
	if maxAmount < 10000 {
		maxAmount = 10000
	}
	metal := rand.Intn(maxAmount-10000+1) + 10000
	crystal := rand.Intn(maxAmount-10000+1) + 10000
	gas := rand.Intn(maxAmount-10000+1) + 10000
	return ExpeditionOutcome{
		Outcome: "resources",
		ResourcesFound: map[string]int{
			"metal":   metal,
			"crystal": crystal,
			"gas":     gas,
		},
	}
}

func generateShipsOutcome(totalShips int) ExpeditionOutcome {
	ratio := math.Max(1.0, float64(totalShips)/10.0)
	ships := make(map[string]int)

	possible := []struct {
		shipType string
		maxQty   int
	}{
		{"light_fighter", int(10.0 * ratio)},
		{"heavy_fighter", int(5.0 * ratio)},
		{"cruiser", int(3.0 * ratio)},
	}

	for _, p := range possible {
		qty := rand.Intn(p.maxQty + 1)
		if qty > 0 {
			ships[p.shipType] = qty
		}
	}

	if len(ships) == 0 {
		ships["light_fighter"] = 1
	}

	return ExpeditionOutcome{
		Outcome:    "ships",
		ShipsFound: ships,
	}
}

func generateDarkMatterOutcome(totalShips int) ExpeditionOutcome {
	var dm int
	switch {
	case totalShips > 100:
		dm = rand.Intn(31) + 20
	case totalShips >= 50:
		dm = rand.Intn(21) + 10
	default:
		dm = rand.Intn(11) + 5
	}
	return ExpeditionOutcome{
		Outcome:    "dark_matter",
		DarkMatter: dm,
	}
}

func generatePiratesOutcome(ships map[string]int) ExpeditionOutcome {
	lost := make(map[string]int)
	enemyFleet := make(map[string]int)
	for shipType, qty := range ships {
		lostQty := int(float64(qty) * 0.3)
		if lostQty < 1 && qty > 0 {
			lostQty = 1
		}
		if lostQty > 0 {
			lost[shipType] = lostQty
		}
		enemyQty := int(float64(qty) * 0.7)
		if enemyQty > 0 {
			enemyFleet[shipType] = enemyQty
		}
	}
	return ExpeditionOutcome{
		Outcome:    "pirates",
		ShipsLost:  lost,
		EnemyFleet: enemyFleet,
	}
}

func generateAliensOutcome(ships map[string]int, totalShips int) ExpeditionOutcome {
	outcome := generateResourcesOutcome(totalShips)
	outcome.Outcome = "aliens"
	for resource, amount := range outcome.ResourcesFound {
		outcome.ResourcesFound[resource] = int(float64(amount) * 1.3)
	}
	enemyFleet := make(map[string]int)
	for shipType, qty := range ships {
		enemyQty := int(float64(qty) * 1.3)
		if enemyQty > 0 {
			enemyFleet[shipType] = enemyQty
		}
	}
	outcome.EnemyFleet = enemyFleet
	return outcome
}

type planetInfoResponse struct {
	PlayerID int            `json:"player_id"`
	PlanetID int            `json:"planet_id"`
	Metal    int            `json:"metal"`
	Crystal  int            `json:"crystal"`
	Gas      int            `json:"gas"`
	Ships    map[string]int `json:"ships"`
}

func (s *NebulaService) getPlanetInfo(ctx context.Context, planetID int) (planetInfoResponse, error) {
	body, _ := json.Marshal(map[string]int{"planet_id": planetID})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/planet/info", "application/json", bytes.NewReader(body))
	if err != nil {
		return planetInfoResponse{}, fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return planetInfoResponse{}, fmt.Errorf("planet service: %s", string(respBody))
	}
	var info planetInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return planetInfoResponse{}, fmt.Errorf("parse planet info: %w", err)
	}
	return info, nil
}

func (s *NebulaService) deductShips(ctx context.Context, planetID int, ships map[string]int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"ships":     ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/deduct", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *NebulaService) addShipsToPlanet(ctx context.Context, planetID int, ships map[string]int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"ships":     ships,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/ships/add", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *NebulaService) addResourceToPlanet(ctx context.Context, planetID int, resource string, amount int) error {
	body, _ := json.Marshal(map[string]any{
		"planet_id": planetID,
		"resource":  resource,
		"amount":    amount,
	})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/resources/add", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planet service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planet service: %s", string(respBody))
	}
	return nil
}

func (s *NebulaService) addDarkMatter(ctx context.Context, playerID, amount int) error {
	if err := s.repo.AddDarkMatter(ctx, playerID, amount); err != nil {
		return err
	}
	balance, _, err := s.repo.GetDarkMatterBalance(ctx, playerID)
	if err != nil {
		return err
	}
	return s.repo.AddDMTransaction(ctx, playerID, amount, balance, "expedition")
}

func (s *NebulaService) AddCredits(ctx context.Context, playerID, amount int, reason string) (int, error) {
	if err := s.repo.AddCredits(ctx, playerID, amount); err != nil {
		return 0, err
	}
	balance, _, err := s.repo.GetCreditsBalance(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if err := s.repo.AddCreditsTransaction(ctx, playerID, amount, balance, reason); err != nil {
		slog.Error("log credits transaction failed", "error", err)
	}
	return balance, nil
}

func (s *NebulaService) SpendCredits(ctx context.Context, playerID, amount int, reason string) (int, error) {
	if err := s.repo.SpendCredits(ctx, playerID, amount); err != nil {
		return 0, err
	}
	balance, _, err := s.repo.GetCreditsBalance(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if err := s.repo.AddCreditsTransaction(ctx, playerID, -amount, balance, reason); err != nil {
		slog.Error("log credits transaction failed", "error", err)
	}
	return balance, nil
}

func (s *NebulaService) GetCreditsBalance(ctx context.Context, playerID int) (PlayerCredits, error) {
	balance, totalEarned, err := s.repo.GetCreditsBalance(ctx, playerID)
	if err != nil {
		return PlayerCredits{}, err
	}
	return PlayerCredits{
		PlayerID:    playerID,
		Balance:     balance,
		TotalEarned: totalEarned,
	}, nil
}

func (s *NebulaService) ListCreditsTransactions(ctx context.Context, playerID int) ([]CreditsTransaction, error) {
	return s.repo.ListCreditsTransactions(ctx, playerID, 50)
}

func (s *NebulaService) CalculateSpeedUpCost(seconds int) int {
	cost := (seconds + 899) / 900
	if cost < 1 {
		return 1
	}
	return cost
}

func (s *NebulaService) SpendDarkMatter(ctx context.Context, playerID, amount int, reason string) (int, error) {
	if err := s.repo.SpendDarkMatter(ctx, playerID, amount); err != nil {
		return 0, err
	}
	balance, _, err := s.repo.GetDarkMatterBalance(ctx, playerID)
	if err != nil {
		return 0, err
	}
	if err := s.repo.AddDMTransaction(ctx, playerID, -amount, balance, reason); err != nil {
		slog.Error("log dm transaction failed", "error", err)
	}
	return balance, nil
}

func (s *NebulaService) SpeedUp(ctx context.Context, playerID int, seconds int) (int, int, error) {
	cost := s.CalculateSpeedUpCost(seconds)
	if _, err := s.SpendDarkMatter(ctx, playerID, cost, "speed_up"); err != nil {
		return 0, 0, err
	}
	return cost, seconds, nil
}

func (s *NebulaService) HireCommander(ctx context.Context, playerID int, commanderType string) (CommanderEntry, error) {
	var config *CommanderConfig
	for i := range Commanders {
		if Commanders[i].Type == commanderType {
			config = &Commanders[i]
			break
		}
	}
	if config == nil {
		return CommanderEntry{}, fmt.Errorf("unknown commander type: %s", commanderType)
	}
	existing, err := s.repo.GetActiveCommanders(ctx, playerID)
	if err != nil {
		return CommanderEntry{}, err
	}
	for _, c := range existing {
		if c.CommanderType == commanderType {
			return CommanderEntry{}, fmt.Errorf("already have active %s commander", commanderType)
		}
	}
	expiresAt := time.Now().AddDate(0, 0, config.Duration)
	if _, err := s.SpendDarkMatter(ctx, playerID, config.DMCost, "hire_"+commanderType); err != nil {
		return CommanderEntry{}, err
	}
	entry, err := s.repo.HireCommander(ctx, playerID, commanderType, 1, expiresAt)
	if err != nil {
		// Refund the DM since the hire failed
		if refundErr := s.repo.AddDarkMatter(ctx, playerID, config.DMCost); refundErr != nil {
			slog.Error("failed to refund DM for commander hire failure", "error", refundErr)
		}
		return CommanderEntry{}, fmt.Errorf("hire commander: %w", err)
	}
	entry.Name = config.Name
	entry.Description = config.Description
	days := int(time.Until(entry.ExpiresAt).Hours() / 24)
	if days < 0 {
		days = 0
	}
	entry.DaysRemaining = days
	return entry, nil
}

func (s *NebulaService) GetPlayerCommanders(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	all, err := s.repo.GetActiveCommanders(ctx, playerID)
	if err != nil {
		return nil, err
	}
	for i := range all {
		for _, c := range Commanders {
			if c.Type == all[i].CommanderType {
				all[i].Name = c.Name
				all[i].Description = c.Description
				break
			}
		}
		days := int(time.Until(all[i].ExpiresAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		all[i].DaysRemaining = days
	}
	return all, nil
}

func (s *NebulaService) GetAvailableCommanders() []CommanderConfig {
	return Commanders
}

func (s *NebulaService) GetActiveCommandersRaw(ctx context.Context, playerID int) ([]CommanderEntry, error) {
	return s.repo.GetActiveCommanders(ctx, playerID)
}

func (s *NebulaService) ClaimDailyGift(ctx context.Context, playerID int) (DailyGiftResult, error) {
	_, _, lastClaimDate, err := s.repo.GetDailyGiftStatus(ctx, playerID)
	if err != nil {
		return DailyGiftResult{}, fmt.Errorf("get gift status: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	if lastClaimDate == today {
		return DailyGiftResult{}, fmt.Errorf("already claimed today")
	}

	newStreakDay, newConsecutiveDays, err := s.repo.ClaimDailyGift(ctx, playerID)
	if err != nil {
		return DailyGiftResult{}, fmt.Errorf("claim gift: %w", err)
	}

	giftIdx := newStreakDay - 1
	if giftIdx < 0 {
		giftIdx = 0
	}
	rewards := make(map[string]int)
	for k, v := range dailyGiftRewards[giftIdx] {
		rewards[k] = v
	}

	if dm, ok := rewards["dm"]; ok && dm > 0 {
		if err := s.addDarkMatter(ctx, playerID, dm); err != nil {
			slog.Error("add dm from daily gift failed", "player", playerID, "error", err)
		}
	}

	if newStreakDay == 7 {
		if err := s.repo.ResetDailyGiftStreak(ctx, playerID); err != nil {
			slog.Error("reset daily gift streak after day 7", "error", err)
		}
	}

	return DailyGiftResult{
		StreakDay:       newStreakDay,
		ConsecutiveDays: newConsecutiveDays,
		Rewards:         rewards,
	}, nil
}

func (s *NebulaService) GetDailyGiftStatus(ctx context.Context, playerID int) (DailyGiftStatus, error) {
	streakDay, consecutiveDays, lastClaimDate, err := s.repo.GetDailyGiftStatus(ctx, playerID)
	if err != nil {
		return DailyGiftStatus{}, err
	}

	today := time.Now().Format("2006-01-02")
	canClaim := lastClaimDate != today

	giftIdx := streakDay
	if giftIdx > 6 {
		giftIdx = 0
	}
	preview := dailyGiftDescriptions[giftIdx]

	return DailyGiftStatus{
		StreakDay:       streakDay,
		ConsecutiveDays: consecutiveDays,
		CanClaim:        canClaim,
		GiftPreview:     preview,
	}, nil
}

func (s *NebulaService) GetDailyTasks(ctx context.Context, playerID int) ([]DailyTask, error) {
	tasks, err := s.repo.GetDailyTasks(ctx, playerID)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		assigned, err := s.assignDailyTasks(ctx, playerID)
		if err != nil {
			return nil, err
		}
		return assigned, nil
	}

	return toDailyTasks(tasks), nil
}

func (s *NebulaService) assignDailyTasks(ctx context.Context, playerID int) ([]DailyTask, error) {
	shuffled := make([]TaskDefinition, len(dailyTaskPool))
	copy(shuffled, dailyTaskPool)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	count := 3
	if len(shuffled) < count {
		count = len(shuffled)
	}

	rows := make([]dailyTaskRow, 0, count)
	for i := 0; i < count; i++ {
		resJSON, _ := json.Marshal(shuffled[i].RewardResources)
		rows = append(rows, dailyTaskRow{
			PlayerID:       playerID,
			TaskType:       shuffled[i].Type,
			Description:    shuffled[i].Description,
			TargetAmount:   shuffled[i].Target,
			Progress:       0,
			RewardDM:       shuffled[i].RewardDM,
			RewardResources: string(resJSON),
			Completed:      false,
			Claimed:        false,
			AssignedDate:   time.Now().Format("2006-01-02"),
		})
	}

	if err := s.repo.AssignDailyTasks(ctx, playerID, rows); err != nil {
		return nil, err
	}

	saved, err := s.repo.GetDailyTasks(ctx, playerID)
	if err != nil {
		return nil, err
	}

	return toDailyTasks(saved), nil
}

func (s *NebulaService) UpdateTaskProgress(ctx context.Context, playerID int, taskType string, amount int) error {
	tasks, err := s.repo.GetDailyTasks(ctx, playerID)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if t.TaskType == taskType && !t.Completed && !t.Claimed {
			if err := s.repo.UpdateTaskProgress(ctx, t.ID, playerID, amount); err != nil {
				return err
			}
			if t.Progress+amount >= t.TargetAmount {
				if err := s.repo.MarkTaskCompleted(ctx, t.ID, playerID); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

func (s *NebulaService) ClaimTask(ctx context.Context, playerID, taskID int) (DailyTask, error) {
	task, err := s.repo.ClaimTaskReward(ctx, taskID, playerID)
	if err != nil {
		return DailyTask{}, err
	}

	if task.RewardDM > 0 {
		if err := s.addDarkMatter(ctx, playerID, task.RewardDM); err != nil {
			return DailyTask{}, fmt.Errorf("add dm reward: %w", err)
		}
	}

	rewardMap := make(map[string]int)
	if task.RewardResources != "" && task.RewardResources != "{}" {
		json.Unmarshal([]byte(task.RewardResources), &rewardMap)
	}

	return DailyTask{
		ID:              task.ID,
		PlayerID:        task.PlayerID,
		TaskType:        task.TaskType,
		Description:     task.Description,
		TargetAmount:    task.TargetAmount,
		Progress:        task.Progress,
		RewardDM:        task.RewardDM,
		RewardResources: rewardMap,
		Completed:       task.Completed,
		Claimed:         task.Claimed,
		AssignedDate:    task.AssignedDate,
	}, nil
}

func (s *NebulaService) ClaimAllTasks(ctx context.Context, playerID int) ([]DailyTask, error) {
	tasks, err := s.repo.ClaimAllTasksReward(ctx, playerID)
	if err != nil {
		return nil, err
	}

	var result []DailyTask
	for _, t := range tasks {
		if t.RewardDM > 0 {
			if err := s.addDarkMatter(ctx, playerID, t.RewardDM); err != nil {
				slog.Error("add dm from task reward failed", "player", playerID, "error", err)
			}
		}
		rewardMap := make(map[string]int)
		if t.RewardResources != "" && t.RewardResources != "{}" {
			json.Unmarshal([]byte(t.RewardResources), &rewardMap)
		}
		result = append(result, DailyTask{
			ID:              t.ID,
			PlayerID:        t.PlayerID,
			TaskType:        t.TaskType,
			Description:     t.Description,
			TargetAmount:    t.TargetAmount,
			Progress:        t.Progress,
			RewardDM:        t.RewardDM,
			RewardResources: rewardMap,
			Completed:       t.Completed,
			Claimed:         t.Claimed,
			AssignedDate:    t.AssignedDate,
		})
	}
	return result, nil
}

func (s *NebulaService) RerollTask(ctx context.Context, playerID, taskID int) (DailyTask, error) {
	tasks, err := s.repo.GetDailyTasks(ctx, playerID)
	if err != nil {
		return DailyTask{}, err
	}

	if len(tasks) == 0 {
		return DailyTask{}, fmt.Errorf("no tasks for today")
	}

	if tasks[0].RerollsUsed >= 1 {
		return DailyTask{}, fmt.Errorf("no rerolls remaining for today")
	}

	if err := s.repo.IncrementTaskRerolls(ctx, playerID); err != nil {
		return DailyTask{}, err
	}

	if _, err := s.repo.RerollTask(ctx, taskID, playerID); err != nil {
		return DailyTask{}, err
	}

	available := make([]TaskDefinition, 0)
	currentTypes := make(map[string]bool)
	for _, t := range tasks {
		currentTypes[t.TaskType] = true
	}
	for _, td := range dailyTaskPool {
		if !currentTypes[td.Type] {
			available = append(available, td)
		}
	}
	if len(available) == 0 {
		available = dailyTaskPool
	}

	chosen := available[rand.Intn(len(available))]
	resJSON, _ := json.Marshal(chosen.RewardResources)
	newTaskRow := dailyTaskRow{
		PlayerID:       playerID,
		TaskType:       chosen.Type,
		Description:    chosen.Description,
		TargetAmount:   chosen.Target,
		Progress:       0,
		RewardDM:       chosen.RewardDM,
		RewardResources: string(resJSON),
		Completed:      false,
		Claimed:        false,
		AssignedDate:   time.Now().Format("2006-01-02"),
	}

	if err := s.repo.AssignDailyTasks(ctx, playerID, []dailyTaskRow{newTaskRow}); err != nil {
		return DailyTask{}, err
	}

	newTasks, err := s.repo.GetDailyTasks(ctx, playerID)
	if err != nil {
		return DailyTask{}, err
	}

	for _, t := range newTasks {
		if t.TaskType == chosen.Type && !t.Claimed && t.Progress == 0 {
			rewardMap := make(map[string]int)
			if t.RewardResources != "" && t.RewardResources != "{}" {
				json.Unmarshal([]byte(t.RewardResources), &rewardMap)
			}
			return DailyTask{
				ID:              t.ID,
				PlayerID:        t.PlayerID,
				TaskType:        t.TaskType,
				Description:     t.Description,
				TargetAmount:    t.TargetAmount,
				Progress:        t.Progress,
				RewardDM:        t.RewardDM,
				RewardResources: rewardMap,
				Completed:       t.Completed,
				Claimed:         t.Claimed,
				AssignedDate:    t.AssignedDate,
			}, nil
		}
	}

	return DailyTask{}, fmt.Errorf("failed to assign new task after reroll")
}

func toDailyTasks(rows []dailyTaskRow) []DailyTask {
	result := make([]DailyTask, len(rows))
	for i, t := range rows {
		rewardMap := make(map[string]int)
		if t.RewardResources != "" && t.RewardResources != "{}" {
			json.Unmarshal([]byte(t.RewardResources), &rewardMap)
		}
		result[i] = DailyTask{
			ID:              t.ID,
			PlayerID:        t.PlayerID,
			TaskType:        t.TaskType,
			Description:     t.Description,
			TargetAmount:    t.TargetAmount,
			Progress:        t.Progress,
			RewardDM:        t.RewardDM,
			RewardResources: rewardMap,
			Completed:       t.Completed,
			Claimed:         t.Claimed,
			AssignedDate:    t.AssignedDate,
		}
	}
	return result
}

func (s *NebulaService) generateDailyTasks(playerID int) []dailyTaskRow {
	rand.Shuffle(len(dailyTaskPool), func(i, j int) {
		dailyTaskPool[i], dailyTaskPool[j] = dailyTaskPool[j], dailyTaskPool[i]
	})
	count := 3
	if count > len(dailyTaskPool) {
		count = len(dailyTaskPool)
	}
	tasks := make([]dailyTaskRow, count)
	for i := 0; i < count; i++ {
		td := dailyTaskPool[i]
		tasks[i] = dailyTaskRow{
			PlayerID:       playerID,
			TaskType:       td.Type,
			Description:    td.Description,
			TargetAmount:   td.Target,
			Progress:       0,
			RewardDM:       td.RewardDM,
			RewardResources: "",
			Completed:      false,
			Claimed:        false,
		}
	}
	return tasks
}

func (s *NebulaService) ListStoreItems() []StoreItem {
	return StoreItems
}

func (s *NebulaService) BuyItem(ctx context.Context, playerID int, itemID string) (map[string]any, error) {
	var item *StoreItem
	for i := range StoreItems {
		if StoreItems[i].ID == itemID {
			item = &StoreItems[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}

	if item.Currency == "dm" {
		if err := s.repo.SpendDarkMatter(ctx, playerID, item.Cost); err != nil {
			return nil, fmt.Errorf("insufficient dark matter")
		}
		balance, _, _ := s.repo.GetDarkMatterBalance(ctx, playerID)
		s.repo.AddDMTransaction(ctx, playerID, -item.Cost, balance, "store_"+itemID)
	} else if item.Currency == "credits" {
		if err := s.repo.SpendCredits(ctx, playerID, item.Cost); err != nil {
			return nil, fmt.Errorf("insufficient credits")
		}
		balance, _, _ := s.repo.GetCreditsBalance(ctx, playerID)
		s.repo.AddCreditsTransaction(ctx, playerID, -item.Cost, balance, "store_"+itemID)
	} else {
		return nil, fmt.Errorf("unknown currency: %s", item.Currency)
	}

	if err := s.repo.CreateStorePurchase(ctx, playerID, itemID, item.Cost, item.Currency); err != nil {
		slog.Error("log store purchase failed", "error", err)
	}

	rewards := make(map[string]any)

	switch item.Type {
	case "shards":
		if err := s.repo.AddGalactoniteShards(ctx, playerID, item.Subtype, item.Count); err != nil {
			slog.Error("add shards failed", "error", err)
			return nil, fmt.Errorf("failed to grant shards")
		}
		rewards["shards"] = item.Count

	case "commander_extension":
		for _, c := range Commanders {
			if err := s.repo.ExtendCommanderDuration(ctx, playerID, c.Type, item.Count); err != nil {
				slog.Error("extend commander duration failed", "commander", c.Type, "error", err)
			}
		}
		rewards["extension_days"] = item.Count

	case "planet_shifter":
		uses, err := s.repo.GetPlayerShifterUses(ctx, playerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shifter uses")
		}
		if err := s.repo.SetPlayerShifterUses(ctx, playerID, uses+item.Count); err != nil {
			return nil, fmt.Errorf("failed to grant shifter uses")
		}
		rewards["shifter_uses"] = item.Count

	case "resources":
		planetID, err := s.repo.GetPlayerPlanetID(ctx, playerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get planet")
		}
		if err := s.addResourceToPlanet(ctx, planetID, "metal", item.Count); err != nil {
			slog.Error("add resource pack metal failed", "error", err)
		}
		if err := s.addResourceToPlanet(ctx, planetID, "crystal", item.Count); err != nil {
			slog.Error("add resource pack crystal failed", "error", err)
		}
		if err := s.addResourceToPlanet(ctx, planetID, "gas", item.Count); err != nil {
			slog.Error("add resource pack gas failed", "error", err)
		}
		rewards["resources"] = item.Count

	case "speed_up":
		rewards["speed_up_seconds"] = item.Count
	}

	return map[string]any{
		"item_id": itemID,
		"cost":    item.Cost,
		"currency": item.Currency,
		"rewards": rewards,
	}, nil
}

func (s *NebulaService) getEspionageTechLevel(ctx context.Context, playerID int) int {
	body, _ := json.Marshal(map[string]int{"player_id": playerID})
	resp, err := s.httpClient.Post(s.planetBaseURL+"/internal/player/techs", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var result struct {
		Technologies map[string]int `json:"technologies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}
	return result.Technologies["espionage_tech"]
}

func (s *NebulaService) GetGalactoniteDiscovererLevel(ctx context.Context, playerID int) (int, error) {
	return s.repo.GetGalactoniteDiscovererLevel(ctx, playerID)
}

func (s *NebulaService) UpgradeGalactoniteDiscoverer(ctx context.Context, playerID int) (int, error) {
	currentLevel, err := s.repo.GetGalactoniteDiscovererLevel(ctx, playerID)
	if err != nil {
		return 0, err
	}
	cost := (currentLevel + 1) * 100
	if _, err := s.SpendDarkMatter(ctx, playerID, cost, "upgrade_discoverer"); err != nil {
		return 0, err
	}
	return s.repo.UpgradeGalactoniteDiscoverer(ctx, playerID)
}
