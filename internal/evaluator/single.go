package evaluator

import (
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/shazmughal/orderbook/internal/model"
)

// SingleRecordEvaluator checks individual records against single-type alert rules.
type SingleRecordEvaluator struct {
	mu       sync.RWMutex
	programs map[string]*vm.Program // rule ID -> compiled expression
	rules    map[string]*model.AlertRule
}

func NewSingleRecordEvaluator() *SingleRecordEvaluator {
	return &SingleRecordEvaluator{
		programs: make(map[string]*vm.Program),
		rules:    make(map[string]*model.AlertRule),
	}
}

// LoadRules compiles and caches alert rule expressions.
func (e *SingleRecordEvaluator) LoadRules(rules []model.AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	newPrograms := make(map[string]*vm.Program, len(rules))
	newRules := make(map[string]*model.AlertRule, len(rules))

	for i := range rules {
		r := &rules[i]
		if r.EvalType != model.EvalTypeSingle || !r.Enabled {
			continue
		}

		prog, err := expr.Compile(r.Condition, expr.AsBool())
		if err != nil {
			return fmt.Errorf("compile rule %q (%s): %w", r.Name, r.Condition, err)
		}
		newPrograms[r.ID] = prog
		newRules[r.ID] = r
	}

	e.programs = newPrograms
	e.rules = newRules
	return nil
}

// CandidateAlert is emitted when a rule matches a record.
type CandidateAlert struct {
	Rule         *model.AlertRule
	Record       *model.DataRecord
	MatchedValue string
}

// Evaluate checks all loaded single-record rules against the given record.
func (e *SingleRecordEvaluator) Evaluate(record *model.DataRecord) []CandidateAlert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	env := map[string]interface{}{
		"source_id": record.SourceID,
		"timestamp": record.Timestamp,
		"payload":   record.Payload,
	}

	var candidates []CandidateAlert
	for id, prog := range e.programs {
		rule := e.rules[id]
		if !rule.MatchesSource(record.SourceID) {
			continue
		}

		result, err := expr.Run(prog, env)
		if err != nil {
			continue
		}

		matched, ok := result.(bool)
		if !ok || !matched {
			continue
		}

		candidates = append(candidates, CandidateAlert{
			Rule:         rule,
			Record:       record,
			MatchedValue: fmt.Sprintf("%v", record.Payload),
		})
	}

	return candidates
}
