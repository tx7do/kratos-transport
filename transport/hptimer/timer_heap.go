package hptimer

// timerHeap 最小堆，用于管理定时任务（触发时间越早，优先级越高）
type timerHeap []*TimerTask

func (h timerHeap) Len() int           { return len(h) }
func (h timerHeap) Less(i, j int) bool { return h[i].At.Before(h[j].At) }
func (h timerHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *timerHeap) Push(x interface{}) {
	*h = append(*h, x.(*TimerTask))
}

func (h *timerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
