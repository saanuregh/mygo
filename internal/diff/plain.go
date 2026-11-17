package diff

// maxCost is how many differing lines Plain compares before it gives up
// and takes the whole middle as one change (zode's editor/diff.zig).
const maxCost = 4000

// Plain compares two sequences as zode's old Zig editor did
// (editor/diff.zig): the common elements at both ends set aside, Myers's
// O(ND) algorithm on the rest with no sliding of the groups afterwards, so
// a line added among equal ones is the first of the changed that the
// algorithm finds.
func Plain(a, b []string) []Hunk {
	ia, ib := Intern(a, b)
	pre := 0
	for pre < len(ia) && pre < len(ib) && ia[pre] == ib[pre] {
		pre++
	}
	suf := 0
	for suf < len(ia)-pre && suf < len(ib)-pre && ia[len(ia)-1-suf] == ib[len(ib)-1-suf] {
		suf++
	}
	x0, y0 := ia[pre:len(ia)-suf], ib[pre:len(ib)-suf]
	n, m := len(x0), len(y0)
	limit := min(n+m, maxCost)
	off := limit
	v := make([]int, 2*limit+3)
	var trace [][]int
	found := false
	d := 0
outer:
	for ; d <= limit; d++ {
		trace = append(trace, append([]int(nil), v...))
		for k := -d; k <= d; k += 2 {
			ki := k + off
			var x int
			if k == -d || (k != d && v[ki-1] < v[ki+1]) {
				x = v[ki+1]
			} else {
				x = v[ki-1] + 1
			}
			y := x - k
			for x < n && y < m && x0[x] == y0[y] {
				x++
				y++
			}
			v[ki] = x
			if x >= n && y >= m {
				found = true
				break outer
			}
		}
	}
	var hunks []Hunk
	add := func(a0, a1, b0, b1 int) {
		if a0 != a1 || b0 != b1 {
			hunks = append(hunks, Hunk{pre + a0, pre + a1, pre + b0, pre + b1})
		}
	}
	if !found {
		add(0, n, 0, m)
		return hunks
	}
	// Back through the traces for the edit path, as (old, new) steps.
	var steps [][4]int
	x, y := n, m
	for dd := d; dd > 0; dd-- {
		prev := trace[dd]
		k := x - y
		ki := k + off
		down := k == -dd || (k != dd && prev[ki-1] < prev[ki+1])
		pk := k - 1
		if down {
			pk = k + 1
		}
		px := prev[pk+off]
		py := px - pk
		stepX, stepY := 1, 0
		if down {
			stepX, stepY = 0, 1
		}
		for x > px+stepX && y > py+stepY {
			x--
			y--
		}
		if down {
			steps = append(steps, [4]int{x, x, y - 1, y})
		} else {
			steps = append(steps, [4]int{x - 1, x, y, y})
		}
		x, y = px, py
	}
	// The steps come last first: neighbours join into hunks.
	var cur *[4]int
	for i := len(steps) - 1; i >= 0; i-- {
		s := steps[i]
		if cur != nil {
			if s[0] <= cur[1] && s[2] <= cur[3] {
				cur[1] = max(cur[1], s[1])
				cur[3] = max(cur[3], s[3])
				continue
			}
			add(cur[0], cur[1], cur[2], cur[3])
		}
		c := s
		cur = &c
	}
	if cur != nil {
		add(cur[0], cur[1], cur[2], cur[3])
	}
	return hunks
}
