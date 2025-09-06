package statsmap

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

// StatMatcher represents a single matcher entry from Exiled-Exchange-2 stats.ndjson
type StatMatcher struct {
    String string `json:"string"`
    Negate bool   `json:"negate,omitempty"`
}

// statNDJSONLine represents only the parts we care about from stats.ndjson
type statNDJSONLine struct {
    Matchers []StatMatcher `json:"matchers"`
    Trade    struct {
        IDs map[string][]string `json:"ids"`
    } `json:"trade"`
}

// matcherToID holds a normalized matcher string to a preferred trade stat id
var matcherToID map[string]string
var loadOnce sync.Once

// preferred ID order when multiple types are present
var idPreference = []string{
    "explicit",
    "implicit",
    "crafted",
    "enchant",
    "fractured",
    "rune",
    "scourge",
    // pseudo are supported by trade API, but usually not desired as exact stat filters
    "pseudo",
}

// Normalize a matcher key for stable lookups
func normalizeMatcherKey(s string) string {
    s = strings.TrimSpace(s)
    // unify whitespace
    s = strings.Join(strings.Fields(s), " ")
    return s
}

// choosePreferredID picks a single id from the ids map using the preference order
func choosePreferredID(ids map[string][]string) (string, bool) {
    for _, k := range idPreference {
        if arr, ok := ids[k]; ok && len(arr) > 0 {
            return arr[0], true
        }
    }
    // fallback: any
    for _, arr := range ids {
        if len(arr) > 0 {
            return arr[0], true
        }
    }
    return "", false
}

// Load attempts to load stats.ndjson from Exiled-Exchange-2 repo or env override once.
// It is safe to call multiple times; the file is parsed at most once.
func Load() {
    loadOnce.Do(func() {
        matcherToID = make(map[string]string)

        // Look for override via env
        // EXILED_EXCHANGE_STATS_PATH can point directly to stats.ndjson
        // EXILED_EXCHANGE_DATA_DIR can point to a folder that contains stats.ndjson
        candidates := []string{}
        if p := os.Getenv("EXILED_EXCHANGE_STATS_PATH"); p != "" {
            candidates = append(candidates, p)
        }
        if dir := os.Getenv("EXILED_EXCHANGE_DATA_DIR"); dir != "" {
            candidates = append(candidates, filepath.Join(dir, "stats.ndjson"))
        }

        // Default known path from the user's repo layout
        if home, err := os.UserHomeDir(); err == nil {
            candidates = append(candidates,
                filepath.Join(home, "git", "other", "Exiled-Exchange-2", "renderer", "public", "data", "en", "stats.ndjson"),
            )
        }

        var f *os.File
        for _, c := range candidates {
            file, err := os.Open(c)
            if err == nil {
                f = file
                break
            }
        }

        if f == nil {
            // No external data found; leave matcherToID empty and rely on built-ins
            return
        }
        defer f.Close()

        scanner := bufio.NewScanner(f)
        // Increase the scanner buffer for large lines
        const maxCapacity = 1024 * 1024
        buf := make([]byte, 0, 64*1024)
        scanner.Buffer(buf, maxCapacity)

        for scanner.Scan() {
            line := scanner.Bytes()
            var node statNDJSONLine
            if err := json.Unmarshal(line, &node); err != nil {
                continue
            }
            if len(node.Matchers) == 0 || node.Trade.IDs == nil {
                continue
            }
            // default choice
            id, ok := choosePreferredID(node.Trade.IDs)
            if !ok {
                continue
            }
            for _, m := range node.Matchers {
                if m.String == "" {
                    continue
                }
                key := normalizeMatcherKey(m.String)
                // Special-case: "# to maximum Energy Shield" should prefer the more local explicit id when available
                if key == "# to maximum Energy Shield" {
                    if arr, exists := node.Trade.IDs["explicit"]; exists && len(arr) > 1 {
                        id = arr[1]
                    }
                }
                if _, exists := matcherToID[key]; !exists {
                    matcherToID[key] = id
                }
            }
        }
        // ignore scanner errors to keep non-fatal
    })
}

// FindID tries to resolve a normalized matcher string to a trade stat id.
// Call Load() once before using this to initialize the dataset if available.
func FindID(normalizedMatcher string) (string, bool) {
    if matcherToID == nil {
        return "", false
    }
    id, ok := matcherToID[normalizedMatcher]
    return id, ok
}
