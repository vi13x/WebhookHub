package event

type ListOptions struct {
	Type        string
	Limit       int
	Offset      int
	NewestFirst bool
}

type Stats struct {
	Total    int            `json:"total"`
	ByType   map[string]int `json:"by_type"`
	ByStatus map[string]int `json:"by_status"`
}

func (s *Service) Query(options ListOptions) ([]Event, int) {
	items := make([]Event, 0)
	for _, e := range s.store.List() {
		if options.Type == "" || e.Type == options.Type {
			items = append(items, e)
		}
	}
	total := len(items)
	if options.NewestFirst {
		for i, j := 0, total-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}
	start := min(max(options.Offset, 0), total)
	count := min(max(options.Limit, 0), total-start)
	return items[start : start+count], total
}

func (s *Service) Stats() Stats {
	stats := Stats{ByType: make(map[string]int), ByStatus: make(map[string]int)}
	for _, e := range s.store.List() {
		stats.Total++
		stats.ByType[e.Type]++
		stats.ByStatus[e.Status]++
	}
	return stats
}
