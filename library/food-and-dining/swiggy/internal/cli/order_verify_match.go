// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

type retryVerdict string

const (
	retryAlreadyPlaced retryVerdict = "already_placed"
	retrySafe          retryVerdict = "safe_to_retry"
	retryAmbiguous     retryVerdict = "ambiguous"
)

type retryMatchResult struct {
	Verdict retryVerdict
	Match   map[string]any
	Reason  string
	Checked int
}

type stampedOrder struct {
	order map[string]any
	at    time.Time
}

// evaluateRetryMatch treats amount/restaurant hits as the attempted order only
// when the row has a parseable timestamp inside window. Untimestamped hits
// fail closed so an older same-amount order cannot block a valid retry and
// an undated hit cannot be reported as safe to retry.
func evaluateRetryMatch(orders []map[string]any, amount, restaurantID string, now time.Time, window time.Duration) retryMatchResult {
	var timed []stampedOrder
	var untimedHits []map[string]any
	for _, o := range orders {
		ts, ok := parseOrderTime(o)
		if !ok {
			if amountMatches(o, amount) && restaurantMatches(o, restaurantID) {
				untimedHits = append(untimedHits, o)
			}
			continue
		}
		timed = append(timed, stampedOrder{order: o, at: ts})
	}
	sort.Slice(timed, func(i, j int) bool { return timed[i].at.After(timed[j].at) })

	checked := len(timed) + len(untimedHits)
	var inWindow map[string]any
	for _, row := range timed {
		age := now.Sub(row.at)
		if age < -time.Minute || age > window {
			continue
		}
		if amountMatches(row.order, amount) && restaurantMatches(row.order, restaurantID) {
			inWindow = row.order
			break
		}
	}
	if inWindow != nil {
		return retryMatchResult{Verdict: retryAlreadyPlaced, Match: inWindow, Reason: "matching order found inside the recent time window", Checked: checked}
	}
	if len(untimedHits) > 0 {
		return retryMatchResult{
			Verdict: retryAmbiguous,
			Match:   untimedHits[0],
			Reason:  "cannot confirm whether the order placed: a matching amount exists but that order has no timestamp, so it is not safe to retry or to treat it as already placed",
			Checked: checked,
		}
	}
	return retryMatchResult{Verdict: retrySafe, Reason: "no matching order inside the recent time window", Checked: checked}
}

func restaurantMatches(order map[string]any, want string) bool {
	if want == "" {
		return true
	}
	for _, key := range []string{"restaurantId", "restaurant_id", "storeId"} {
		if rid, ok := order[key].(string); ok && rid == want {
			return true
		}
	}
	return false
}

func parseOrderTime(order map[string]any) (time.Time, bool) {
	for _, key := range []string{
		"orderTime", "orderedAt", "placedAt", "placedOn", "createdAt", "created_at",
		"orderDate", "orderedOn", "timestamp", "placed_time", "order_time", "time",
	} {
		v, ok := order[key]
		if !ok || v == nil {
			continue
		}
		if ts, ok := coerceOrderTime(v); ok {
			return ts, true
		}
	}
	return time.Time{}, false
}

func coerceOrderTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return unixLikeTime(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return time.Time{}, false
		}
		return unixLikeTime(f)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, time.RFC1123, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts, true
			}
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return unixLikeTime(f)
		}
	}
	return time.Time{}, false
}

func unixLikeTime(n float64) (time.Time, bool) {
	if n <= 0 {
		return time.Time{}, false
	}
	sec := n
	if n > 1e12 {
		sec = n / 1000
	}
	ts := time.Unix(int64(sec), 0).UTC()
	if ts.Year() < 2000 || ts.Year() > 2100 {
		return time.Time{}, false
	}
	return ts, true
}
