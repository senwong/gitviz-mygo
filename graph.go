package main

func assignLanes(commits []Commit) int {
	var lanes []string
	free := func() int {
		for i, s := range lanes {
			if s == "" {
				return i
			}
		}
		lanes = append(lanes, "")
		return len(lanes) - 1
	}
	maximum := 1
	for i := range commits {
		c := &commits[i]
		before := append([]string(nil), lanes...)
		var arriving []int
		for l, s := range lanes {
			if s == c.SHA {
				arriving = append(arriving, l)
			}
		}
		if len(arriving) > 0 {
			c.Lane = arriving[0]
			c.Top = true
		} else {
			c.Lane = free()
		}
		arrived := map[int]bool{}
		for _, l := range arriving {
			arrived[l] = true
			if l != c.Lane {
				c.Incoming = append(c.Incoming, l)
				lanes[l] = ""
			}
		}
		lanes[c.Lane] = ""
		if len(c.Parents) > 0 {
			lanes[c.Lane] = c.Parents[0]
			c.Bottom = true
		}
		for _, parent := range c.Parents[minimum(1, len(c.Parents)):] {
			found := false
			for _, s := range lanes {
				if s == parent {
					found = true
					break
				}
			}
			if !found {
				l := free()
				lanes[l] = parent
				c.Outgoing = append(c.Outgoing, l)
			}
		}
		for l, s := range before {
			if s != "" && l != c.Lane && !arrived[l] && l < len(lanes) && lanes[l] != "" {
				c.Through = append(c.Through, l)
			}
		}
		maximum = max(maximum, len(lanes))
	}
	return maximum
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}
