package watchers

import (
    "regexp"
    "strconv"
)

type RegexWatcher struct {
    regexp *regexp.Regexp
}

func NewRegexWatcher(toCompile string) (*RegexWatcher, error) {
    re, err := regexp.Compile(toCompile)
    if err != nil {
        return nil, err
    }

    return &RegexWatcher{regexp: re /*, usingRegexp2: false*/ }, nil
}

func (r *RegexWatcher) reMatchToMap(s string) (bool, map[string]string) {
    out := make(map[string]string)
    match := r.regexp.FindStringSubmatch(s)
    if match == nil {
        return false, nil
    }

    for i, name := range r.regexp.SubexpNames() {
        if i == 0 {
            continue
        }
        if name != "" {
            out[name] = match[i]
        } else {
            out[strconv.Itoa(i)] = match[i]
        }
    }

    return true, out
}

// MatchToMap matches a string against a regex, and returns whether it matched and if it did, a map of the names in the
// groups in the regexp to their contents
func (r *RegexWatcher) MatchToMap(s string) (bool, map[string]string) {
    return r.reMatchToMap(s)
}

// MatchLine matches a line against a regexp and returns whether the match happened, and an object representing the matched
// data
func (r *RegexWatcher) MatchLine(s string) (bool, MatchedLine) {
    if isMatched, mapped := r.MatchToMap(s); isMatched {
        return isMatched, MatchedLine{Groups: mapped}
    }

    return false, MatchedLine{}
}
