package main

import (
	"fmt"
	"math"
)

const pointsPerYuan = 100

func pointsToYuanEquivalentStr(points int64) string {
	sign := ""
	a := points
	if a < 0 {
		sign = "-"
		a = -a
	}
	return fmt.Sprintf("%s%d.%02d", sign, a/pointsPerYuan, a%pointsPerYuan)
}

func yuanFloatToPoints(yuan float64) int64 {
	return int64(math.Round(yuan * pointsPerYuan))
}
