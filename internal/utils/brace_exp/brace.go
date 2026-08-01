package braceexp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const defaultLimit = 10_000

func Expand(pattern string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	count, err := expansionCount(pattern, limit)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, count)
	buf := make([]byte, 0, len(pattern)+16)

	if err := expand(&out, buf, pattern); err != nil {
		return nil, err
	}

	return out, nil
}

func expand(out *[]string, buf []byte, pattern string) error {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		buf = append(buf, pattern...)
		*out = append(*out, string(buf))
		return nil
	}

	close := strings.IndexByte(pattern[open+1:], '}')
	if close < 0 {
		return fmt.Errorf("unclosed brace at byte %d", open)
	}
	close += open + 1

	body := pattern[open+1 : close]
	if strings.IndexByte(body, '{') >= 0 {
		return errors.New("nested braces are not supported")
	}

	buf = append(buf, pattern[:open]...)
	tail := pattern[close+1:]

	start, end, step, width, padded, isRange, err := parseRange(body)
	if err != nil {
		return err
	}

	if isRange {
		for value := start; ; {
			size := len(buf)
			buf = appendNumber(buf, value, width, padded)

			if err := expand(out, buf, tail); err != nil {
				return err
			}

			buf = buf[:size]

			if value == end {
				return nil
			}

			if start < end {
				if uint64(end)-uint64(value) < step {
					return nil
				}
				value += int64(step)
			} else {
				if uint64(value)-uint64(end) < step {
					return nil
				}
				value -= int64(step)
			}
		}
	}

	if !strings.ContainsRune(body, ',') {
		return fmt.Errorf("invalid brace expression {%s}", body)
	}

	for {
		part, rest, found := strings.Cut(body, ",")
		size := len(buf)
		buf = append(buf, part...)

		if err := expand(out, buf, tail); err != nil {
			return err
		}

		buf = buf[:size]

		if !found {
			return nil
		}

		body = rest
	}
}

func expansionCount(pattern string, limit int) (int, error) {
	count := 1

	for offset := 0; ; {
		relativeOpen := strings.IndexByte(pattern[offset:], '{')
		if relativeOpen < 0 {
			return count, nil
		}

		open := offset + relativeOpen
		relativeClose := strings.IndexByte(pattern[open+1:], '}')
		if relativeClose < 0 {
			return 0, fmt.Errorf("unclosed brace at byte %d", open)
		}

		close := open + relativeClose + 1
		body := pattern[open+1 : close]

		if strings.IndexByte(body, '{') >= 0 {
			return 0, errors.New("nested braces are not supported")
		}

		variants, err := variantCount(body)
		if err != nil {
			return 0, err
		}

		if variants > limit/count {
			return 0, fmt.Errorf("expansion exceeds limit %d", limit)
		}

		count *= variants
		offset = close + 1
	}
}

func variantCount(body string) (int, error) {
	start, end, step, _, _, isRange, err := parseRange(body)
	if err != nil {
		return 0, err
	}

	if !isRange {
		if !strings.ContainsRune(body, ',') {
			return 0, fmt.Errorf("invalid brace expression {%s}", body)
		}

		return strings.Count(body, ",") + 1, nil
	}

	var distance uint64
	if start <= end {
		distance = uint64(end) - uint64(start)
	} else {
		distance = uint64(start) - uint64(end)
	}

	count := distance/step + 1
	if count > uint64(^uint(0)>>1) {
		return 0, errors.New("expansion is too large")
	}

	return int(count), nil
}

func parseRange(body string) (
	start int64,
	end int64,
	step uint64,
	width int,
	padded bool,
	ok bool,
	err error,
) {
	first := strings.Index(body, "..")
	if first < 0 {
		return
	}

	startText := body[:first]
	endText := body[first+2:]
	stepText := ""

	if relativeSecond := strings.Index(endText, ".."); relativeSecond >= 0 {
		stepText = endText[relativeSecond+2:]
		endText = endText[:relativeSecond]

		if strings.Contains(stepText, "..") {
			err = fmt.Errorf("invalid range {%s}", body)
			return
		}
	}

	start, err = strconv.ParseInt(startText, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid range {%s}", body)
		return
	}

	end, err = strconv.ParseInt(endText, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid range {%s}", body)
		return
	}

	step = 1
	if stepText != "" {
		step, err = strconv.ParseUint(trimSign(stepText), 10, 63)
		if err != nil || step == 0 {
			err = fmt.Errorf("invalid range {%s}", body)
			return
		}
	}

	startDigits := trimSign(startText)
	endDigits := trimSign(endText)

	width = max(len(startText), len(endText))
	padded = hasLeadingZero(startDigits) || hasLeadingZero(endDigits)
	ok = true

	return
}

func appendNumber(
	dst []byte,
	value int64,
	width int,
	padded bool,
) []byte {
	var raw [20]byte
	text := strconv.AppendInt(raw[:0], value, 10)

	if !padded {
		return append(dst, text...)
	}

	if text[0] == '-' {
		dst = append(dst, '-')
		text = text[1:]
		width--
	}

	for i := len(text); i < width; i++ {
		dst = append(dst, '0')
	}

	return append(dst, text...)
}

func trimSign(value string) string {
	if len(value) > 0 && (value[0] == '-' || value[0] == '+') {
		return value[1:]
	}

	return value
}

func hasLeadingZero(value string) bool {
	return len(value) > 1 && value[0] == '0'
}
