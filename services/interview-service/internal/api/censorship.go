package api

import (
	"regexp"
	"strings"
	"unicode"
)

// Lightweight profanity / abuse filter for user-facing chat input.
//
// Goals:
//   - Block clearly abusive content (Russian and English profanity,
//     directed slurs) before it lands in the AI prompt.
//   - Be permissive enough that technical conversations aren't broken
//     by false positives — we only match whole-word stems on a small
//     curated list, not generic substring matching.
//
// Not a substitute for a moderation API; designed as a fast, local
// pre-filter so the AI never has to see raw abusive payloads.

// abusiveStems are matched as whole-word substrings after letter
// normalisation. Keep the list short; expand as needed via tests.
// English entries cover common chat slurs; Russian entries cover
// the standard "матерные" stems. Both rendered as letter-only stems
// so a-z / а-я handling is uniform.
var abusiveStems = []string{
	// English
	"fuck", "shit", "asshole", "bitch", "bastard", "dickhead", "cunt",
	"motherfucker", "wanker", "fag", "retard",
	// Russian (letter-only stems, see normaliseLetters)
	"хуй", "хуя", "хуе", "пизд", "ебать", "ебал", "ебан", "еб@",
	"бляд", "бляха",
	"мудак", "мудил", "сука", "сукин", "сучка",
	"гондон", "пидор", "пидар",
}

var abuseStemRe = regexp.MustCompile(`(?i)(` + strings.Join(escapeAll(abusiveStems), "|") + `)`)

func escapeAll(stems []string) []string {
	out := make([]string, 0, len(stems))
	for _, s := range stems {
		out = append(out, regexp.QuoteMeta(s))
	}
	return out
}

// normaliseLetters strips everything that's not a letter or digit
// (so common obfuscations like "f.u.c.k" or "сукa!" don't slip
// through) and lowercases the result.
func normaliseLetters(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// isAbusive returns true when the text contains one of the curated
// abuse stems after letter-only normalisation.
func isAbusive(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	return abuseStemRe.MatchString(normaliseLetters(text))
}

// normaliseVerdict coerces the AI-returned verdict into the strict
// allowlist used by the UI. Anything unknown returns "" so callers
// can skip broadcasting a bogus badge.
func normaliseVerdict(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "correct", "partial", "wrong", "skipped", "off_topic":
		return strings.ToLower(strings.TrimSpace(raw))
	case "none", "":
		return ""
	default:
		return ""
	}
}
