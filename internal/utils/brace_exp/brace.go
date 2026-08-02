// Bash brace expansion
//
// * NOTICE
// * This file contains code initially generated with the assistance of AI tooling (this file only not the project as a whole).
// * The resulting implementation has been reviewed, tested, and, where necessary,
// * corrected by the project maintainer prior to publication.

package braceexp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const defaultLimit = 10_000

type rangeSpec struct {
	start  int64
	end    int64
	step   uint64
	width  int
	padded bool
}

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

	if err := appendExpanded(&out, buf, pattern); err != nil {
		return nil, err
	}

	return out, nil
}

func appendExpanded(out *[]string, buf []byte, pattern string) error {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		buf = append(buf, pattern...)
		*out = append(*out, string(buf))
		return nil
	}

	braceEnd := strings.IndexByte(pattern[open+1:], '}')
	if braceEnd < 0 {
		return fmt.Errorf("unclosed brace at byte %d", open)
	}
	braceEnd += open + 1

	body := pattern[open+1 : braceEnd]
	if strings.IndexByte(body, '{') >= 0 {
		return errors.New("nested braces are not supported")
	}

	buf = append(buf, pattern[:open]...)
	tail := pattern[braceEnd+1:]

	if strings.IndexByte(body, ',') >= 0 {
		for {
			part, rest, found := strings.Cut(body, ",")
			size := len(buf)
			buf = append(buf, part...)

			if err := appendExpanded(out, buf, tail); err != nil {
				return err
			}

			buf = buf[:size]
			if !found {
				return nil
			}
			body = rest
		}
	}

	spec, isRange, err := parseRange(body)
	if err != nil {
		return err
	}
	if !isRange {
		return fmt.Errorf("invalid brace expression {%s}", body)
	}

	return appendRange(out, buf, tail, spec)
}

func appendRange(
	out *[]string,
	buf []byte,
	tail string,
	spec rangeSpec,
) error {
	for value := spec.start; ; {
		size := len(buf)
		buf = appendNumber(
			buf,
			value,
			spec.width,
			spec.padded,
		)

		if err := appendExpanded(out, buf, tail); err != nil {
			return err
		}

		buf = buf[:size]
		if value == spec.end {
			return nil
		}

		if spec.start < spec.end {
			if uint64(spec.end)-uint64(value) < spec.step {
				return nil
			}
			value += int64(spec.step)
			continue
		}

		if uint64(value)-uint64(spec.end) < spec.step {
			return nil
		}
		value -= int64(spec.step)
	}
}

func expansionCount(pattern string, limit int) (int, error) {
	count := 1

	for offset := 0; ; {
		relativeOpen := strings.IndexByte(
			pattern[offset:],
			'{',
		)
		if relativeOpen < 0 {
			return count, nil
		}

		open := offset + relativeOpen
		relativeEnd := strings.IndexByte(
			pattern[open+1:],
			'}',
		)
		if relativeEnd < 0 {
			return 0, fmt.Errorf(
				"unclosed brace at byte %d",
				open,
			)
		}

		braceEnd := open + relativeEnd + 1
		body := pattern[open+1 : braceEnd]

		if strings.IndexByte(body, '{') >= 0 {
			return 0, errors.New(
				"nested braces are not supported",
			)
		}

		variants, err := variantCount(body)
		if err != nil {
			return 0, err
		}

		if variants > limit/count {
			return 0, fmt.Errorf(
				"expansion exceeds limit %d",
				limit,
			)
		}

		count *= variants
		offset = braceEnd + 1
	}
}

func variantCount(body string) (int, error) {
	if strings.IndexByte(body, ',') >= 0 {
		return strings.Count(body, ",") + 1, nil
	}

	spec, isRange, err := parseRange(body)
	if err != nil {
		return 0, err
	}
	if !isRange {
		return 0, fmt.Errorf(
			"invalid brace expression {%s}",
			body,
		)
	}

	var distance uint64
	if spec.start <= spec.end {
		distance = uint64(spec.end) - uint64(spec.start)
	} else {
		distance = uint64(spec.start) - uint64(spec.end)
	}

	count := distance/spec.step + 1
	if count > uint64(^uint(0)>>1) {
		return 0, errors.New("expansion is too large")
	}

	return int(count), nil
}

func parseRange(body string) (rangeSpec, bool, error) {
	first := strings.Index(body, "..")
	if first < 0 {
		return rangeSpec{}, false, nil
	}

	startText := body[:first]
	endText := body[first+2:]
	stepText := ""

	if second := strings.Index(endText, ".."); second >= 0 {
		stepText = endText[second+2:]
		endText = endText[:second]

		if stepText == "" ||
			strings.Contains(stepText, "..") {
			return rangeSpec{}, false, fmt.Errorf(
				"invalid range {%s}",
				body,
			)
		}
	}

	start, err := strconv.ParseInt(startText, 10, 64)
	if err != nil {
		return rangeSpec{}, false, fmt.Errorf(
			"invalid range {%s}",
			body,
		)
	}

	end, err := strconv.ParseInt(endText, 10, 64)
	if err != nil {
		return rangeSpec{}, false, fmt.Errorf(
			"invalid range {%s}",
			body,
		)
	}

	step := uint64(1)
	if stepText != "" {
		step, err = strconv.ParseUint(
			trimSign(stepText),
			10,
			63,
		)
		if err != nil || step == 0 {
			return rangeSpec{}, false, fmt.Errorf(
				"invalid range {%s}",
				body,
			)
		}
	}

	return rangeSpec{
		start: start,
		end:   end,
		step:  step,
		width: max(len(startText), len(endText)),
		padded: hasLeadingZero(startText) ||
			hasLeadingZero(endText),
	}, true, nil
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
	if value != "" &&
		(value[0] == '-' || value[0] == '+') {
		return value[1:]
	}

	return value
}

func hasLeadingZero(value string) bool {
	return len(value) > 1 && value[0] == '0' ||
		len(value) > 2 &&
			value[0] == '-' &&
			value[1] == '0'
}
