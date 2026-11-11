package sim

func (w *World) buildRoute(fromStar, toStar int, jumpRange float64) []int {
	if fromStar == toStar {
		return []int{fromStar}
	}
	n := len(w.Stars)
	prev := make([]int, n)
	for i := range prev {
		prev[i] = -1
	}
	visited := make([]bool, n)
	visited[fromStar] = true
	queue := []int{fromStar}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == toStar {
			break
		}
		for next := 0; next < n; next++ {
			if visited[next] {
				continue
			}
			if w.Dist(cur, next) <= jumpRange {
				visited[next] = true
				prev[next] = cur
				queue = append(queue, next)
			}
		}
	}
	if !visited[toStar] {
		return nil
	}
	path := []int{}
	for at := toStar; at != -1; at = prev[at] {
		path = append(path, at)
		if at == fromStar {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}