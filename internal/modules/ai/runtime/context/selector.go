package context

// Message is the runtime context unit that can be budgeted and compressed.
type Message struct {
	Role    string
	Content string
	Pinned  bool
}

// Budget describes how many pinned, recent, and historical messages to retain.
type Budget struct {
	Pinned  int
	Recent  int
	History int
}

// DefaultBudget is the shared chat context policy for app/runtime callers.
var DefaultBudget = Budget{
	Pinned:  1,
	Recent:  12,
	History: 6,
}

// SelectBudgeted returns a budgeted slice of messages while preserving stable order.
func SelectBudgeted(history []Message, budget Budget) []Message {
	if len(history) == 0 {
		return nil
	}

	budget = normalizeBudget(budget)

	pinned := make([]Message, 0, budget.Pinned)
	nonPinned := make([]Message, 0, len(history))
	for _, msg := range history {
		if msg.Pinned && len(pinned) < budget.Pinned {
			pinned = append(pinned, msg)
			continue
		}
		if !msg.Pinned {
			nonPinned = append(nonPinned, msg)
		}
	}

	recentCount := budget.Recent
	if recentCount < 0 {
		recentCount = 0
	}
	if recentCount > len(nonPinned) {
		recentCount = len(nonPinned)
	}
	recentStart := len(nonPinned) - recentCount
	recent := append([]Message(nil), nonPinned[recentStart:]...)

	historyCount := budget.History
	if historyCount < 0 {
		historyCount = 0
	}
	if historyCount > recentStart {
		historyCount = recentStart
	}
	olderStart := recentStart - historyCount
	older := append([]Message(nil), nonPinned[olderStart:recentStart]...)

	out := make([]Message, 0, len(pinned)+len(older)+len(recent))
	out = append(out, pinned...)
	out = append(out, older...)
	out = append(out, recent...)
	return out
}

func normalizeBudget(budget Budget) Budget {
	if budget == (Budget{}) {
		return DefaultBudget
	}
	if budget.Pinned < 0 {
		budget.Pinned = 0
	}
	if budget.Recent < 0 {
		budget.Recent = 0
	}
	if budget.History < 0 {
		budget.History = 0
	}
	return budget
}
