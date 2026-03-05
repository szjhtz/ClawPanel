package config

import (
	"strconv"
	"strings"
)

// NormalizeOpenClawConfig 对 openclaw.json 做兼容性清洗。
// 返回 true 表示内容被修改。
//
// 兼容点：
//  1. 旧版面板曾写入 agents.defaults.model.contextTokens / maxTokens，
//     但新版 OpenClaw 仅允许 model 为 string 或 {primary,fallbacks}。
//  2. 将 legacy contextTokens 迁移到 agents.defaults.contextTokens。
func NormalizeOpenClawConfig(cfg map[string]interface{}) bool {
	if cfg == nil {
		return false
	}
	changed := false

	agents, ok := cfg["agents"].(map[string]interface{})
	if ok && agents != nil {
		if defaults, ok := agents["defaults"].(map[string]interface{}); ok && defaults != nil {
			modelRaw, exists := defaults["model"]
			if exists {
				switch m := modelRaw.(type) {
				case string:
					if strings.TrimSpace(m) == "" {
						delete(defaults, "model")
						changed = true
					}
				case map[string]interface{}:
					if defaults["contextTokens"] == nil {
						if v, ok := toPositiveInt(m["contextTokens"]); ok {
							defaults["contextTokens"] = v
							changed = true
						}
					}

					clean := map[string]interface{}{}
					if p, ok := m["primary"].(string); ok {
						p = strings.TrimSpace(p)
						if p != "" {
							clean["primary"] = p
						}
					}
					if fbAny, ok := m["fallbacks"].([]interface{}); ok {
						fallbacks := make([]interface{}, 0, len(fbAny))
						for _, item := range fbAny {
							s, ok := item.(string)
							if !ok {
								changed = true
								continue
							}
							s = strings.TrimSpace(s)
							if s == "" {
								changed = true
								continue
							}
							fallbacks = append(fallbacks, s)
						}
						if len(fallbacks) > 0 {
							clean["fallbacks"] = fallbacks
						}
					}

					if len(m) != len(clean) {
						changed = true
					}
					for k := range m {
						if k != "primary" && k != "fallbacks" {
							changed = true
							break
						}
					}

					if len(clean) == 0 {
						delete(defaults, "model")
						changed = true
					} else {
						defaults["model"] = clean
					}
				default:
					delete(defaults, "model")
					changed = true
				}
			}

			if compaction, ok := defaults["compaction"].(map[string]interface{}); ok && compaction != nil {
				if mode, ok := compaction["mode"].(string); ok {
					switch strings.TrimSpace(mode) {
					case "", "default", "safeguard":
						// no-op
					case "aggressive":
						compaction["mode"] = "safeguard"
						changed = true
					case "off":
						compaction["mode"] = "default"
						changed = true
					default:
						delete(compaction, "mode")
						changed = true
					}
				}
			}
		}
	}

	if gateway, ok := cfg["gateway"].(map[string]interface{}); ok && gateway != nil {
		if mode, ok := gateway["mode"].(string); ok && strings.TrimSpace(mode) == "hosted" {
			gateway["mode"] = "remote"
			changed = true
		}
		if custom, ok := gateway["customBindHost"].(string); !ok || strings.TrimSpace(custom) == "" {
			if bindAddr, ok := gateway["bindAddress"].(string); ok && strings.TrimSpace(bindAddr) != "" {
				gateway["customBindHost"] = strings.TrimSpace(bindAddr)
				changed = true
			}
		}
		if _, ok := gateway["bindAddress"]; ok {
			delete(gateway, "bindAddress")
			changed = true
		}
	}

	if hooks, ok := cfg["hooks"].(map[string]interface{}); ok && hooks != nil {
		if p, ok := hooks["path"].(string); !ok || strings.TrimSpace(p) == "" {
			if legacyPath, ok := hooks["basePath"].(string); ok && strings.TrimSpace(legacyPath) != "" {
				hooks["path"] = strings.TrimSpace(legacyPath)
				changed = true
			}
		}
		if tok, ok := hooks["token"].(string); !ok || strings.TrimSpace(tok) == "" {
			if legacyToken, ok := hooks["secret"].(string); ok && strings.TrimSpace(legacyToken) != "" {
				hooks["token"] = legacyToken
				changed = true
			}
		}
		if _, ok := hooks["basePath"]; ok {
			delete(hooks, "basePath")
			changed = true
		}
		if _, ok := hooks["secret"]; ok {
			delete(hooks, "secret")
			changed = true
		}
	}

	if messages, ok := cfg["messages"].(map[string]interface{}); ok && messages != nil {
		if _, ok := messages["systemPrompt"]; ok {
			delete(messages, "systemPrompt")
			changed = true
		}
		if _, ok := messages["maxHistoryMessages"]; ok {
			delete(messages, "maxHistoryMessages")
			changed = true
		}
	}

	return changed
}

func toPositiveInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int:
		if x > 0 {
			return x, true
		}
	case int32:
		if x > 0 {
			return int(x), true
		}
	case int64:
		if x > 0 {
			return int(x), true
		}
	case float64:
		if x > 0 {
			return int(x), true
		}
	case string:
		x = strings.TrimSpace(x)
		if x == "" {
			return 0, false
		}
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0, false
		}
		if n > 0 {
			return n, true
		}
	}
	return 0, false
}
