// Package schedule 提供简单的 Cron 表达式解析（5字段：分 时 日 月 周）
package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseNextTime 解析 5 字段 Cron 表达式，返回 after 时刻之后的下次触发时间。
// 格式：分(0-59) 时(0-23) 日(1-31) 月(1-12) 周(0-6，0=周日)
// 支持：数字、* 、*/n（步进）、a-b（范围）、逗号列表
func ParseNextTime(expr string, after time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron: expected 5 fields, got %d", len(fields))
	}

	minutes, err := parseField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron minute: %w", err)
	}
	hours, err := parseField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron hour: %w", err)
	}
	days, err := parseField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron day: %w", err)
	}
	months, err := parseField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron month: %w", err)
	}
	weekdays, err := parseField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron weekday: %w", err)
	}

	// 从 after 的下一分钟开始逐分搜索（最多搜索 4 年）
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.Add(4 * 365 * 24 * time.Hour)

	for t.Before(limit) {
		if !inSet(months, int(t.Month())) {
			// 跳到下个月 1 日 0:00
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !inSet(days, t.Day()) || !inSet(weekdays, int(t.Weekday())) {
			// 跳到明天 0:00
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !inSet(hours, t.Hour()) {
			// 跳到下一小时 :00
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}
		if !inSet(minutes, t.Minute()) {
			t = t.Add(time.Minute)
			continue
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cron: no trigger time found within 4 years for expr %q", expr)
}

// parseField 解析单个 cron 字段，返回合法值集合。
func parseField(field string, min, max int) (map[int]struct{}, error) {
	set := make(map[int]struct{})

	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "*" {
			for i := min; i <= max; i++ {
				set[i] = struct{}{}
			}
			continue
		}

		// */n 步进
		if strings.HasPrefix(part, "*/") {
			step, err := strconv.Atoi(part[2:])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step %q", part)
			}
			for i := min; i <= max; i += step {
				set[i] = struct{}{}
			}
			continue
		}

		// a-b 范围
		if idx := strings.Index(part, "-"); idx >= 0 {
			lo, err1 := strconv.Atoi(part[:idx])
			hi, err2 := strconv.Atoi(part[idx+1:])
			if err1 != nil || err2 != nil || lo > hi {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			if lo < min || hi > max {
				return nil, fmt.Errorf("range %q out of bounds [%d,%d]", part, min, max)
			}
			for i := lo; i <= hi; i++ {
				set[i] = struct{}{}
			}
			continue
		}

		// 单个数字
		v, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", part)
		}
		if v < min || v > max {
			return nil, fmt.Errorf("value %d out of bounds [%d,%d]", v, min, max)
		}
		set[v] = struct{}{}
	}
	return set, nil
}

func inSet(set map[int]struct{}, v int) bool {
	_, ok := set[v]
	return ok
}

