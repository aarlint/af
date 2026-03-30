package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/aarlint/af/internal/models"
)

func GetPresidents(db *sql.DB) ([]models.President, error) {
	rows, err := db.Query("SELECT id, name, term, start_date, end_date FROM presidents ORDER BY start_date")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.President
	for rows.Next() {
		var p models.President
		if err := rows.Scan(&p.ID, &p.Name, &p.Term, &p.StartDate, &p.EndDate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type ActionFilter struct {
	President string
	Category  string
	Country   string
	Impact    string // "positive", "negative", "neutral"
	Search    string
	Sort      string // "date", "impact-desc", "impact-asc"
}

func GetActions(db *sql.DB, f ActionFilter) ([]models.Action, int, error) {
	// Get total count
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM actions").Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build filtered query
	var where []string
	var args []interface{}
	argN := 0

	if f.President != "" {
		argN++
		where = append(where, fmt.Sprintf("a.president = ?%d", argN))
		args = append(args, f.President)
	}
	if f.Category != "" {
		argN++
		where = append(where, fmt.Sprintf("a.category = ?%d", argN))
		args = append(args, f.Category)
	}
	if f.Search != "" {
		argN++
		where = append(where, fmt.Sprintf("LOWER(a.title) LIKE ?%d", argN))
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
	}

	// Country + impact filter requires joining impacts
	needImpactJoin := f.Country != "" && f.Impact != ""
	if needImpactJoin {
		argN++
		where = append(where, fmt.Sprintf("imp_filter.country = ?%d", argN))
		args = append(args, f.Country)

		switch f.Impact {
		case "positive":
			where = append(where, "imp_filter.score > 0")
		case "negative":
			where = append(where, "imp_filter.score < 0")
		case "neutral":
			where = append(where, "imp_filter.score = 0")
		}
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	impactJoin := ""
	if needImpactJoin {
		impactJoin = "JOIN impacts imp_filter ON imp_filter.action_id = a.id"
	}

	// Order
	orderClause := "ORDER BY a.date DESC, a.id DESC"
	if f.Sort == "impact-desc" && f.Country != "" {
		orderClause = "ORDER BY imp_sort.score DESC, a.date DESC"
	} else if f.Sort == "impact-asc" && f.Country != "" {
		orderClause = "ORDER BY imp_sort.score ASC, a.date DESC"
	}

	needSortJoin := (f.Sort == "impact-desc" || f.Sort == "impact-asc") && f.Country != ""
	sortJoin := ""
	if needSortJoin {
		argN++
		sortJoin = fmt.Sprintf("LEFT JOIN impacts imp_sort ON imp_sort.action_id = a.id AND imp_sort.country = ?%d", argN)
		args = append(args, f.Country)
	}

	// Use positional params ($1, $2, ...) — modernc/sqlite uses ? only
	query := fmt.Sprintf(
		"SELECT a.id, a.president, COALESCE(a.eo,''), a.title, a.date, a.category, COALESCE(a.url,'') FROM actions a %s %s %s %s",
		impactJoin, sortJoin, whereClause, orderClause,
	)

	// Replace ?N with ? for modernc/sqlite
	for i := argN; i >= 1; i-- {
		query = strings.ReplaceAll(query, fmt.Sprintf("?%d", i), "?")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query actions: %w", err)
	}
	defer rows.Close()

	var actions []models.Action
	var actionIDs []int
	for rows.Next() {
		var a models.Action
		if err := rows.Scan(&a.ID, &a.President, &a.EO, &a.Title, &a.Date, &a.Category, &a.URL); err != nil {
			return nil, 0, err
		}
		actions = append(actions, a)
		actionIDs = append(actionIDs, a.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Batch load impacts for all actions
	if len(actionIDs) > 0 {
		impactMap, err := getImpactsBatch(db, actionIDs)
		if err != nil {
			return nil, 0, err
		}
		for i := range actions {
			if imp, ok := impactMap[actions[i].ID]; ok {
				actions[i].Impacts = imp.scores
				actions[i].Reasons = imp.reasons
			}
		}
	}

	return actions, total, nil
}

type actionImpacts struct {
	scores  map[string]int
	reasons map[string]string
}

func getImpactsBatch(db *sql.DB, ids []int) (map[int]*actionImpacts, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT action_id, country, score, reason FROM impacts WHERE action_id IN (%s)",
		strings.Join(placeholders, ","),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]*actionImpacts)
	for rows.Next() {
		var actionID, score int
		var country, reason string
		if err := rows.Scan(&actionID, &country, &score, &reason); err != nil {
			return nil, err
		}
		if result[actionID] == nil {
			result[actionID] = &actionImpacts{
				scores:  make(map[string]int),
				reasons: make(map[string]string),
			}
		}
		result[actionID].scores[country] = score
		result[actionID].reasons[country] = reason
	}
	return result, rows.Err()
}

func GetScores(db *sql.DB, presidentFilter string) (models.ScoresResponse, error) {
	var resp models.ScoresResponse

	// Get presidents
	presidents, err := GetPresidents(db)
	if err != nil {
		return resp, err
	}
	resp.Presidents = presidents

	// Get distinct countries
	rows, err := db.Query("SELECT DISTINCT country FROM impacts ORDER BY country")
	if err != nil {
		return resp, err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return resp, err
		}
		resp.Countries = append(resp.Countries, c)
	}
	if err := rows.Err(); err != nil {
		return resp, err
	}

	// Get scores per country
	resp.Scores = make(map[string]models.CountryScore)

	// Build cumulative scores from actions ordered by date
	var actionWhere string
	var actionArgs []interface{}
	if presidentFilter != "" {
		actionWhere = "WHERE a.president = ?"
		actionArgs = append(actionArgs, presidentFilter)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.date, i.country, i.score
		FROM actions a
		JOIN impacts i ON i.action_id = a.id
		%s
		ORDER BY a.date ASC, a.id ASC
	`, actionWhere)

	impactRows, err := db.Query(query, actionArgs...)
	if err != nil {
		return resp, err
	}
	defer impactRows.Close()

	// Track running totals per country
	type running struct {
		total    int
		pos      int
		neg      int
		cumulate []int
	}
	countryData := make(map[string]*running)

	// Track action ordering for cumulative
	type actionCountry struct {
		country string
		score   int
	}
	// Group by action to build cumulative correctly
	type actionEntry struct {
		id       int
		impacts  []actionCountry
	}
	var actionOrder []actionEntry
	currentAction := -1
	var currentEntry *actionEntry

	for impactRows.Next() {
		var actionID, score int
		var date, country string
		if err := impactRows.Scan(&actionID, &date, &country, &score); err != nil {
			return resp, err
		}

		if actionID != currentAction {
			if currentEntry != nil {
				actionOrder = append(actionOrder, *currentEntry)
			}
			currentAction = actionID
			currentEntry = &actionEntry{id: actionID}
		}
		currentEntry.impacts = append(currentEntry.impacts, actionCountry{country: country, score: score})

		if countryData[country] == nil {
			countryData[country] = &running{}
		}
	}
	if currentEntry != nil {
		actionOrder = append(actionOrder, *currentEntry)
	}
	if err := impactRows.Err(); err != nil {
		return resp, err
	}

	// Build cumulative
	for _, ae := range actionOrder {
		for _, ic := range ae.impacts {
			r := countryData[ic.country]
			r.total += ic.score
			if ic.score > 0 {
				r.pos += ic.score
			}
			if ic.score < 0 {
				r.neg += -ic.score
			}
			r.cumulate = append(r.cumulate, r.total)
		}
	}

	for _, c := range resp.Countries {
		r := countryData[c]
		if r == nil {
			r = &running{}
		}
		resp.Scores[c] = models.CountryScore{
			Country:    c,
			Total:      r.total,
			Positive:   r.pos,
			Negative:   r.neg,
			Cumulative: r.cumulate,
		}
	}

	return resp, nil
}

func InsertAction(tx *sql.Tx, president, eo, title, date, category, url string) (int64, error) {
	res, err := tx.Exec(
		"INSERT INTO actions (president, eo, title, date, category, url) VALUES (?, ?, ?, ?, ?, ?)",
		president, eo, title, date, category, url,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func InsertImpact(tx *sql.Tx, actionID int64, country string, score int, reason string) error {
	_, err := tx.Exec(
		"INSERT INTO impacts (action_id, country, score, reason) VALUES (?, ?, ?, ?)",
		actionID, country, score, reason,
	)
	return err
}
