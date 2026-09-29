package edgetts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ListVoices returns all available voices and their attributes.
func ListVoices(ctx context.Context, proxy string) ([]Voice, error) {
	for attempt := 0; attempt < 2; attempt++ {
		voices, status, dateHeader, err := fetchVoices(ctx, proxy)
		if err == nil {
			return voices, nil
		}
		if status == http.StatusForbidden && attempt == 0 {
			if herr := Handle403DateHeader(dateHeader); herr != nil {
				return nil, herr
			}
			continue
		}
		return nil, err
	}
	return nil, &SkewAdjustmentError{Msg: "unreachable"}
}

func fetchVoices(ctx context.Context, proxy string) ([]Voice, int, string, error) {
	client, err := httpClientWithProxy(proxy, 10*time.Second)
	if err != nil {
		return nil, 0, "", err
	}
	u := VoiceList + "&Sec-MS-GEC=" + GenerateSecMSGEC() + "&Sec-MS-GEC-Version=" + SecMSGECVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, "", err
	}
	for k, v := range HeadersWithMUID(VoiceHeaders()) {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, resp.Header.Get("Date"), fmt.Errorf("voices request forbidden (403)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, resp.Header.Get("Date"), fmt.Errorf("voices request failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", err
	}
	var voices []Voice
	if err := json.Unmarshal(body, &voices); err != nil {
		return nil, 0, "", err
	}
	for i := range voices {
		if voices[i].VoiceTag.ContentCategories == nil {
			voices[i].VoiceTag.ContentCategories = []string{}
		}
		if voices[i].VoiceTag.VoicePersonalities == nil {
			voices[i].VoiceTag.VoicePersonalities = []string{}
		}
	}
	return voices, resp.StatusCode, "", nil
}

func httpClientWithProxy(proxy string, timeout time.Duration) (*http.Client, error) {
	transport := &http.Transport{}
	if proxy != "" {
		pu, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy %q: %w", proxy, err)
		}
		transport.Proxy = http.ProxyURL(pu)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

// VoicesManager finds voices by attribute, mirroring Python's VoicesManager.
type VoicesManager struct {
	Voices  []ManagerVoice
	created bool
}

// NewVoicesManager fetches the live voice list (or uses customVoices when given).
func NewVoicesManager(ctx context.Context, customVoices []Voice) (*VoicesManager, error) {
	var voices []Voice
	var err error
	if customVoices == nil {
		voices, err = ListVoices(ctx, "")
		if err != nil {
			return nil, err
		}
	} else {
		voices = customVoices
	}
	m := &VoicesManager{created: true}
	for _, v := range voices {
		lang := v.Locale
		if i := strings.Index(lang, "-"); i != -1 {
			lang = lang[:i]
		}
		m.Voices = append(m.Voices, ManagerVoice{Voice: v, Language: lang})
	}
	return m, nil
}

// Find returns all voices matching every entry in attrs.
// Supported keys: Gender, Locale, Language, ShortName, Name, Status.
func (m *VoicesManager) Find(attrs map[string]string) ([]ManagerVoice, error) {
	if !m.created {
		return nil, fmt.Errorf("VoicesManager.Find() called before NewVoicesManager()")
	}
	var out []ManagerVoice
	for _, v := range m.Voices {
		if matchesVoice(v, attrs) {
			out = append(out, v)
		}
	}
	return out, nil
}

func matchesVoice(v ManagerVoice, attrs map[string]string) bool {
	for k, want := range attrs {
		var got string
		switch k {
		case "Gender":
			got = v.Gender
		case "Locale":
			got = v.Locale
		case "Language":
			got = v.Language
		case "ShortName":
			got = v.ShortName
		case "Name":
			got = v.Name
		case "Status":
			got = v.Status
		default:
			return false
		}
		if got != want {
			return false
		}
	}
	return true
}
