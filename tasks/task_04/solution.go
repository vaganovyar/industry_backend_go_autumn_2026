package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	count := 0
	sum := int64(0)
	minStats := int64(math.MaxInt64)
	maxStats := int64(math.MinInt64)
	for i := 1; i < len(nums); i++ {
		count++
		if i > 0 {
			diff := nums[i] - nums[i-1]
			if diff < minStats {
				minStats = diff
			}
			if diff > maxStats {
				maxStats = diff
			}
			sum += diff
		}
	}

	return Stats{count, sum, minStats, maxStats}
}
