// Package difflib は Python の difflib.SequenceMatcher（isjunk=None, autojunk=False）を移植する。
package difflib

import "sort"

// Opcode は get_opcodes() の 1 項目。
type Opcode struct {
	Tag            string
	I1, I2, J1, J2 int
}

// Matcher は 2 つのルーン列を比べる。
type Matcher struct {
	a, b     []rune
	b2j      map[rune][]int
	matching [][3]int
}

// New は SequenceMatcher(None, a, b, autojunk=False) と同じものを作る。
func New(a, b []rune) *Matcher {
	m := &Matcher{a: a, b: b, b2j: map[rune][]int{}}
	for j, r := range b {
		m.b2j[r] = append(m.b2j[r], j)
	}
	return m
}

func (m *Matcher) findLongestMatch(alo, ahi, blo, bhi int) (besti, bestj, bestsize int) {
	besti, bestj = alo, blo
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newj2len := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newj2len[j] = k
			if k > bestsize {
				besti, bestj, bestsize = i-k+1, j-k+1, k
			}
		}
		j2len = newj2len
	}
	for besti > alo && bestj > blo && m.a[besti-1] == m.b[bestj-1] {
		besti, bestj, bestsize = besti-1, bestj-1, bestsize+1
	}
	for besti+bestsize < ahi && bestj+bestsize < bhi && m.a[besti+bestsize] == m.b[bestj+bestsize] {
		bestsize++
	}
	return besti, bestj, bestsize
}

func (m *Matcher) matchingBlocks() [][3]int {
	if m.matching != nil {
		return m.matching
	}
	la, lb := len(m.a), len(m.b)
	queue := [][4]int{{0, la, 0, lb}}
	var blocks [][3]int
	for len(queue) > 0 {
		q := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		alo, ahi, blo, bhi := q[0], q[1], q[2], q[3]
		i, j, k := m.findLongestMatch(alo, ahi, blo, bhi)
		if k == 0 {
			continue
		}
		blocks = append(blocks, [3]int{i, j, k})
		if alo < i && blo < j {
			queue = append(queue, [4]int{alo, i, blo, j})
		}
		if i+k < ahi && j+k < bhi {
			queue = append(queue, [4]int{i + k, ahi, j + k, bhi})
		}
	}
	sort.Slice(blocks, func(x, y int) bool {
		for n := 0; n < 3; n++ {
			if blocks[x][n] != blocks[y][n] {
				return blocks[x][n] < blocks[y][n]
			}
		}
		return false
	})
	var out [][3]int
	i1, j1, k1 := 0, 0, 0
	for _, b := range blocks {
		if i1+k1 == b[0] && j1+k1 == b[1] {
			k1 += b[2]
			continue
		}
		if k1 > 0 {
			out = append(out, [3]int{i1, j1, k1})
		}
		i1, j1, k1 = b[0], b[1], b[2]
	}
	if k1 > 0 {
		out = append(out, [3]int{i1, j1, k1})
	}
	out = append(out, [3]int{la, lb, 0})
	m.matching = out
	return m.matching
}

// Opcodes は get_opcodes() と同じ。
func (m *Matcher) Opcodes() []Opcode {
	var out []Opcode
	i, j := 0, 0
	for _, b := range m.matchingBlocks() {
		ai, bj, size := b[0], b[1], b[2]
		tag := ""
		switch {
		case i < ai && j < bj:
			tag = "replace"
		case i < ai:
			tag = "delete"
		case j < bj:
			tag = "insert"
		}
		if tag != "" {
			out = append(out, Opcode{tag, i, ai, j, bj})
		}
		i, j = ai+size, bj+size
		if size > 0 {
			out = append(out, Opcode{"equal", ai, i, bj, j})
		}
	}
	return out
}

// Ratio は ratio() と同じ。
func (m *Matcher) Ratio() float64 {
	matches := 0
	for _, b := range m.matchingBlocks() {
		matches += b[2]
	}
	total := len(m.a) + len(m.b)
	if total == 0 {
		return 1.0
	}
	return 2.0 * float64(matches) / float64(total)
}
