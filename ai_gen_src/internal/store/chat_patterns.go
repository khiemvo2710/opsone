package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// CommandPatternStatus mirrors chat_command_patterns.status.
type CommandPatternStatus string

const (
	PatternCandidate  CommandPatternStatus = "candidate"
	PatternApproved   CommandPatternStatus = "approved"
	PatternDeprecated CommandPatternStatus = "deprecated"
)

// CommandPattern is one row from chat_command_patterns.
type CommandPattern struct {
	ID           int64
	CommandKey   string
	PatternType  string
	PatternDef   json.RawMessage
	DefaultSlots json.RawMessage
	HitCount     int
	SuccessCount int
	FailCount    int
	Status       CommandPatternStatus
	MinRole      string
	ApprovedBy   string
	ApprovedAt   *time.Time
	CreatedAt    time.Time
}

// FewShotExample is one row from chat_few_shot_examples.
type FewShotExample struct {
	ID               int64
	CommandKey       string
	UserExample      string
	AssistantExample string
	SuccessRate      *float64
	Priority         int
	Status           CommandPatternStatus
	CreatedAt        time.Time
}

// MinePatternResult summarizes one mined candidate.
type MinePatternResult struct {
	CommandKey string
	Upserted   int
}

// MineCommandPatterns reads chat_interaction_log → upserts candidates into chat_command_patterns (§7.6.5.5 P3).
// Logic: group by (route, message_norm), take top hits, extract keywords as candidate patterns.
func (db *DB) MineCommandPatterns(ctx context.Context, minHits int, lookbackHours int) ([]MinePatternResult, error) {
	if minHits <= 0 {
		minHits = 3
	}
	if lookbackHours <= 0 {
		lookbackHours = 168 // 1 week
	}
	// Query: normalized messages grouped by route with enough hits
	const query = `
		SELECT route, message_norm, COUNT(*) AS cnt,
		       SUM(CASE WHEN action_result = 'success' THEN 1 ELSE 0 END) AS ok
		FROM chat_interaction_log
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? HOUR)
		  AND route NOT IN ('stub','unknown','llm')
		  AND message_norm != ''
		GROUP BY route, message_norm
		HAVING cnt >= ?
		ORDER BY ok DESC, cnt DESC
		LIMIT 200`
	rows, err := db.QueryContext(ctx, query, lookbackHours, minHits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type mineRow struct {
		Route    string
		Norm     string
		HitCount int
		OkCount  int
	}
	var items []mineRow
	for rows.Next() {
		var m mineRow
		if err := rows.Scan(&m.Route, &m.Norm, &m.HitCount, &m.OkCount); err != nil {
			continue
		}
		items = append(items, m)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Map route → command_key
	routeToCommandKey := map[string]string{
		"direct_metrics":              "metrics_query",
		"direct_maintenance":          "maintenance_query",
		"direct_list_pending":         "list_pending",
		"direct_approve":              "approve",
		"direct_reject":               "reject",
		"direct_approve_reject_scoped": "approve_reject_scoped",
		"direct_set_maintenance":      "set_maintenance",
		"direct_reopen_service":       "reopen_service",
		"direct_set_scope_auto":       "set_scope_auto",
	}

	resultMap := map[string]*MinePatternResult{}

	const upsert = `
		INSERT INTO chat_command_patterns
			(command_key, pattern_type, pattern_def, hit_count, success_count, fail_count, status)
		VALUES (?, 'keywords', ?, ?, ?, ?, 'candidate')
		ON DUPLICATE KEY UPDATE
			hit_count = hit_count + VALUES(hit_count),
			success_count = success_count + VALUES(success_count),
			fail_count = fail_count + VALUES(fail_count)`

	for _, m := range items {
		cmdKey, ok := routeToCommandKey[m.Route]
		if !ok {
			// Keep route as command_key but strip "direct_" prefix if present
			cmdKey = strings.TrimPrefix(m.Route, "direct_")
		}
		kws := extractKeywords(m.Norm)
		if len(kws) == 0 {
			continue
		}
		def, _ := json.Marshal(map[string]any{
			"keywords": kws,
			"logic":    "all",
			"source":   m.Norm,
		})
		failCount := m.HitCount - m.OkCount
		if failCount < 0 {
			failCount = 0
		}
		if _, err := db.ExecContext(ctx, upsert, cmdKey, def, m.HitCount, m.OkCount, failCount); err != nil {
			continue
		}
		if resultMap[cmdKey] == nil {
			resultMap[cmdKey] = &MinePatternResult{CommandKey: cmdKey}
		}
		resultMap[cmdKey].Upserted++
	}

	results := make([]MinePatternResult, 0, len(resultMap))
	for _, v := range resultMap {
		results = append(results, *v)
	}
	return results, nil
}

// extractKeywords splits a normalized message into meaningful keywords (len >= 3, not stopwords).
func extractKeywords(norm string) []string {
	stopwords := map[string]bool{
		"và": true, "của": true, "cho": true, "với": true, "hay": true,
		"như": true, "các": true, "có": true, "không": true, "đang": true,
		"hiện": true, "tại": true, "đó": true, "này": true, "khi": true,
		"thì": true, "nếu": true, "mà": true, "về": true, "ở": true,
		"đã": true, "sẽ": true, "đến": true, "từ": true, "qua": true,
		"the": true, "and": true, "for": true, "are": true, "was": true,
	}
	words := strings.Fields(norm)
	seen := map[string]bool{}
	var kws []string
	for _, w := range words {
		w = strings.Trim(w, ".,?!:;\"'")
		if len([]rune(w)) < 2 {
			continue
		}
		if stopwords[w] {
			continue
		}
		if seen[w] {
			continue
		}
		seen[w] = true
		kws = append(kws, w)
		if len(kws) >= 6 {
			break
		}
	}
	return kws
}

// ListCommandPatterns returns chat_command_patterns filtered by status (empty = all).
func (db *DB) ListCommandPatterns(ctx context.Context, status string, limit int) ([]CommandPattern, error) {
	if limit <= 0 {
		limit = 100
	}
	args := []any{limit}
	where := ""
	if status != "" {
		where = "WHERE status = ? "
		args = append([]any{status}, args...)
	}
	query := `SELECT id, command_key, pattern_type, pattern_def, default_slots,
	                 hit_count, success_count, fail_count, status, min_role,
	                 COALESCE(approved_by,''), approved_at, created_at
	          FROM chat_command_patterns
	          ` + where + `ORDER BY hit_count DESC LIMIT ?`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommandPattern
	for rows.Next() {
		var p CommandPattern
		var defRaw, slotsRaw []byte
		var approvedAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.CommandKey, &p.PatternType, &defRaw, &slotsRaw,
			&p.HitCount, &p.SuccessCount, &p.FailCount, &p.Status, &p.MinRole,
			&p.ApprovedBy, &approvedAt, &p.CreatedAt,
		); err != nil {
			continue
		}
		p.PatternDef = defRaw
		if len(slotsRaw) > 0 {
			p.DefaultSlots = slotsRaw
		}
		if approvedAt.Valid {
			p.ApprovedAt = &approvedAt.Time
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetCommandPatternByID returns one pattern row.
func (db *DB) GetCommandPatternByID(ctx context.Context, id int64) (CommandPattern, error) {
	const query = `SELECT id, command_key, pattern_type, pattern_def, default_slots,
	                      hit_count, success_count, fail_count, status, min_role,
	                      COALESCE(approved_by,''), approved_at, created_at
	               FROM chat_command_patterns WHERE id = ?`
	var p CommandPattern
	var defRaw, slotsRaw []byte
	var approvedAt sql.NullTime
	err := db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.CommandKey, &p.PatternType, &defRaw, &slotsRaw,
		&p.HitCount, &p.SuccessCount, &p.FailCount, &p.Status, &p.MinRole,
		&p.ApprovedBy, &approvedAt, &p.CreatedAt,
	)
	if err != nil {
		return CommandPattern{}, err
	}
	p.PatternDef = defRaw
	if len(slotsRaw) > 0 {
		p.DefaultSlots = slotsRaw
	}
	if approvedAt.Valid {
		p.ApprovedAt = &approvedAt.Time
	}
	return p, nil
}

// PromoteCommandPattern sets status=approved for the given pattern id.
func (db *DB) PromoteCommandPattern(ctx context.Context, id int64, approvedBy string) error {
	const query = `UPDATE chat_command_patterns
	               SET status = 'approved', approved_by = ?, approved_at = NOW()
	               WHERE id = ? AND status = 'candidate'`
	res, err := db.ExecContext(ctx, query, approvedBy, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeprecateCommandPattern sets status=deprecated.
func (db *DB) DeprecateCommandPattern(ctx context.Context, id int64) error {
	const query = `UPDATE chat_command_patterns SET status = 'deprecated' WHERE id = ?`
	_, err := db.ExecContext(ctx, query, id)
	return err
}

// ListApprovedPatterns returns approved patterns for a given command key (empty = all approved).
func (db *DB) ListApprovedPatterns(ctx context.Context, commandKey string) ([]CommandPattern, error) {
	args := []any{}
	where := "WHERE status = 'approved'"
	if commandKey != "" {
		where += " AND command_key = ?"
		args = append(args, commandKey)
	}
	query := `SELECT id, command_key, pattern_type, pattern_def, default_slots,
	                 hit_count, success_count, fail_count, status, min_role,
	                 COALESCE(approved_by,''), approved_at, created_at
	          FROM chat_command_patterns ` + where + ` ORDER BY hit_count DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommandPattern
	for rows.Next() {
		var p CommandPattern
		var defRaw, slotsRaw []byte
		var approvedAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.CommandKey, &p.PatternType, &defRaw, &slotsRaw,
			&p.HitCount, &p.SuccessCount, &p.FailCount, &p.Status, &p.MinRole,
			&p.ApprovedBy, &approvedAt, &p.CreatedAt,
		); err != nil {
			continue
		}
		p.PatternDef = defRaw
		if len(slotsRaw) > 0 {
			p.DefaultSlots = slotsRaw
		}
		if approvedAt.Valid {
			p.ApprovedAt = &approvedAt.Time
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MatchApprovedPattern checks if userMsg matches any approved pattern for the given commandKey.
// Returns (commandKey, matched) — commandKey may differ if routing by pattern.
func MatchApprovedPattern(msg string, patterns []CommandPattern) (string, bool) {
	norm := normalizeChatPattern(msg)
	words := strings.Fields(norm)
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}
	for _, p := range patterns {
		var def struct {
			Keywords []string `json:"keywords"`
			Logic    string   `json:"logic"`
		}
		if err := json.Unmarshal(p.PatternDef, &def); err != nil {
			continue
		}
		if len(def.Keywords) == 0 {
			continue
		}
		matched := 0
		for _, kw := range def.Keywords {
			if wordSet[kw] || strings.Contains(norm, kw) {
				matched++
			}
		}
		logic := def.Logic
		if logic == "" {
			logic = "all"
		}
		switch logic {
		case "all":
			if matched == len(def.Keywords) {
				return p.CommandKey, true
			}
		case "any":
			if matched > 0 {
				return p.CommandKey, true
			}
		case "majority":
			if matched*2 >= len(def.Keywords) {
				return p.CommandKey, true
			}
		}
	}
	return "", false
}

// --- few-shot examples ---

// GetApprovedFewShotExamples returns top-K approved examples for a command key.
func (db *DB) GetApprovedFewShotExamples(ctx context.Context, commandKey string, limit int) ([]FewShotExample, error) {
	if limit <= 0 {
		limit = 5
	}
	const query = `SELECT id, command_key, user_example, assistant_example,
	                      success_rate, priority, status, created_at
	               FROM chat_few_shot_examples
	               WHERE status = 'approved' AND command_key = ?
	               ORDER BY priority DESC, success_rate DESC
	               LIMIT ?`
	rows, err := db.QueryContext(ctx, query, commandKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FewShotExample
	for rows.Next() {
		var e FewShotExample
		var sr sql.NullFloat64
		if err := rows.Scan(&e.ID, &e.CommandKey, &e.UserExample, &e.AssistantExample,
			&sr, &e.Priority, &e.Status, &e.CreatedAt); err != nil {
			continue
		}
		if sr.Valid {
			e.SuccessRate = &sr.Float64
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListFewShotExamples returns few-shot examples (admin listing).
func (db *DB) ListFewShotExamples(ctx context.Context, commandKey, status string, limit int) ([]FewShotExample, error) {
	if limit <= 0 {
		limit = 100
	}
	args := []any{}
	var conds []string
	if commandKey != "" {
		conds = append(conds, "command_key = ?")
		args = append(args, commandKey)
	}
	if status != "" {
		conds = append(conds, "status = ?")
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	query := `SELECT id, command_key, user_example, assistant_example,
	                 success_rate, priority, status, created_at
	          FROM chat_few_shot_examples ` + where + ` ORDER BY priority DESC, created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FewShotExample
	for rows.Next() {
		var e FewShotExample
		var sr sql.NullFloat64
		if err := rows.Scan(&e.ID, &e.CommandKey, &e.UserExample, &e.AssistantExample,
			&sr, &e.Priority, &e.Status, &e.CreatedAt); err != nil {
			continue
		}
		if sr.Valid {
			e.SuccessRate = &sr.Float64
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PromoteFewShotExample sets a few-shot example to approved.
func (db *DB) PromoteFewShotExample(ctx context.Context, id int64) error {
	const query = `UPDATE chat_few_shot_examples SET status = 'approved' WHERE id = ? AND status = 'candidate'`
	res, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeprecateFewShotExample sets status=deprecated.
func (db *DB) DeprecateFewShotExample(ctx context.Context, id int64) error {
	const query = `UPDATE chat_few_shot_examples SET status = 'deprecated' WHERE id = ?`
	_, err := db.ExecContext(ctx, query, id)
	return err
}

// InsertChatFeedback records user feedback on a chat interaction.
func (db *DB) InsertChatFeedback(ctx context.Context, interactionID int64, rating, userCorrection, expectedCommand string) error {
	var correction sql.NullString
	if strings.TrimSpace(userCorrection) != "" {
		correction = sql.NullString{String: userCorrection, Valid: true}
	}
	var expected sql.NullString
	if strings.TrimSpace(expectedCommand) != "" {
		expected = sql.NullString{String: expectedCommand, Valid: true}
	}
	const query = `INSERT INTO chat_feedback (interaction_id, rating, user_correction, expected_command)
	               VALUES (?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query, interactionID, rating, correction, expected)
	return err
}

// MineFewShotExamples reads successful chat interactions and inserts candidate few-shot examples (§7.6.5.5 P3).
func (db *DB) MineFewShotExamples(ctx context.Context, minSuccessRate float64, lookbackHours int) (int, error) {
	if minSuccessRate <= 0 {
		minSuccessRate = 0.8
	}
	if lookbackHours <= 0 {
		lookbackHours = 168
	}
	// Get sessions with high success rate messages and their assistant replies
	const query = `
		SELECT l.route, l.user_message, l.reply_preview
		FROM chat_interaction_log l
		WHERE l.created_at >= DATE_SUB(NOW(), INTERVAL ? HOUR)
		  AND l.action_result = 'success'
		  AND l.route NOT IN ('stub','unknown')
		  AND l.reply_preview IS NOT NULL
		  AND CHAR_LENGTH(l.reply_preview) > 20
		  AND CHAR_LENGTH(l.user_message) > 5
		ORDER BY l.created_at DESC
		LIMIT 100`
	rows, err := db.QueryContext(ctx, query, lookbackHours)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	routeToCommandKey := map[string]string{
		"direct_metrics":              "metrics_query",
		"direct_maintenance":          "maintenance_query",
		"direct_list_pending":         "list_pending",
		"direct_approve":              "approve",
		"direct_reject":               "reject",
		"direct_set_maintenance":      "set_maintenance",
		"direct_reopen_service":       "reopen_service",
		"direct_set_scope_auto":       "set_scope_auto",
	}

	const upsert = `
		INSERT IGNORE INTO chat_few_shot_examples
			(command_key, user_example, assistant_example, status)
		VALUES (?, ?, ?, 'candidate')`

	inserted := 0
	for rows.Next() {
		var route, userMsg, replyPreview string
		if err := rows.Scan(&route, &userMsg, &replyPreview); err != nil {
			continue
		}
		cmdKey, ok := routeToCommandKey[route]
		if !ok {
			cmdKey = strings.TrimPrefix(route, "direct_")
		}
		if cmdKey == "" {
			continue
		}
		res, err := db.ExecContext(ctx, upsert, cmdKey, userMsg, replyPreview)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}
	return inserted, rows.Err()
}
