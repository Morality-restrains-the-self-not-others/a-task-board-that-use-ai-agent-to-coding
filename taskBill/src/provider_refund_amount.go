package main

import "fmt"

// proportionalAmountMinor maps points onto original amount_minor by original points ratio.
func proportionalAmountMinor(amountMinor, originalPoints, refundPoints int64) int64 {
	if refundPoints <= 0 || originalPoints <= 0 || amountMinor <= 0 {
		return 0
	}
	if refundPoints >= originalPoints {
		return amountMinor
	}
	return amountMinor * refundPoints / originalPoints
}

// formatMoneyMinor formats ISO minor units as a 2-decimal string (e.g. 1050 → "10.50").
func formatMoneyMinor(minor int64) string {
	if minor < 0 {
		minor = -minor
	}
	return fmt.Sprintf("%d.%02d", minor/100, minor%100)
}
