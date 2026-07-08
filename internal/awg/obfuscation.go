package awg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/random"
)

const maxAwgHeaderValue = 2_147_483_647

type headerSpan struct {
	lo int64
	hi int64
}

type ObfuscationParams struct {
	Jc   int
	Jmin int
	Jmax int
	S1   int
	S2   int
	S3   int
	S4   int
	H1   string
	H2   string
	H3   string
	H4   string
	I1   string
	I2   string
	I3   string
	I4   string
	I5   string
}

func GenerateObfuscationParams() ObfuscationParams {
	out := ObfuscationParams{
		Jc:   3 + random.Num(4),
		Jmin: 64 + random.Num(50),
	}
	out.Jmax = out.Jmin + 50 + random.Num(100)
	if out.Jmax > 1024 {
		out.Jmax = 1024
	}
	if out.Jmax < out.Jmin {
		out.Jmax = out.Jmin + 1
	}
	for {
		s1 := 15 + random.Num(49)
		s2 := 15 + random.Num(49)
		s3 := 10 + random.Num(54)
		s4 := 1 + random.Num(15)
		if !uniqueInts(s1, s2, s3, s4) {
			continue
		}
		if s1+148 == s2+92 || s3+64 == s1+148 || s3+64 == s2+92 {
			continue
		}
		out.S1, out.S2, out.S3, out.S4 = s1, s2, s3, s4
		break
	}
	h1, h2, h3, h4 := generateHeaderRanges()
	out.H1, out.H2, out.H3, out.H4 = h1, h2, h3, h4
	i1Len := 15 + random.Num(26)
	out.I1 = fmt.Sprintf("<r %d>", i1Len)
	out.I2 = fmt.Sprintf("<r %d>", 10+random.Num(20))
	out.I3 = fmt.Sprintf("<r %d>", 10+random.Num(20))
	out.I4 = fmt.Sprintf("<r %d>", 10+random.Num(20))
	out.I5 = fmt.Sprintf("<r %d>", 10+random.Num(20))
	return out
}

func ValidateObfuscationParams(p ObfuscationParams) error {
	if p.Jc < 0 || p.Jc > 10 {
		return fmt.Errorf("Jc must be 0-10, got %d", p.Jc)
	}
	if p.Jmin < 64 || p.Jmin > 1024 {
		return fmt.Errorf("Jmin must be 64-1024, got %d", p.Jmin)
	}
	if p.Jmax < 64 || p.Jmax > 1024 {
		return fmt.Errorf("Jmax must be 64-1024, got %d", p.Jmax)
	}
	if p.Jmin > p.Jmax {
		return fmt.Errorf("Jmin must be <= Jmax")
	}
	for _, value := range []int{p.S1, p.S2, p.S3} {
		if value < 0 || value > 64 {
			return fmt.Errorf("S1-S3 must be 0-64, got %d", value)
		}
	}
	if p.S4 < 0 || p.S4 > 32 {
		return fmt.Errorf("S4 must be 0-32, got %d", p.S4)
	}
	if !uniqueInts(p.S1, p.S2, p.S3, p.S4) {
		return fmt.Errorf("S1-S4 must be unique")
	}
	if p.S1+148 == p.S2+92 || p.S3+64 == p.S1+148 || p.S3+64 == p.S2+92 {
		return fmt.Errorf("padded control packet sizes must not collide")
	}
	for _, header := range []string{p.H1, p.H2, p.H3, p.H4} {
		if err := validateHeaderRange(header); err != nil {
			return err
		}
	}
	if err := validateHeadersDisjoint(p.H1, p.H2, p.H3, p.H4); err != nil {
		return err
	}
	if strings.TrimSpace(p.I1) == "" {
		return nil
	}
	for _, chain := range []string{p.I1, p.I2, p.I3, p.I4, p.I5} {
		if strings.TrimSpace(chain) == "" {
			return fmt.Errorf("I1-I5 must all be set when AWG 2.0 CPS chain is enabled")
		}
	}
	return nil
}

func generateHeaderRanges() (h1, h2, h3, h4 string) {
	current := 150_000_000 + random.Num(50_000_000)
	ranges := make([]string, 4)
	for i := range ranges {
		start := current
		end := start + 50_000_000 + random.Num(100_000_000)
		if end > maxAwgHeaderValue {
			end = maxAwgHeaderValue
		}
		ranges[i] = fmt.Sprintf("%d-%d", start, end)
		gap := 10_000_000 + random.Num(20_000_000)
		current = end + gap
		if current > maxAwgHeaderValue-100_000_000 {
			current = 150_000_000 + random.Num(50_000_000)
		}
	}
	shuffleStrings(ranges)
	return ranges[0], ranges[1], ranges[2], ranges[3]
}

func validateHeaderRange(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("header range is empty")
	}
	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header range %q", value)
		}
		lo, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		hi, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err1 != nil || err2 != nil || lo > hi {
			return fmt.Errorf("invalid header range %q", value)
		}
		if lo < 0 || hi > maxAwgHeaderValue {
			return fmt.Errorf("header range out of bounds %q", value)
		}
		return nil
	}
	single, err := strconv.ParseInt(value, 10, 64)
	if err != nil || single < 0 || single > maxAwgHeaderValue {
		return fmt.Errorf("invalid header value %q", value)
	}
	return nil
}

func validateHeadersDisjoint(headers ...string) error {
	parsed := make([]headerSpan, 0, len(headers))
	for _, header := range headers {
		lo, hi, err := headerBounds(header)
		if err != nil {
			return err
		}
		parsed = append(parsed, headerSpan{lo: lo, hi: hi})
	}
	for i := range parsed {
		for j := i + 1; j < len(parsed); j++ {
			if rangesOverlap(parsed[i], parsed[j]) {
				return fmt.Errorf("header ranges overlap")
			}
		}
	}
	return nil
}

func headerBounds(value string) (lo int64, hi int64, err error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		lo, err = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return 0, 0, err
		}
		hi, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		return lo, hi, err
	}
	lo, err = strconv.ParseInt(value, 10, 64)
	return lo, lo, err
}

func rangesOverlap(a, b headerSpan) bool {
	return a.lo <= b.hi && b.lo <= a.hi
}

func uniqueInts(values ...int) bool {
	seen := map[int]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func shuffleStrings(values []string) {
	for i := len(values) - 1; i > 0; i-- {
		j := random.Num(i + 1)
		values[i], values[j] = values[j], values[i]
	}
}
